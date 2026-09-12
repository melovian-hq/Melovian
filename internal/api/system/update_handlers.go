// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package system

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"melovian/internal/api/apishared"
	"melovian/internal/api/realtime"
	"melovian/internal/appconfig"
	"melovian/internal/cache"
	"melovian/internal/compat"
	"melovian/internal/httputil"
	"melovian/internal/jukebox"
	"melovian/internal/localmusic"
	"melovian/internal/store"
	"melovian/internal/update"
)

// updateCheckTTL bounds how often a status read re-hits the update feed.
const updateCheckTTL = time.Hour

// errUpdateCheckInFlight reports a duplicate check request while a feed
// check is already running.
var errUpdateCheckInFlight = errors.New("update check already in progress")

// UpdateSettings is the per-user update preference blob.
type UpdateSettings struct {
	// AutoUpdate opts the desktop app into background download+install.
	// Off by default. Updates are never applied silently without it.
	AutoUpdate bool `json:"autoUpdate"`
	// Channel is "stable" (default) or "prerelease".
	Channel string `json:"channel"`
	// LastChecked is the most recent feed check time.
	LastChecked time.Time `json:"lastChecked"`
}

func defaultUpdateSettings() UpdateSettings {
	return UpdateSettings{AutoUpdate: false, Channel: string(update.ChannelStable)}
}

func mergeUpdateSettings(raw json.RawMessage) (UpdateSettings, error) {
	settings := defaultUpdateSettings()
	if len(raw) == 0 {
		return settings, nil
	}
	if err := json.Unmarshal(raw, &settings); err != nil {
		return UpdateSettings{}, err
	}
	switch settings.Channel {
	case "", string(update.ChannelStable):
		settings.Channel = string(update.ChannelStable)
	case string(update.ChannelPrerelease):
	default:
		settings.Channel = string(update.ChannelStable)
	}
	return settings, nil
}

// DesktopUpdateHooks lets the Wails shell drive updates through the
// framework updater (staged download, helper swap, relaunch). main.go
// installs these in desktop builds. They stay nil in server mode.
type DesktopUpdateHooks struct {
	Apply   func(ctx context.Context) map[string]any
	Restart func(ctx context.Context) error
	Status  func() map[string]any
}

// SetUpdateHooks installs the desktop updater bridge. Call before serving.
func (h *Handler) SetUpdateHooks(hooks *DesktopUpdateHooks) {
	h.upd.mu.Lock()
	h.upd.updHooks = hooks
	h.upd.mu.Unlock()
}

// updateState is the server-side view of a check or in-flight apply.
type updateState struct {
	mu          sync.Mutex
	checking    bool
	lastCheck   time.Time
	lastResult  *update.CheckResult
	checkErr    string
	applying    bool
	applyStage  string
	applyMsg    string
	applyErr    string
	written     int64
	total       int64
	applyResult *update.ApplyResult
	updHooks    *DesktopUpdateHooks

	// bgCtx and bgWG track detached check/apply work. bgCtx is created
	// lazily on the first background job and cancelled on server shutdown
	// so in-flight downloads stop instead of outliving the listener.
	bgCtx    context.Context
	bgCancel context.CancelFunc
	bgWG     sync.WaitGroup
}

func (h *Handler) loadUpdateSettings(userID string) (UpdateSettings, error) {
	raw, err := h.preferences.Get(userID, store.PrefKeyUpdateSettings)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return defaultUpdateSettings(), nil
		}
		return UpdateSettings{}, err
	}
	return mergeUpdateSettings(json.RawMessage(raw))
}

