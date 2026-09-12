// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"strings"

	"melovian/internal/api/apishared"
	"melovian/internal/api/auth"
	downloadsapi "melovian/internal/api/downloads"
	extapi "melovian/internal/api/extensions"
	instancesapi "melovian/internal/api/instances"
	"melovian/internal/api/library"
	"melovian/internal/api/music"
	"melovian/internal/api/notify"
	"melovian/internal/api/realtime"
	"melovian/internal/api/sharing"
	"melovian/internal/api/system"
	"melovian/internal/api/videos"
	"melovian/internal/appconfig"
	"melovian/internal/cache"
	"melovian/internal/compat"
	"melovian/internal/consts"
	"melovian/internal/democatalog"
	"melovian/internal/dlna"
	"melovian/internal/extensions"
	"melovian/internal/httputil"
	"melovian/internal/jukebox"
	"melovian/internal/localmusic"
	"melovian/internal/observability"
	"melovian/internal/store"
	"melovian/internal/subsonic"
	"melovian/internal/subsonicserver"
	"melovian/internal/transcode"
	"melovian/internal/video"
)

type Server struct {
	cfg              appconfig.Config
	instances        *store.InstanceStore
	localLibraries   *store.LocalLibraryStore
	localTracks      *store.LocalTrackStore
	auth             *store.AuthStore
	listen           *store.ListenStore
	preferences      *store.PreferencesStore
	downloads        *store.DownloadStore
	videoLinks       *store.TrackVideoLinkStore
	resolver         *apishared.Resolver
	mux              *http.ServeMux
	server           *http.Server
	listenAddr       string
	cache            *cache.ResponseCache
	downloadSem      chan struct{}
	catalogCache     *localmusic.CatalogCache
	coverCache       *localmusic.CoverCache
	events           *realtime.EventHub
	devices          *realtime.DeviceRegistry
	realtimeH        *realtime.Handler
	libraryH         *library.Handler
	downloadsH       *downloadsapi.Handler
	videosH          *videos.Handler
	sharingH         *sharing.Handler
	systemH          *system.Handler
	instancesH       *instancesapi.Handler
	lyricsH          *music.LyricsHandler
	notifyH          *notify.Handler
	authH            *auth.Handler
	subsonicServer   *subsonicserver.Server
	shares           *store.ShareStore
	notifications    *store.NotificationStore
	jukebox          *jukebox.Controller
	dlna             *dlna.Server
	db               *store.DB
	authLimiter      *apishared.RateLimiter
	shareLimiter     *apishared.RateLimiter
	clientLogLimiter *apishared.RateLimiter
}

