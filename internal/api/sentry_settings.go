// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"melovian/internal/appconfig"
	"melovian/internal/httputil"
	"melovian/internal/observability"
	"melovian/internal/store"
)

func (s *Server) registerSentrySettingsRoutes() {
	s.mux.HandleFunc("GET /api/settings/sentry", s.handleGetSentryServerSettings)
	s.mux.HandleFunc("PUT /api/settings/sentry", s.handlePutSentryServerSettings)
	s.mux.HandleFunc("POST /api/settings/sentry/test", s.handlePostSentryTestEvent)
	s.mux.HandleFunc("GET /api/music/settings/sentry-client", s.handleGetSentryClientSettings)
	s.mux.HandleFunc("PUT /api/music/settings/sentry-client", s.handlePutSentryClientSettings)
}

func (s *Server) loadStoredSentrySettings() (appconfig.StoredSentrySettings, error) {
	raw, err := s.db.GetAppSetting(appconfig.SettingSentryServer)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return appconfig.DefaultStoredSentrySettings(), nil
		}
		return appconfig.StoredSentrySettings{}, err
	}
	return appconfig.MergeStoredSentrySettings(json.RawMessage(raw))
}

func (s *Server) saveStoredSentrySettings(settings appconfig.StoredSentrySettings) error {
	encoded, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return s.db.SetAppSetting(appconfig.SettingSentryServer, string(encoded))
}

func (s *Server) sentryEnvConfig() appconfig.SentryConfig {
	return appconfig.LoadSentryConfigFromEnv()
}

func (s *Server) sentryEnvLocks() appconfig.SentryEnvLocks {
	return appconfig.LoadSentryEnvLocks()
}

func (s *Server) mergeEffectiveSentry(stored appconfig.StoredSentrySettings) appconfig.SentryConfig {
	return appconfig.MergeSentryConfig(s.sentryEnvConfig(), stored, s.sentryEnvLocks())
}

func (s *Server) refreshSentryRuntime(stored appconfig.StoredSentrySettings) error {
	effective := s.mergeEffectiveSentry(stored)
	s.mu.Lock()
	s.sentryCfg = effective
	s.mu.Unlock()
	if err := observability.ReinitSentry(effective); err != nil {
		return fmt.Errorf("sentry init: %w", err)
	}
	if effective.Enabled() {
		slog.Info("sentry runtime updated",
			"environment", effective.Environment,
			"release", effective.Release,
		)
	}
	return nil
}

func (s *Server) effectiveSentryConfig() appconfig.SentryConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sentryCfg
}

func (s *Server) initSentryFromStore() error {
	stored, err := s.loadStoredSentrySettings()
	if err != nil {
		return err
	}
	return s.refreshSentryRuntime(stored)
}

func (s *Server) canManageServerSentrySettings(r *http.Request) bool {
	if s.cfg.DemoModeEffective() {
		return false
	}
	if !s.cfg.AuthEnabled() {
		return true
	}
	return UserIDFromContext(r.Context()) != ""
}

func (s *Server) handleGetSentryServerSettings(w http.ResponseWriter, r *http.Request) {
	if !s.canManageServerSentrySettings(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	stored, err := s.loadStoredSentrySettings()
	if err != nil {
		http.Error(w, "failed to load sentry settings", http.StatusInternalServerError)
		return
	}
	effective := s.mergeEffectiveSentry(stored)
	envLocks := s.sentryEnvLocks()

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"stored": stored,
		"effective": map[string]any{
			"enabled":                effective.Enabled(),
			"dsnConfigured":          effective.Enabled(),
			"frontendDsnConfigured":  effective.FrontendDSNEffective() != "",
			"environment":            effective.Environment,
			"release":                effective.Release,
			"tracesSampleRate":       effective.TracesSampleRate,
			"clientReportingAllowed": effective.ClientReportingAllowed,
		},
		"envLocks": envLocks,
	})
}

