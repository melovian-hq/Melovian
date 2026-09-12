// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"melovian/internal/appconfig"
	"melovian/internal/cache"
	"melovian/internal/compat"
	"melovian/internal/democatalog"
	"melovian/internal/dlna"
	"melovian/internal/extensions"
	"melovian/internal/httputil"
	"melovian/internal/jukebox"
	"melovian/internal/localmusic"
	"melovian/internal/metaloader"
	"melovian/internal/observability"
	"melovian/internal/store"
	"melovian/internal/subsonic"
	"melovian/internal/subsonicserver"
	"melovian/internal/transcode"
	"melovian/internal/video"
)

type Server struct {
	cfg                appconfig.Config
	instances          *store.InstanceStore
	localLibraries     *store.LocalLibraryStore
	localTracks        *store.LocalTrackStore
	auth               *store.AuthStore
	listen             *store.ListenStore
	preferences        *store.PreferencesStore
	downloads          *store.DownloadStore
	videoLinks         *store.TrackVideoLinkStore
	videoClient        *video.Client
	videoSettingsCache sync.Map
	subsonic           *subsonic.Client
	clientCache        map[string]*subsonic.Client
	mux                *http.ServeMux
	server             *http.Server
	listenAddr         string
	cache              *cache.ResponseCache
	downloadSem        chan struct{}
	downloadLocks      sync.Map
	mu                 sync.RWMutex
	catalogCache       *localmusic.CatalogCache
	coverCache         *localmusic.CoverCache
	libraryScanner     *metaloader.Scanner
	libraryWatcher     *metaloader.LibraryWatcher
	events             *EventHub
	devices            *DeviceRegistry
	subsonicServer     *subsonicserver.Server
	shares             *store.ShareStore
	notifications      *store.NotificationStore
	jukebox            *jukebox.Controller
	dlna               *dlna.Server
	oidcOnce           sync.Once
	oidcInitErr        error
	oidcRuntimeValue   *oidcRuntime
	sentryCfg          appconfig.SentryConfig
	db                 *store.DB
	upd                updateState
	setupMu            sync.Mutex
	authLimiter        *rateLimiter
	shareLimiter       *rateLimiter
	clientLogLimiter   *rateLimiter
}