func NewServer(cfg appconfig.Config, db *store.DB) *Server {
	ConfigureCORSOrigins(cfg.CORSOrigins)
	instances := store.NewInstanceStore(db)
	preferences := store.NewPreferencesStore(db)
	if cipher, err := store.LoadSecretCipher(cfg.DataDir); err != nil {
		slog.Error("instance credential encryption unavailable", "err", err)
	} else if cipher != nil {
		instances.SetCipher(cipher)
		preferences.SetCipher(cipher)
		if err := instances.MigratePasswords(); err != nil {
			slog.Error("instance password migration failed", "err", err)
		}
		if err := preferences.EnsureSecretCipher(cfg.DataDir); err != nil {
			slog.Error("preference secret migration failed", "err", err)
		}
	}
	localLibraries := store.NewLocalLibraryStore(db)
	localTracks := store.NewLocalTrackStore(db)
	listen := store.NewListenStore(db)
	downloads := store.NewDownloadStore(db)
	responseCache := cache.NewResponseCache()
	events := realtime.NewEventHub()

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
		mux:              http.NewServeMux(),
		cache:            responseCache,
		resolver:         apishared.NewResolver(instances),
		downloadSem:      make(chan struct{}, 3),
		catalogCache:     localmusic.NewCatalogCache(),
		coverCache:       localmusic.NewCoverCache(),
		events:           events,
		devices:          realtime.NewDeviceRegistry(events),
		shares:           store.NewShareStore(db),
		notifications:    store.NewNotificationStore(db),
		jukebox:          jukebox.NewController(),
		authLimiter:      apishared.NewRateLimiter(consts.AuthRateLimit, consts.AuthRateWindow),
		shareLimiter:     apishared.NewRateLimiter(consts.ShareRateLimit, consts.ShareRateWindow),
		clientLogLimiter: apishared.NewRateLimiter(consts.ClientLogRateLimit, consts.ClientLogRateWindow),
	}
	s.devices.SetUsernameLookup(func(userID string) string {
		if userID == "" || s.auth == nil {
			return ""
		}
		u, err := s.auth.GetUser(userID)
		if err != nil {
			return ""
		}
		return u.Username
	})
	s.dlna = dlna.New(
		cfg.DLNAServerEffective(),
		cfg.DLNAHost,
		cfg.DLNAPort,
		apishared.PublicBaseURL(s.cfg),
		s.dlnaCatalogAdapter(),
	)
	s.subsonicServer = s.newSubsonicServer()
	_ = s.resolver.ReloadActive()
	if err := extensions.InstallBundled(cfg.DataDir); err != nil {
		slog.Error("failed to install bundled extensions", "err", err)
	}

	musicSvc := music.NewMusicService(listen, preferences, cfg.DataDir, httputil.NewRetryHTTPClient())
	musicSvc.Register(s.mux)

	s.mux.HandleFunc("GET /api/config", s.handleGetConfig)
	s.authH = auth.New(s.auth, s.cfg, s.authLimiter)
	s.authH.Register(s.mux)
	s.systemH = system.New(system.Deps{
		Config:           s.cfg,
		DB:               s.db,
		Preferences:      s.preferences,
		Cache:            s.cache,
		CatalogCache:     s.catalogCache,
		CoverCache:       s.coverCache,
		Events:           s.events,
		DownloadSem:      s.downloadSem,
		Resolver:         s.resolver,
		ClientLogLimiter: s.clientLogLimiter,
		Jukebox:          s.jukebox,
		LocalTracks:      s.localTracks,
		ServerFn:         func() *http.Server { return s.server },
	})
	s.systemH.Register(s.mux)
	if err := s.systemH.InitSentryFromStore(); err != nil {
		slog.Error("sentry init failed", "err", err)
	}
	s.instancesH = instancesapi.New(s.instances, s.localLibraries, s.preferences, s.resolver, s.cfg)
	s.instancesH.Register(s.mux)
	s.notifyH = notify.New(s.notifications, s.events, s.cfg)
	s.notifyH.Register(s.mux)
	s.realtimeH = realtime.New(s.auth, s.events, s.devices, s.notifyH.Notify)
	s.realtimeH.Register(s.mux)
	s.libraryH = library.New(library.Deps{
		Config:       &s.cfg,
		Instances:    s.instances,
		Preferences:  s.preferences,
		Listen:       s.listen,
		Libraries:    s.localLibraries,
		Tracks:       s.localTracks,
		CatalogCache: s.catalogCache,
		CoverCache:   s.coverCache,
		Events:       s.events,
		Resolver:     s.resolver,
	})
	s.libraryH.Register(s.mux)
	s.videosH = videos.New(s.preferences, s.videoLinks, video.NewClient(), s.localLibraries, s.localTracks, s.libraryH)
	s.videosH.Register(s.mux)
	extensionsH := extapi.New(s.cfg)
	extensionsH.Register(s.mux)
	music.NewProxyHandler(s.cfg, s.instances, s.localLibraries, s.localTracks, s.preferences, s.resolver, s.cache).Register(s.mux)
	s.downloadsH = downloadsapi.New(downloadsapi.Deps{
		Config:      s.cfg,
		Downloads:   s.downloads,
		Preferences: s.preferences,
		Listen:      s.listen,
		Libraries:   s.localLibraries,
		Tracks:      s.localTracks,
		Resolver:    s.resolver,
		Library:     s.libraryH,
		DownloadSem: s.downloadSem,
	})
	s.downloadsH.Register(s.mux)
	s.lyricsH = music.NewLyricsHandler(s.cfg, s.preferences, s.resolver)
	s.lyricsH.Register(s.mux)
	s.sharingH = sharing.New(sharing.Deps{
		Config:      s.cfg,
		Auth:        s.auth,
		Shares:      s.shares,
		Instances:   s.instances,
		Libraries:   s.localLibraries,
		Tracks:      s.localTracks,
		Listen:      s.listen,
		Preferences: s.preferences,
		Resolver:    s.resolver,
		Devices:     s.devices,
		Limiter:     s.shareLimiter,
		Library:     s.libraryH,
		Notify:      s.notifyH.Notify,
	})
	s.sharingH.Register(s.mux)

	handler := s.buildAPIHandler()
	s.server = &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: consts.UpstreamTimeout,
	}

	return s
}

func (s *Server) ResolveInstanceID(r *http.Request) (string, error) {
	return s.resolver.ResolveInstanceID(r)
}

// reloadActiveSubsonic is kept for tests that exercise instance switching.
func (s *Server) reloadActiveSubsonic() error {
	return s.resolver.ReloadActive()
}

// subsonicForContext is kept for tests that exercise instance isolation.
func (s *Server) subsonicForContext(ctx context.Context) *subsonic.Client {
	return s.resolver.ForContext(ctx)
}

// writeSession is kept for tests that exercise session cookies.
func (s *Server) writeSession(w http.ResponseWriter, r *http.Request, userID string) error {
	return s.authH.WriteSession(w, r, userID)
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
	if s.libraryH != nil {
		s.libraryH.Close()
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
	inner = system.MetricsMiddleware(inner)
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
			s.sharingH.HandlePublicShare(w, r)
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
	if sentryPayload := s.systemH.ClientSentryPayload(r); sentryPayload != nil {
		payload["sentry"] = sentryPayload
	}
	httputil.WriteJSON(w, http.StatusOK, payload)
}

func (s *Server) configRequestAuthed(r *http.Request) bool {
	if userID := apishared.UserIDFromContext(r.Context()); userID != "" {
		return true
	}
	if s.auth == nil {
		return false
	}
	token := apishared.SessionTokenFromRequest(r)
	if token == "" {
		return false
	}
	_, err := s.auth.UserIDFromToken(token)
	return err == nil
}

func frontendDevServerEnabled() bool {
	return appconfig.FrontendDevServerEnabled()
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