func (s *Server) handlePutSentryServerSettings(w http.ResponseWriter, r *http.Request) {
	if !s.canManageServerSentrySettings(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	body, err := httputil.ReadJSONBytes(r)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	stored, err := appconfig.MergeStoredSentrySettings(body)
	if err != nil {
		http.Error(w, "invalid sentry settings", http.StatusBadRequest)
		return
	}
	stored.DSN = strings.TrimSpace(stored.DSN)
	stored.FrontendDSN = strings.TrimSpace(stored.FrontendDSN)
	stored.Environment = strings.TrimSpace(stored.Environment)
	stored.Release = strings.TrimSpace(stored.Release)
	if stored.TracesSampleRate < 0 {
		stored.TracesSampleRate = 0
	}
	if stored.TracesSampleRate > 1 {
		stored.TracesSampleRate = 1
	}
	if !stored.Enabled {
		stored.DSN = ""
		stored.FrontendDSN = ""
	}

	envLocks := s.sentryEnvLocks()
	if envLocks.DSN {
		env := s.sentryEnvConfig()
		stored.DSN = env.DSN
		stored.Enabled = env.Enabled()
	}
	if envLocks.FrontendDSN {
		stored.FrontendDSN = s.sentryEnvConfig().FrontendDSN
	}
	if envLocks.Environment {
		stored.Environment = s.sentryEnvConfig().Environment
	}
	if envLocks.Release {
		stored.Release = s.sentryEnvConfig().Release
	}
	if envLocks.TracesSampleRate {
		stored.TracesSampleRate = s.sentryEnvConfig().TracesSampleRate
	}

	if err := s.saveStoredSentrySettings(stored); err != nil {
		http.Error(w, "failed to save sentry settings", http.StatusInternalServerError)
		return
	}
	if err := s.refreshSentryRuntime(stored); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.handleGetSentryServerSettings(w, r)
}

func (s *Server) handlePostSentryTestEvent(w http.ResponseWriter, r *http.Request) {
	if !s.canManageServerSentrySettings(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if !observability.Enabled() {
		http.Error(w, "error tracking is not active. Save a DSN with server tracking enabled first.", http.StatusBadRequest)
		return
	}
	eventID, err := observability.SendTestEvent()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"eventId": eventID,
		"message": "Test event sent.",
	})
}

func (s *Server) loadSentryClientSettings(userID string) (store.SentryClientSettings, error) {
	raw, err := s.preferences.Get(userID, store.PrefKeySentryClient)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.DefaultSentryClientSettings(), nil
		}
		return store.SentryClientSettings{}, err
	}
	settings := store.DefaultSentryClientSettings()
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return store.SentryClientSettings{}, err
	}
	return settings, nil
}

func (s *Server) handleGetSentryClientSettings(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	settings, err := s.loadSentryClientSettings(userID)
	if err != nil {
		http.Error(w, "failed to load client sentry settings", http.StatusInternalServerError)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, settings)
}

func (s *Server) handlePutSentryClientSettings(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	body, err := httputil.ReadJSONBytes(r)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	settings := store.DefaultSentryClientSettings()
	if err := json.Unmarshal(body, &settings); err != nil {
		http.Error(w, "invalid client sentry settings", http.StatusBadRequest)
		return
	}
	encoded, err := json.Marshal(settings)
	if err != nil {
		http.Error(w, "encode client sentry settings", http.StatusInternalServerError)
		return
	}
	if err := s.preferences.Set(userID, store.PrefKeySentryClient, string(encoded)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, settings)
}

func (s *Server) clientSentryPayload(r *http.Request) map[string]any {
	effective := s.effectiveSentryConfig()
	if !effective.Enabled() || !effective.ClientReportingAllowed {
		return nil
	}

	userID := ResolveProgressUserID(r.Context())
	clientSettings, err := s.loadSentryClientSettings(userID)
	if err != nil || !clientSettings.Enabled {
		return nil
	}

	frontendDSN := effective.FrontendDSNEffective()
	if frontendDSN == "" {
		return nil
	}

	return map[string]any{
		"dsn":              frontendDSN,
		"environment":      effective.Environment,
		"release":          effective.Release,
		"tracesSampleRate": effective.TracesSampleRate,
		"clientReporting":  true,
	}
}
