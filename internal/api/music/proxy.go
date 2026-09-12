// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package music

import (
	"context"
	"melovian/internal/api/apishared"
	"melovian/internal/appconfig"
	"melovian/internal/cache"
	"melovian/internal/httputil"
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
	cache          *cache.ResponseCache
}

func NewProxyHandler(cfg appconfig.Config, instances *store.InstanceStore, localLibraries *store.LocalLibraryStore, localTracks *store.LocalTrackStore, preferences *store.PreferencesStore, resolver *apishared.Resolver, cache *cache.ResponseCache) *ProxyHandler {
	return &ProxyHandler{
		cfg:            cfg,
		instances:      instances,
		localLibraries: localLibraries,
		localTracks:    localTracks,
		preferences:    preferences,
		resolver:       resolver,
		cache:          cache,
	}
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

	proxy := subsonic.NewProxy(h.subsonicForContext, h.cache, h.cfg.CacheEnabled, apishared.ResolveProgressUserID)
	mux.Handle("/api/subsonic/", proxy)
	mux.Handle("/api/subsonic", proxy)
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