// canApplyUpdate reports whether this process may self-update: never inside
// a container (the operator pulls a new image) and only when the binary is
// writable.
func canApplyUpdate() bool {
	if inContainer() {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	info, err := os.Stat(exe)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	// The swap renames inside the binary's directory, so the directory must
	// be writable. Probe with a temp file rather than trusting mode bits.
	probe, err := os.CreateTemp(filepath.Dir(exe), ".melovian-wcheck-*")
	if err != nil {
		return false
	}
	_ = probe.Close()
	_ = os.Remove(probe.Name())
	return true
}

func inContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	for _, k := range []string{"MELOVIAN_CONTAINER", "container", "KUBERNETES_SERVICE_HOST"} {
		if os.Getenv(k) != "" {
			return true
		}
	}
	return false
}

func (h *Handler) updateStatusPayload(userID string) map[string]any {
	h.upd.mu.Lock()
	defer h.upd.mu.Unlock()
	settings, _ := h.loadUpdateSettings(userID)
	payload := map[string]any{
		"currentVersion": compat.Version,
		"signed":         update.Pinned(),
		"channel":        settings.Channel,
		"autoUpdate":     settings.AutoUpdate,
		"canApply":       canApplyUpdate(),
		"serverMode":     h.cfg.ServerMode,
		"inContainer":    inContainer(),
		"checking":       h.upd.checking,
		"applying":       h.upd.applying,
		"stage":          h.upd.applyStage,
		"stageMessage":   h.upd.applyMsg,
		"written":        h.upd.written,
		"total":          h.upd.total,
	}
	if !h.upd.lastCheck.IsZero() {
		payload["checkedAt"] = h.upd.lastCheck
	}
	if h.upd.checkErr != "" {
		payload["checkError"] = h.upd.checkErr
	}
	if h.upd.applyErr != "" {
		payload["applyError"] = h.upd.applyErr
	}
	if res := h.upd.lastResult; res != nil {
		payload["upToDate"] = res.UpToDate
		if res.Latest != nil {
			payload["latestVersion"] = res.Latest.Version
			payload["releaseUrl"] = res.Latest.NotesURL
			payload["publishedAt"] = res.Latest.PublishedAt
		}
	}
	if h.upd.applyResult != nil {
		payload["appliedVersion"] = h.upd.applyResult.Version
		payload["applyMethod"] = h.upd.applyResult.Method
	}
	if h.upd.updHooks != nil && h.upd.updHooks.Status != nil {
		payload["desktop"] = h.upd.updHooks.Status()
	}
	return payload
}

// spawnUpdateJob runs fn on the detached update context and tracks it so
// shutdown can cancel and drain in-flight work.
func (h *Handler) spawnUpdateJob(fn func(ctx context.Context)) {
	h.upd.mu.Lock()
	if h.upd.bgCtx == nil {
		h.upd.bgCtx, h.upd.bgCancel = context.WithCancel(context.Background())
		if srv := h.serverFn(); srv != nil {
			srv.RegisterOnShutdown(h.stopUpdateJobs)
		}
	}
	ctx := h.upd.bgCtx
	h.upd.bgWG.Add(1)
	h.upd.mu.Unlock()

	go func() {
		defer h.upd.bgWG.Done()
		fn(ctx)
	}()
}

// stopUpdateJobs cancels detached update work and waits for it to drain.
// It is registered with the HTTP server the first time a job spawns, so
// Server.Stop triggers it.
func (h *Handler) stopUpdateJobs() {
	h.upd.mu.Lock()
	if h.upd.bgCancel != nil {
		h.upd.bgCancel()
	}
	h.upd.mu.Unlock()
	h.upd.bgWG.Wait()
}

// runUpdateCheck performs a feed check once. Concurrent callers share the
// in-flight result via the state fields.
func (h *Handler) runUpdateCheck(ctx context.Context, userID string) (*update.CheckResult, error) {
	settings, err := h.loadUpdateSettings(userID)
	if err != nil {
		settings = defaultUpdateSettings()
	}
	h.upd.mu.Lock()
	if h.upd.checking {
		h.upd.mu.Unlock()
		return nil, errUpdateCheckInFlight
	}
	h.upd.checking = true
	h.upd.checkErr = ""
	h.upd.mu.Unlock()

	res, err := update.Check(ctx, compat.Version, update.Options{
		Channel: update.Channel(settings.Channel),
	})

	h.upd.mu.Lock()
	defer h.upd.mu.Unlock()
	h.upd.checking = false
	h.upd.lastCheck = time.Now()
	if err != nil {
		h.upd.checkErr = err.Error()
		return nil, err
	}
	h.upd.lastResult = res
	return res, nil
}

