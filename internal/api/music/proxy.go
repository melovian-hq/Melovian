// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package music

import (
	"context"
	"melovian/internal/api/apishared"
	"melovian/internal/appconfig"
	"melovian/internal/cache"
	"melovian/internal/httputil"
	"melovian/internal/sources"
	"melovian/internal/store"
	"melovian/internal/subsonic"
	"net/http"
)

// ProxyHandler serves the remote music status routes and the subsonic API
// proxy under /api/subsonic.
type ProxyHandler struct {
	cfg            appconfig.Config
	instances      *store.InstanceStore
	localLibraries *store.LocalLibraryStore
	localTracks    *store.LocalTrackStore
	preferences    *store.PreferencesStore
	resolver       *apishared.Resolver
	sources        *sources.Registry
	cache          *cache.ResponseCache
	fallbackProxy  http.Handler
	deps           sources.Deps
}

func NewProxyHandler(cfg appconfig.Config, instances *store.InstanceStore, localLibraries *store.LocalLibraryStore, localTracks *store.LocalTrackStore, preferences *store.PreferencesStore, resolver *apishared.Resolver, reg *sources.Registry, cache *cache.ResponseCache) *ProxyHandler {
	h := &ProxyHandler{
		cfg:            cfg,
		instances:      instances,
		localLibraries: localLibraries,
		localTracks:    localTracks,
		preferences:    preferences,
		resolver:       resolver,
		sources:        reg,
		cache:          cache,
	}
	h.fallbackProxy = subsonic.NewProxy(h.subsonicForContext, cache, cfg.CacheEnabled, apishared.ResolveProgressUserID)
	h.deps = sources.Deps{
		Cache:        cache,
		CacheEnabled: cfg.CacheEnabled,
		CacheScope:   apishared.ResolveProgressUserID,
		ClientFor: func(inst store.SourceInstance) *subsonic.Client {
			return h.resolver.CachedClient(inst.ID, inst.ServerURL, inst.Username, inst.Password)
		},
	}
	return h
}

func (h *ProxyHandler) subsonicForContext(ctx context.Context) *subsonic.Client {
	return h.resolver.ForContext(ctx)
}

func (h *ProxyHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/music/status", func(w http.ResponseWriter, r *http.Request) {
		userID := apishared.UserIDFromContext(r.Context())
		mode := apishared.SourceViewModeForUser(h.preferences, userID)
		instanceID, _ := h.instances.GetActiveIDForUser(userID)
		localLib, localErr := h.localLibraries.GetActiveForUser(userID)

		if store.IsUnifiedSourceView(mode) && instanceID != "" && localErr == nil {
			httputil.WriteJSON(w, http.StatusOK, map[string]any{
				"enabled":    true,
				"connected":  true,
				"serverName": "All sources",
				"version":    "unified",
				"source":     "unified",
			})
			return
		}

		if store.IsLocalSourceView(mode) && localErr == nil {
			httputil.WriteJSON(w, http.StatusOK, map[string]any{
				"enabled":    true,
				"connected":  true,
				"serverName": localLib.Name,
				"version":    "local",
				"source":     "local",
			})
			return
		}

		if instanceID != "" || store.IsSubsonicSourceView(mode) {
			if h.sourceUnavailable(w, r) {
				return
			}
			client := h.resolver.ForContext(r.Context())
			subsonic.StatusHandler(client, h.cache, h.cfg.CacheEnabled, apishared.ResolveProgressUserID)(w, r)
			return
		}

		if localErr == nil {
			httputil.WriteJSON(w, http.StatusOK, map[string]any{
				"enabled":    true,
				"connected":  true,
				"serverName": localLib.Name,
				"version":    "local",
				"source":     "local",
			})
			return
		}
		if h.sourceUnavailable(w, r) {
			return
		}
		client := h.resolver.ForContext(r.Context())
		subsonic.StatusHandler(client, h.cache, h.cfg.CacheEnabled, apishared.ResolveProgressUserID)(w, r)
	})
	mux.HandleFunc("GET /api/music/library-stats", func(w http.ResponseWriter, r *http.Request) {
		userID := apishared.UserIDFromContext(r.Context())
		mode := apishared.SourceViewModeForUser(h.preferences, userID)
		instanceID, _ := h.instances.GetActiveIDForUser(userID)
		localLib, localErr := h.localLibraries.GetActiveForUser(userID)

		if store.IsUnifiedSourceView(mode) && instanceID != "" && localErr == nil {
			client := h.resolver.ForContext(r.Context())
			remote, remoteErr := client.LibraryStats()
			if remoteErr != nil {
				httputil.WriteError(w, http.StatusBadGateway, "bad_gateway", "failed to load library stats")
				return
			}
			artistCount, albumCount, countErr := h.localTracks.CountDistinctArtistsAlbums(localLib.ID)
			if countErr != nil {
				httputil.WriteInternalError(w, r, "count library stats", countErr)
				return
			}
			httputil.WriteJSON(w, http.StatusOK, map[string]any{
				"songCount":   remote.SongCount + localLib.TrackCount,
				"albumCount":  remote.AlbumCount + albumCount,
				"artistCount": remote.ArtistCount + artistCount,
				"folderCount": remote.FolderCount + 1,
				"scanning":    remote.Scanning || localLib.ScanStatus == "scanning",
				"lastScan":    remote.LastScan,
			})
			return
		}

		if store.IsLocalSourceView(mode) && localErr == nil {
			artistCount, albumCount, countErr := h.localTracks.CountDistinctArtistsAlbums(localLib.ID)
			if countErr != nil {
				httputil.WriteInternalError(w, r, "count library stats", countErr)
				return
			}
			httputil.WriteJSON(w, http.StatusOK, map[string]any{
				"songCount":   localLib.TrackCount,
				"albumCount":  albumCount,
				"artistCount": artistCount,
				"folderCount": 1,
				"scanning":    localLib.ScanStatus == "scanning",
			})
			return
		}

		if instanceID != "" || store.IsSubsonicSourceView(mode) {
			if h.sourceUnavailable(w, r) {
				return
			}
			client := h.resolver.ForContext(r.Context())
			subsonic.LibraryStatsHandler(client, h.cache, h.cfg.CacheEnabled, apishared.ResolveProgressUserID)(w, r)
			return
		}

		if localErr == nil {
			artistCount, albumCount, countErr := h.localTracks.CountDistinctArtistsAlbums(localLib.ID)
			if countErr != nil {
				httputil.WriteInternalError(w, r, "count library stats", countErr)
				return
			}
			httputil.WriteJSON(w, http.StatusOK, map[string]any{
				"songCount":   localLib.TrackCount,
				"albumCount":  albumCount,
				"artistCount": artistCount,
				"folderCount": 1,
				"scanning":    localLib.ScanStatus == "scanning",
			})
			return
		}
		httputil.WriteError(w, http.StatusServiceUnavailable, "service_unavailable", "no music source configured")
	})
	mux.HandleFunc("POST /api/music/library/refresh", h.handleRefreshLibraryCache)

	if h.sources != nil {
		mux.Handle("/api/subsonic/", http.HandlerFunc(h.serveSourceREST))
		mux.Handle("/api/subsonic", http.HandlerFunc(h.serveSourceREST))
	} else {
		mux.Handle("/api/subsonic/", h.fallbackProxy)
		mux.Handle("/api/subsonic", h.fallbackProxy)
	}
}