func NewServer(cfg appconfig.Config, db *store.DB) *Server {
	ConfigureCORSOrigins(cfg.CORSOrigins)
	instances := store.NewInstanceStore(db)
	if cipher, err := store.LoadSecretCipher(cfg.DataDir); err != nil {
		slog.Error("instance credential encryption unavailable", "err", err)
	} else if cipher != nil {
		instances.SetCipher(cipher)
		if err := instances.MigratePasswords(); err != nil {
			slog.Error("instance password migration failed", "err", err)
		}
	}
	localLibraries := store.NewLocalLibraryStore(db)
	localTracks := store.NewLocalTrackStore(db)
	listen := store.NewListenStore(db)
	preferences := store.NewPreferencesStore(db)
	downloads := store.NewDownloadStore(db)
	responseCache := cache.NewResponseCache()
	events := NewEventHub()

	s := &Server{
		cfg:              cfg,
		db:               db,
		instances:        instances,
		localLibraries:   localLibraries,
		localTracks:      localTracks,
		auth:             store.NewAuthStore(db, cfg.AuthSecret),
		listen:           listen,
		preferences:      preferences,
		downloads:        downloads,
		videoLinks:       store.NewTrackVideoLinkStore(db),
		videoClient:      video.NewClient(),
		mux:              http.NewServeMux(),
		cache:            responseCache,
		clientCache:      make(map[string]*subsonic.Client),
		downloadSem:      make(chan struct{}, 3),
		catalogCache:     localmusic.NewCatalogCache(),
		coverCache:       localmusic.NewCoverCache(),
		libraryScanner:   metaloader.NewScanner(localLibraries, localTracks),
		events:           events,
		devices:          NewDeviceRegistry(events),
		shares:           store.NewShareStore(db),
		notifications:    store.NewNotificationStore(db),
		jukebox:          jukebox.NewController(),
		authLimiter:      newRateLimiter(10, 5*time.Minute),
		shareLimiter:     newRateLimiter(10, 5*time.Minute),
		clientLogLimiter: newRateLimiter(60, time.Minute),
	}
	s.devices.lookupUsername = func(userID string) string {
		if userID == "" || s.auth == nil {
			return ""
		}
		u, err := s.auth.GetUser(userID)
		if err != nil {
			return ""
		}
		return u.Username
	}
	s.dlna = dlna.New(
		cfg.DLNAServerEffective(),
		cfg.DLNAHost,
		cfg.DLNAPort,
		s.publicBaseURL(),
		s.dlnaCatalogAdapter(),
	)
	s.subsonicServer = s.newSubsonicServer()
	s.libraryScanner.SetProgressHook(s.emitScanProgress)
	if watcher, err := metaloader.NewLibraryWatcher(s.libraryScanner, s.handleLibraryWatchUpdate); err != nil {
		slog.Warn("local library file watching unavailable", "err", err)
	} else {
		s.libraryWatcher = watcher
		if s.localLibraryEnabled() {
			if libs, listErr := localLibraries.List(); listErr == nil {
				for _, lib := range libs {
					s.watchLocalLibrary(lib)
				}
			}
		}
	}
	_ = s.reloadActiveSubsonic()
	if err := extensions.InstallBundled(cfg.DataDir); err != nil {
		slog.Error("failed to install bundled extensions", "err", err)
	}
	if err := s.initSentryFromStore(); err != nil {
		slog.Error("sentry init failed", "err", err)
	}

	musicSvc := NewMusicService(listen, preferences, cfg.DataDir, httputil.NewRetryHTTPClient())
	musicSvc.Register(s.mux)

	s.mux.HandleFunc("GET /api/config", s.handleGetConfig)
	s.registerAuthRoutes()
	s.registerOIDCRoutes()
	s.registerMetricsRoutes()
	s.registerDebugRoutes()
	s.registerClientLogRoutes()
	s.registerSentrySettingsRoutes()
	s.registerInstanceRoutes()
	s.registerSourceRoutes()
	s.registerWSRoutes()
	s.registerLocalLibraryRoutes()
	s.registerFilesystemRoutes()
	s.registerLocalMusicRoutes()
	s.registerLocalMetadataRoutes()
	s.registerVideoRoutes()
	s.registerExtensionRoutes()
	s.registerSubsonicRoutes()
	s.registerDownloadRoutes()
	s.registerMediaDownloadRoutes()
	s.registerLyricsRoutes()
	s.registerSmartPlaylistRoutes()
	s.registerShareRoutes()
	s.registerNotificationRoutes()
	s.registerPartyRoutes()
	s.registerJukeboxRoutes()
	s.registerUpdateRoutes()

	handler := s.buildAPIHandler()
	s.server = &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 15 * time.Second,
	}

	return s
}

func (s *Server) ResolveInstanceID(r *http.Request) (string, error) {
	headerValue := instanceIDFromRequest(r)
	userID := UserIDFromContext(r.Context())

	if headerValue != "" {
		if _, err := s.instances.GetForUser(userID, headerValue); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", errors.New("unknown instance id")
			}
			return "", err
		}
		return headerValue, nil
	}

	activeID, err := s.instances.GetActiveIDForUser(userID)
	if err != nil {
		return "", err
	}
	if activeID == "" {
		return "", nil
	}
	if _, err := s.instances.GetForUser(userID, activeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = s.instances.ClearActiveForUser(userID)
			return "", nil
		}
		return "", err
	}
	return activeID, nil
}

func (s *Server) reloadActiveSubsonic() error {
	inst, err := s.instances.GetActive()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.mu.Lock()
			s.subsonic = subsonic.NewClient("", "", "")
			s.mu.Unlock()
			slog.Debug("no active subsonic instance configured")
			return nil
		}
		return err
	}

	client := subsonic.NewClient(inst.ServerURL, inst.Username, inst.Password)
	s.mu.Lock()
	s.subsonic = client
	s.clientCache[inst.ID] = client
	s.mu.Unlock()
	slog.Info("active subsonic instance loaded",
		"instance_id", inst.ID,
		"url", httputil.RedactURLUserinfo(inst.ServerURL),
		"name", inst.Name,
	)
	return nil
}

func (s *Server) cachedClient(instanceID, serverURL, username, password string) *subsonic.Client {
	s.mu.RLock()
	if client, ok := s.clientCache[instanceID]; ok {
		s.mu.RUnlock()
		return client
	}
	s.mu.RUnlock()

	client := subsonic.NewClient(serverURL, username, password)
	s.mu.Lock()
	s.clientCache[instanceID] = client
	s.mu.Unlock()
	return client
}