func (h *Handler) handleGetUpdateStatus(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	h.upd.mu.Lock()
	stale := !h.upd.checking && (h.upd.lastResult == nil || time.Since(h.upd.lastCheck) > updateCheckTTL)
	h.upd.mu.Unlock()
	if stale {
		h.spawnUpdateJob(func(ctx context.Context) {
			_, _ = h.runUpdateCheck(ctx, userID)
		})
	}
	httputil.WriteJSON(w, http.StatusOK, h.updateStatusPayload(userID))
}

func (h *Handler) handleCheckUpdate(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	if _, err := h.runUpdateCheck(r.Context(), userID); err != nil {
		if errors.Is(err, errUpdateCheckInFlight) {
			httputil.WriteJSON(w, http.StatusAccepted, h.updateStatusPayload(userID))
			return
		}
		// The real error is stored on the state and surfaces as checkError
		// in the status payload. Keep feed URLs and internals off the wire.
		httputil.WriteError(w, http.StatusBadGateway, "update_check_failed", "update check failed")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, h.updateStatusPayload(userID))
}

func (h *Handler) handleGetUpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	settings, err := h.loadUpdateSettings(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load update settings")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, settings)
}

func (h *Handler) handlePutUpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	body, err := httputil.ReadJSONBytes(r)
	if err != nil || !json.Valid(body) {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	settings, err := mergeUpdateSettings(body)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_update_settings", "invalid update settings")
		return
	}
	encoded, err := json.Marshal(settings)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "encode update settings")
		return
	}
	if err := h.preferences.Set(userID, store.PrefKeyUpdateSettings, string(encoded)); err != nil {
		httputil.WriteInternalError(w, r, "handlePutUpdateSettings", err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, settings)
}