// sourceUnavailable writes a disconnected status when the resolved
// instance's source extension is disabled, unknown, or breaker-open.
// It returns true when it wrote a response.
func (h *ProxyHandler) sourceUnavailable(w http.ResponseWriter, r *http.Request) bool {
	if h.sources == nil {
		return false
	}
	inst, ok := h.instanceForRequest(r)
	if !ok {
		return false
	}
	status := subsonic.StatusResponse{Enabled: true, Connected: false}
	if _, srcErr := h.sources.Available(inst); srcErr != nil {
		status.Error = srcErr.Error()
		httputil.WriteJSON(w, http.StatusOK, status)
		return true
	}
	if err := h.sources.AllowRequest(inst.ID); err != nil {
		status.ServerName = inst.ServerName
		status.Error = err.Error()
		httputil.WriteJSON(w, http.StatusOK, status)
		return true
	}
	return false
}

// serveSourceREST dispatches a proxied request to the source extension
// backing the resolved instance. Requests with no resolvable instance
// fall back to the legacy active-client proxy.
func (h *ProxyHandler) serveSourceREST(w http.ResponseWriter, r *http.Request) {
	inst, ok := h.instanceForRequest(r)
	if !ok {
		h.fallbackProxy.ServeHTTP(w, r)
		return
	}
	h.sources.ServeREST(w, r, inst, h.deps)
}

// instanceForRequest returns the instance row for this request, reusing the
// row the middleware already resolved when available so a proxied call does
// not re-read and re-decrypt the same record.
func (h *ProxyHandler) instanceForRequest(r *http.Request) (store.SourceInstance, bool) {
	if inst, ok := apishared.ResolvedInstanceFromContext(r.Context()); ok {
		return inst, inst.ID != ""
	}
	userID := apishared.UserIDFromContext(r.Context())
	instanceID, err := h.resolver.ResolveInstanceID(r)
	if err != nil || instanceID == "" {
		return store.SourceInstance{}, false
	}
	inst, err := h.instances.GetForUser(userID, instanceID)
	if err != nil {
		return store.SourceInstance{}, false
	}
	return inst, true
}

func (h *ProxyHandler) handleRefreshLibraryCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	scope := apishared.ResolveProgressUserID(r.Context())
	prefix := scope + "|"
	removed := h.cache.InvalidatePrefix(prefix)
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"invalidated": removed})
}