func (s *Server) subsonicForContext(ctx context.Context) *subsonic.Client {
	instanceID := InstanceIDFromContext(ctx)
	userID := UserIDFromContext(ctx)
	if instanceID == "" {
		if userID != "" {
			return subsonic.NewClient("", "", "")
		}
		s.mu.RLock()
		client := s.subsonic
		s.mu.RUnlock()
		return client
	}

	activeID, _ := s.instances.GetActiveID()
	s.mu.RLock()
	if userID == "" && activeID == instanceID && s.subsonic != nil && s.subsonic.Enabled() {
		client := s.subsonic
		s.mu.RUnlock()
		return client
	}
	s.mu.RUnlock()

	inst, err := s.instances.GetForUser(userID, instanceID)
	if err != nil {
		return subsonic.NewClient("", "", "")
	}
	return s.cachedClient(instanceID, inst.ServerURL, inst.Username, inst.Password)
}

func (s *Server) registerSubsonicRoutes() {
	s.mux.HandleFunc("GET /api/music/status", func(w http.ResponseWriter, r *http.Request) {
		userID := UserIDFromContext(r.Context())
		mode := s.sourceViewModeForUser(userID)
		instanceID, _ := s.instances.GetActiveIDForUser(userID)
		localLib, localErr := s.localLibraries.GetActiveForUser(userID)

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
			client := s.subsonicForContext(r.Context())
			subsonic.StatusHandler(client, s.cache, s.cfg.CacheEnabled, ResolveProgressUserID)(w, r)
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
		client := s.subsonicForContext(r.Context())
		subsonic.StatusHandler(client, s.cache, s.cfg.CacheEnabled, ResolveProgressUserID)(w, r)
	})
	s.mux.HandleFunc("GET /api/music/library-stats", func(w http.ResponseWriter, r *http.Request) {
		userID := UserIDFromContext(r.Context())
		mode := s.sourceViewModeForUser(userID)
		instanceID, _ := s.instances.GetActiveIDForUser(userID)
		localLib, localErr := s.localLibraries.GetActiveForUser(userID)

		if store.IsUnifiedSourceView(mode) && instanceID != "" && localErr == nil {
			client := s.subsonicForContext(r.Context())
			remote, remoteErr := client.LibraryStats()
			if remoteErr != nil {
				httputil.WriteError(w, http.StatusBadGateway, "bad_gateway", "failed to load library stats")
				return
			}
			artistCount, albumCount, countErr := s.localTracks.CountDistinctArtistsAlbums(localLib.ID)
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
			artistCount, albumCount, countErr := s.localTracks.CountDistinctArtistsAlbums(localLib.ID)
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
			client := s.subsonicForContext(r.Context())
			subsonic.LibraryStatsHandler(client, s.cache, s.cfg.CacheEnabled, ResolveProgressUserID)(w, r)
			return
		}

		if localErr == nil {
			artistCount, albumCount, countErr := s.localTracks.CountDistinctArtistsAlbums(localLib.ID)
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
	s.mux.HandleFunc("POST /api/music/library/refresh", s.handleRefreshLibraryCache)

	proxy := subsonic.NewProxy(s.subsonicForContext, s.cache, s.cfg.CacheEnabled, ResolveProgressUserID)
	s.mux.Handle("/api/subsonic/", proxy)
	s.mux.Handle("/api/subsonic", proxy)
}

func (s *Server) Start() error {
	if s.dlna != nil && s.dlna.Enabled() {
		if err := s.dlna.Start(context.Background()); err != nil {
			slog.Error("dlna start failed", "err", err)
		}
	}
	ln, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return err
	}
	s.listenAddr = ln.Addr().String()
	go func() {
		if serveErr := s.server.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			slog.Error("api server stopped", "err", serveErr)
		}
	}()
	return nil
}

func (s *Server) ListenAddr() string {
	if s.listenAddr != "" {
		return s.listenAddr
	}
	return s.server.Addr
}

func (s *Server) Stop(ctx context.Context) error {
	if s.libraryWatcher != nil {
		s.libraryWatcher.Close()
	}
	if s.dlna != nil {
		_ = s.dlna.Stop(ctx)
	}
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func (s *Server) Handler() http.Handler {
	return s.buildAPIHandler()
}

func (s *Server) buildAPIHandler() http.Handler {
	inner := ChainAuthInstance(s.auth, s.cfg.DemoModeEffective(), s, s.mux)
	inner = DemoReadOnlyMiddleware(s.cfg.DemoModeEffective(), inner)
	inner = CompatMiddleware(inner)
	inner = RequestIDMiddleware(inner)
	inner = LoggingMiddleware(inner)
	inner = RecoverMiddleware(inner)
	inner = MetricsMiddleware(inner)
	api := observability.HTTPMiddleware()(inner)
	return CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/rest/") {
			s.subsonicServer.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/s/") {
			// /rest/ and /s/ bypass the demo middleware chain, so public
			// shares keep only read methods plus the unlock POST in demo mode.
			if s.cfg.DemoModeEffective() && !demoShareMethodAllowed(r) {
				httputil.WriteJSON(w, http.StatusForbidden, map[string]any{
					"error": "demo mode is read-only",
				})
				return
			}
			s.handlePublicShare(w, r)
			return
		}
		api.ServeHTTP(w, r)
	}))
}