// handleApplyUpdate runs the self-update pipeline in the background. The
// client polls GET /api/update/status for stage and progress. The binary is
// swapped on disk. The running process exits old code until the supervisor
// or user restarts it.
func (h *Handler) handleApplyUpdate(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	if h.cfg.AuthEnabled() && userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "authentication_required", "authentication required")
		return
	}
	h.upd.mu.Lock()
	hooks := h.upd.updHooks
	h.upd.mu.Unlock()
	if hooks != nil && hooks.Apply != nil {
		// Desktop build: the Wails updater owns staging, swap, and restart.
		payload := hooks.Apply(r.Context())
		status := http.StatusOK
		if _, failed := payload["error"]; failed {
			status = http.StatusBadGateway
		}
		httputil.WriteJSON(w, status, payload)
		return
	}
	if !canApplyUpdate() {
		httputil.WriteError(w, http.StatusConflict, "self_update_is_not_available_on_this_ins", "self-update is not available on this install (container or read-only binary); update via your package manager or image")
		return
	}
	h.upd.mu.Lock()
	if h.upd.applying {
		h.upd.mu.Unlock()
		httputil.WriteJSON(w, http.StatusAccepted, h.updateStatusPayload(userID))
		return
	}
	h.upd.applying = true
	h.upd.applyErr = ""
	h.upd.applyResult = nil
	h.upd.applyStage = string(update.StageCheck)
	h.upd.mu.Unlock()

	settings, _ := h.loadUpdateSettings(userID)
	var req struct {
		Version string `json:"version"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	h.spawnUpdateJob(func(ctx context.Context) {
		_, err := update.Apply(ctx, compat.Version, update.Options{
			Channel:       update.Channel(settings.Channel),
			TargetVersion: req.Version,
			OnProgress: func(p update.Progress) {
				h.upd.mu.Lock()
				h.upd.applyStage = string(p.Stage)
				h.upd.applyMsg = p.Message
				h.upd.written = p.Written
				h.upd.total = p.Total
				h.upd.mu.Unlock()
			},
		})
		h.upd.mu.Lock()
		defer h.upd.mu.Unlock()
		h.upd.applying = false
		if err != nil {
			h.upd.applyErr = err.Error()
			return
		}
		h.upd.applyStage = string(update.StageDone)
		h.upd.applyMsg = "Update installed; restart to run the new version"
	})

	httputil.WriteJSON(w, http.StatusAccepted, h.updateStatusPayload(userID))
}

// handleRestartUpdate asks the desktop shell to relaunch into the staged
// update. Server mode has no staged-update concept. A supervisor restarts
// the process instead.
func (h *Handler) handleRestartUpdate(w http.ResponseWriter, r *http.Request) {
	h.upd.mu.Lock()
	hooks := h.upd.updHooks
	h.upd.mu.Unlock()
	if hooks == nil || hooks.Restart == nil {
		httputil.WriteError(w, http.StatusConflict, "restart_into_update_is_only_available_on", "restart-into-update is only available on desktop builds")
		return
	}
	if err := hooks.Restart(r.Context()); err != nil {
		httputil.WriteJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"restarting": true})
}

// Handler serves the server-level routes: update management, sentry
// settings, client logs, debug endpoints, metrics, and jukebox control.
type Handler struct {
	cfg              appconfig.Config
	db               *store.DB
	preferences      *store.PreferencesStore
	cache            *cache.ResponseCache
	catalogCache     *localmusic.CatalogCache
	coverCache       *localmusic.CoverCache
	events           *realtime.EventHub
	downloadSem      chan struct{}
	resolver         *apishared.Resolver
	clientLogLimiter *apishared.RateLimiter
	jukebox          *jukebox.Controller
	localTracks      *store.LocalTrackStore
	serverFn         func() *http.Server

	mu        sync.RWMutex
	sentryCfg appconfig.SentryConfig
	upd       updateState
}

type Deps struct {
	Config           appconfig.Config
	DB               *store.DB
	Preferences      *store.PreferencesStore
	Cache            *cache.ResponseCache
	CatalogCache     *localmusic.CatalogCache
	CoverCache       *localmusic.CoverCache
	Events           *realtime.EventHub
	DownloadSem      chan struct{}
	Resolver         *apishared.Resolver
	ClientLogLimiter *apishared.RateLimiter
	Jukebox          *jukebox.Controller
	LocalTracks      *store.LocalTrackStore
	ServerFn         func() *http.Server
}

func New(d Deps) *Handler {
	return &Handler{
		cfg:              d.Config,
		db:               d.DB,
		preferences:      d.Preferences,
		cache:            d.Cache,
		catalogCache:     d.CatalogCache,
		coverCache:       d.CoverCache,
		events:           d.Events,
		downloadSem:      d.DownloadSem,
		resolver:         d.Resolver,
		clientLogLimiter: d.ClientLogLimiter,
		jukebox:          d.Jukebox,
		localTracks:      d.LocalTracks,
		serverFn:         d.ServerFn,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	h.registerMetricsRoutes(mux)
	h.registerDebugRoutes(mux)
	h.registerClientLogRoutes(mux)
	h.registerSentrySettingsRoutes(mux)
	h.registerJukeboxRoutes(mux)
	h.registerUpdateRoutes(mux)
}

func (h *Handler) registerUpdateRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/update/status", h.handleGetUpdateStatus)
	mux.HandleFunc("POST /api/update/check", h.handleCheckUpdate)
	mux.HandleFunc("GET /api/update/settings", h.handleGetUpdateSettings)
	mux.HandleFunc("PUT /api/update/settings", h.handlePutUpdateSettings)
	mux.HandleFunc("POST /api/update/apply", h.handleApplyUpdate)
	mux.HandleFunc("POST /api/update/restart", h.handleRestartUpdate)
}
