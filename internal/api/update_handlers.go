// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

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

	"melovian/internal/compat"
	"melovian/internal/httputil"
	"melovian/internal/store"
	"melovian/internal/update"
)

// updateCheckTTL bounds how often a status read re-hits the update feed.
const updateCheckTTL = time.Hour

// UpdateSettings is the per-user update preference blob.
type UpdateSettings struct {
	// AutoUpdate opts the desktop app into background download+install.
	// Off by default. Updates are never applied silently without it.
	AutoUpdate bool `json:"autoUpdate"`
	// Channel is "stable" (default) or "prerelease".
	Channel string `json:"channel"`
	// LastChecked is the most recent feed check time.
	LastChecked time.Time `json:"lastChecked,omitempty"`
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
func (s *Server) SetUpdateHooks(h *DesktopUpdateHooks) {
	s.upd.mu.Lock()
	s.upd.updHooks = h
	s.upd.mu.Unlock()
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
}

func (s *Server) loadUpdateSettings(userID string) (UpdateSettings, error) {
	raw, err := s.preferences.Get(userID, store.PrefKeyUpdateSettings)
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
func (s *Server) canApplyUpdate() bool {
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
	probe.Close()
	os.Remove(probe.Name())
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

func (s *Server) updateStatusPayload(userID string) map[string]any {
	s.upd.mu.Lock()
	defer s.upd.mu.Unlock()
	settings, _ := s.loadUpdateSettings(userID)
	payload := map[string]any{
		"currentVersion": compat.Version,
		"signed":         update.Pinned(),
		"channel":        settings.Channel,
		"autoUpdate":     settings.AutoUpdate,
		"canApply":       s.canApplyUpdate(),
		"serverMode":     s.cfg.ServerMode,
		"inContainer":    inContainer(),
		"checking":       s.upd.checking,
		"applying":       s.upd.applying,
		"stage":          s.upd.applyStage,
		"stageMessage":   s.upd.applyMsg,
		"written":        s.upd.written,
		"total":          s.upd.total,
	}
	if !s.upd.lastCheck.IsZero() {
		payload["checkedAt"] = s.upd.lastCheck
	}
	if s.upd.checkErr != "" {
		payload["checkError"] = s.upd.checkErr
	}
	if s.upd.applyErr != "" {
		payload["applyError"] = s.upd.applyErr
	}
	if res := s.upd.lastResult; res != nil {
		payload["upToDate"] = res.UpToDate
		if res.Latest != nil {
			payload["latestVersion"] = res.Latest.Version
			payload["releaseUrl"] = res.Latest.NotesURL
			payload["publishedAt"] = res.Latest.PublishedAt
		}
	}
	if s.upd.applyResult != nil {
		payload["appliedVersion"] = s.upd.applyResult.Version
		payload["applyMethod"] = s.upd.applyResult.Method
	}
	if s.upd.updHooks != nil && s.upd.updHooks.Status != nil {
		payload["desktop"] = s.upd.updHooks.Status()
	}
	return payload
}

// runUpdateCheck performs a feed check once. Concurrent callers share the
// in-flight result via the state fields.
func (s *Server) runUpdateCheck(ctx context.Context, userID string) (*update.CheckResult, error) {
	settings, err := s.loadUpdateSettings(userID)
	if err != nil {
		settings = defaultUpdateSettings()
	}
	s.upd.mu.Lock()
	if s.upd.checking {
		s.upd.mu.Unlock()
		return nil, errors.New("update check already in progress")
	}
	s.upd.checking = true
	s.upd.checkErr = ""
	s.upd.mu.Unlock()

	res, err := update.Check(ctx, compat.Version, update.Options{
		Channel: update.Channel(settings.Channel),
	})

	s.upd.mu.Lock()
	defer s.upd.mu.Unlock()
	s.upd.checking = false
	s.upd.lastCheck = time.Now()
	if err != nil {
		s.upd.checkErr = err.Error()
		return nil, err
	}
	s.upd.lastResult = res
	return res, nil
}

func (s *Server) handleGetUpdateStatus(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	s.upd.mu.Lock()
	stale := s.upd.lastResult == nil || time.Since(s.upd.lastCheck) > updateCheckTTL
	s.upd.mu.Unlock()
	if stale {
		go func() { _, _ = s.runUpdateCheck(context.Background(), userID) }()
	}
	httputil.WriteJSON(w, http.StatusOK, s.updateStatusPayload(userID))
}

func (s *Server) handleCheckUpdate(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	res, err := s.runUpdateCheck(r.Context(), userID)
	if err != nil {
		if err.Error() == "update check already in progress" {
			httputil.WriteJSON(w, http.StatusAccepted, s.updateStatusPayload(userID))
			return
		}
		httputil.WriteJSON(w, http.StatusBadGateway, map[string]any{
			"error": err.Error(),
		})
		return
	}
	_ = res
	httputil.WriteJSON(w, http.StatusOK, s.updateStatusPayload(userID))
}

func (s *Server) handleGetUpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	settings, err := s.loadUpdateSettings(userID)
	if err != nil {
		http.Error(w, "failed to load update settings", http.StatusInternalServerError)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, settings)
}

func (s *Server) handlePutUpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	body, err := httputil.ReadJSONBytes(r)
	if err != nil || !json.Valid(body) {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	settings, err := mergeUpdateSettings(body)
	if err != nil {
		http.Error(w, "invalid update settings", http.StatusBadRequest)
		return
	}
	encoded, err := json.Marshal(settings)
	if err != nil {
		http.Error(w, "encode update settings", http.StatusInternalServerError)
		return
	}
	if err := s.preferences.Set(userID, store.PrefKeyUpdateSettings, string(encoded)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, settings)
}

// handleApplyUpdate runs the self-update pipeline in the background. The
// client polls GET /api/update/status for stage and progress. The binary is
// swapped on disk. The running process exits old code until the supervisor
// or user restarts it.
func (s *Server) handleApplyUpdate(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	if s.cfg.AuthEnabled() && userID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	s.upd.mu.Lock()
	hooks := s.upd.updHooks
	s.upd.mu.Unlock()
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
	if !s.canApplyUpdate() {
		http.Error(w, "self-update is not available on this install (container or read-only binary); update via your package manager or image", http.StatusConflict)
		return
	}
	s.upd.mu.Lock()
	if s.upd.applying {
		s.upd.mu.Unlock()
		httputil.WriteJSON(w, http.StatusAccepted, s.updateStatusPayload(userID))
		return
	}
	s.upd.applying = true
	s.upd.applyErr = ""
	s.upd.applyResult = nil
	s.upd.applyStage = string(update.StageCheck)
	s.upd.mu.Unlock()

	settings, _ := s.loadUpdateSettings(userID)
	var req struct {
		Version string `json:"version"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	go func() {
		_, err := update.Apply(context.Background(), compat.Version, update.Options{
			Channel:       update.Channel(settings.Channel),
			TargetVersion: req.Version,
			OnProgress: func(p update.Progress) {
				s.upd.mu.Lock()
				s.upd.applyStage = string(p.Stage)
				s.upd.applyMsg = p.Message
				s.upd.written = p.Written
				s.upd.total = p.Total
				s.upd.mu.Unlock()
			},
		})
		s.upd.mu.Lock()
		defer s.upd.mu.Unlock()
		s.upd.applying = false
		if err != nil {
			s.upd.applyErr = err.Error()
			return
		}
		s.upd.applyStage = string(update.StageDone)
		s.upd.applyMsg = "Update installed; restart to run the new version"
	}()

	httputil.WriteJSON(w, http.StatusAccepted, s.updateStatusPayload(userID))
}

// handleRestartUpdate asks the desktop shell to relaunch into the staged
// update. Server mode has no staged-update concept. A supervisor restarts
// the process instead.
func (s *Server) handleRestartUpdate(w http.ResponseWriter, r *http.Request) {
	s.upd.mu.Lock()
	hooks := s.upd.updHooks
	s.upd.mu.Unlock()
	if hooks == nil || hooks.Restart == nil {
		http.Error(w, "restart-into-update is only available on desktop builds", http.StatusConflict)
		return
	}
	if err := hooks.Restart(r.Context()); err != nil {
		httputil.WriteJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"restarting": true})
}

func (s *Server) registerUpdateRoutes() {
	s.mux.HandleFunc("GET /api/update/status", s.handleGetUpdateStatus)
	s.mux.HandleFunc("POST /api/update/check", s.handleCheckUpdate)
	s.mux.HandleFunc("GET /api/update/settings", s.handleGetUpdateSettings)
	s.mux.HandleFunc("PUT /api/update/settings", s.handlePutUpdateSettings)
	s.mux.HandleFunc("POST /api/update/apply", s.handleApplyUpdate)
	s.mux.HandleFunc("POST /api/update/restart", s.handleRestartUpdate)
}