func demoShareMethodAllowed(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	case http.MethodPost:
		return strings.HasSuffix(strings.TrimSuffix(r.URL.Path, "/"), "/unlock")
	default:
		return false
	}
}

func (s *Server) handleRefreshLibraryCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	scope := ResolveProgressUserID(r.Context())
	prefix := scope + "|"
	removed := s.cache.InvalidatePrefix(prefix)
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"invalidated": removed})
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	libCfg := s.cfg.LocalLibraryEffective()
	payload := map[string]any{
		"listenAddr":         s.cfg.ListenAddr,
		"publicUrl":          s.cfg.PublicURL,
		"authEnabled":        s.cfg.AuthEnabled(),
		"oidcEnabled":        s.cfg.OIDCEnabled(),
		"oidcLoginUrl":       "/api/auth/oidc/login",
		"serverMode":         s.cfg.ServerMode,
		"demoMode":           s.cfg.DemoModeEffective(),
		"fakeCatalog":        democatalog.UsesFakeCatalog(s.cfg),
		"connectionDefaults": s.cfg.ConnectionDefaults,
		"subsonicServer": map[string]any{
			"enabled": s.cfg.SubsonicServerEffective(),
			"restUrl": "/rest",
		},
		"transcoding": map[string]any{
			"available": transcode.Available(),
		},
		"extensions": map[string]any{},
		"dlna": map[string]any{
			"enabled": s.cfg.DLNAServerEffective(),
			"port":    s.cfg.DLNAPort,
		},
		"jukebox": map[string]any{
			"enabled": true,
		},
		"localLibrary": map[string]any{
			"enabled":         libCfg.Enabled,
			"allowCustomPath": libCfg.AllowCustomPath,
		},
	}
	// Host filesystem paths are only useful to a signed-in client or a local
	// desktop install. Keep them out of the public response on anything that
	// can be reached by strangers (auth enabled and signed out, server mode,
	// demo mode).
	trusted := s.configRequestAuthed(r) ||
		(!s.cfg.AuthEnabled() && !s.cfg.ServerMode && !s.cfg.DemoModeEffective())
	if trusted {
		payload["dataDir"] = s.cfg.DataDir
		payload["extensions"] = map[string]any{
			"dir": extensions.ExtensionsDir(s.cfg.DataDir),
		}
		payload["localLibrary"].(map[string]any)["defaultPath"] = libCfg.DefaultPath
	}
	maps.Copy(payload, compat.ConfigFields())
	if sentryPayload := s.clientSentryPayload(r); sentryPayload != nil {
		payload["sentry"] = sentryPayload
	}
	httputil.WriteJSON(w, http.StatusOK, payload)
}

func (s *Server) configRequestAuthed(r *http.Request) bool {
	if userID := UserIDFromContext(r.Context()); userID != "" {
		return true
	}
	if s.auth == nil {
		return false
	}
	token := sessionTokenFromRequest(r)
	if token == "" {
		return false
	}
	_, err := s.auth.UserIDFromToken(token)
	return err == nil
}

func frontendDevServerEnabled() bool {
	return os.Getenv("FRONTEND_DEVSERVER_URL") != ""
}

func isStaticAssetPath(path string) bool {
	if strings.HasPrefix(path, "/assets/") || strings.HasPrefix(path, "/wails/") {
		return true
	}
	if strings.HasPrefix(path, "/@") || strings.HasPrefix(path, "/src/") || strings.HasPrefix(path, "/node_modules/") || strings.HasPrefix(path, "/bindings/") {
		return true
	}
	name := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		name = path[i+1:]
	}
	return name != "" && strings.Contains(name, ".")
}

func isAPIPath(path string) bool {
	return path == "/health" || path == "/metrics" || path == "/api" ||
		strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/rest/") ||
		strings.HasPrefix(path, "/s/") || strings.HasPrefix(path, "/debug/")
}

func needsSPAFallback(path string) bool {
	if path == "" || path == "/" {
		return false
	}

	name := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		name = path[i+1:]
	}
	if name == "" {
		return false
	}

	return !strings.Contains(name, ".")
}
