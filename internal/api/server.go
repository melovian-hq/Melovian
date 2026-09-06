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
}

func NewServer(cfg appconfig.Config, db *store.DB) *Server {
	ConfigureCORSOrigins(cfg.CORSOrigins)
	instances := store.NewInstanceStore(db)
	localLibraries := store.NewLocalLibraryStore(db)
	localTracks := store.NewLocalTrackStore(db)
	listen := store.NewListenStore(db)
	preferences := store.NewPreferencesStore(db)
	downloads := store.NewDownloadStore(db)
	responseCache := cache.NewResponseCache()
	events := NewEventHub()

	s := &Server{
		cfg:            cfg,
		db:             db,
		instances:      instances,
		localLibraries: localLibraries,
		localTracks:    localTracks,
		auth:           store.NewAuthStore(db, cfg.AuthSecret),
		listen:         listen,
		preferences:    preferences,
		downloads:      downloads,
		videoLinks:     store.NewTrackVideoLinkStore(db),
		videoClient:    video.NewClient(),
		mux:            http.NewServeMux(),
		cache:          responseCache,
		clientCache:    make(map[string]*subsonic.Client),
		downloadSem:    make(chan struct{}, 3),
		catalogCache:   localmusic.NewCatalogCache(),
		coverCache:     localmusic.NewCoverCache(),
		libraryScanner: metaloader.NewScanner(localLibraries, localTracks),
		events:         events,
		devices:        NewDeviceRegistry(events),
		shares:         store.NewShareStore(db),
		notifications:  store.NewNotificationStore(db),
		jukebox:        jukebox.NewController(),
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
		"url", inst.ServerURL,
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
				http.Error(w, "failed to load library stats", http.StatusBadGateway)
				return
			}
			artistCount, albumCount, countErr := s.localTracks.CountDistinctArtistsAlbums(localLib.ID)
			if countErr != nil {
				http.Error(w, countErr.Error(), http.StatusInternalServerError)
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
				http.Error(w, countErr.Error(), http.StatusInternalServerError)
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
				http.Error(w, countErr.Error(), http.StatusInternalServerError)
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
		http.Error(w, "no music source configured", http.StatusServiceUnavailable)
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
			s.handlePublicShare(w, r)
			return
		}
		api.ServeHTTP(w, r)
	}))
}

func (s *Server) handleRefreshLibraryCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
		"dataDir":            s.cfg.DataDir,
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
		"extensions": map[string]any{
			"dir": extensions.ExtensionsDir(s.cfg.DataDir),
		},
		"dlna": map[string]any{
			"enabled": s.cfg.DLNAServerEffective(),
			"port":    s.cfg.DLNAPort,
		},
		"jukebox": map[string]any{
			"enabled": true,
		},
		"localLibrary": map[string]any{
			"enabled":         libCfg.Enabled,
			"defaultPath":     libCfg.DefaultPath,
			"allowCustomPath": libCfg.AllowCustomPath,
		},
	}
	maps.Copy(payload, compat.ConfigFields())
	if sentryPayload := s.clientSentryPayload(r); sentryPayload != nil {
		payload["sentry"] = sentryPayload
	}
	httputil.WriteJSON(w, http.StatusOK, payload)
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
