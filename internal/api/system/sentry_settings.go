// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package system

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"melovian/internal/api/apishared"
	"melovian/internal/appconfig"
	"melovian/internal/httputil"
	"melovian/internal/melog"
	"melovian/internal/observability"
	"melovian/internal/store"
)

func (h *Handler) registerSentrySettingsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/settings/sentry", h.handleGetSentryServerSettings)
	mux.HandleFunc("PUT /api/settings/sentry", h.handlePutSentryServerSettings)
	mux.HandleFunc("POST /api/settings/sentry/test", h.handlePostSentryTestEvent)
	mux.HandleFunc("GET /api/music/settings/sentry-client", h.handleGetSentryClientSettings)
	mux.HandleFunc("PUT /api/music/settings/sentry-client", h.handlePutSentryClientSettings)
}

func (h *Handler) loadStoredSentrySettings() (appconfig.StoredSentrySettings, error) {
	raw, err := h.db.GetAppSetting(appconfig.SettingSentryServer)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return appconfig.DefaultStoredSentrySettings(), nil
		}
		return appconfig.StoredSentrySettings{}, err
	}
	return appconfig.MergeStoredSentrySettings(json.RawMessage(raw))
}

func (h *Handler) saveStoredSentrySettings(settings appconfig.StoredSentrySettings) error {
	encoded, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return h.db.SetAppSetting(appconfig.SettingSentryServer, string(encoded))
}

func (h *Handler) sentryEnvConfig() appconfig.SentryConfig {
	return appconfig.LoadSentryConfigFromEnv()
}

func (h *Handler) sentryEnvLocks() appconfig.SentryEnvLocks {
	return appconfig.LoadSentryEnvLocks()
}

func (h *Handler) mergeEffectiveSentry(stored appconfig.StoredSentrySettings) appconfig.SentryConfig {
	return appconfig.MergeSentryConfig(h.sentryEnvConfig(), stored, h.sentryEnvLocks())
}

func (h *Handler) RefreshSentryRuntime(stored appconfig.StoredSentrySettings) error {
	effective := h.mergeEffectiveSentry(stored)
	h.mu.Lock()
	h.sentryCfg = effective
	h.mu.Unlock()
	if err := observability.ReinitSentry(effective); err != nil {
		return fmt.Errorf("sentry init: %w", err)
	}
	if effective.Enabled() {
		slog.Info("sentry runtime updated",
			"environment", melog.Sanitize(effective.Environment),
			"release", melog.Sanitize(effective.Release),
		)
	}
	return nil
}

func (h *Handler) EffectiveSentryConfig() appconfig.SentryConfig {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.sentryCfg
}

func (h *Handler) InitSentryFromStore() error {
	stored, err := h.loadStoredSentrySettings()
	if err != nil {
		return err
	}
	return h.RefreshSentryRuntime(stored)
}

func (h *Handler) canManageServerSentrySettings(r *http.Request) bool {
	if h.cfg.DemoModeEffective() {
		return false
	}
	if !h.cfg.AuthEnabled() {
		return true
	}
	return apishared.UserIDFromContext(r.Context()) != ""
}

func (h *Handler) handleGetSentryServerSettings(w http.ResponseWriter, r *http.Request) {
	if !h.canManageServerSentrySettings(r) {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
		return
	}

	stored, err := h.loadStoredSentrySettings()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load sentry settings")
		return
	}
	effective := h.mergeEffectiveSentry(stored)
	envLocks := h.sentryEnvLocks()

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

func (h *Handler) handlePutSentryServerSettings(w http.ResponseWriter, r *http.Request) {
	if !h.canManageServerSentrySettings(r) {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
		return
	}

	body, err := httputil.ReadJSONBytes(r)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	stored, err := appconfig.MergeStoredSentrySettings(body)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_sentry_settings", "invalid sentry settings")
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

	envLocks := h.sentryEnvLocks()
	if envLocks.DSN {
		env := h.sentryEnvConfig()
		stored.DSN = env.DSN
		stored.Enabled = env.Enabled()
	}
	if envLocks.FrontendDSN {
		stored.FrontendDSN = h.sentryEnvConfig().FrontendDSN
	}
	if envLocks.Environment {
		stored.Environment = h.sentryEnvConfig().Environment
	}
	if envLocks.Release {
		stored.Release = h.sentryEnvConfig().Release
	}
	if envLocks.TracesSampleRate {
		stored.TracesSampleRate = h.sentryEnvConfig().TracesSampleRate
	}

	if err := h.saveStoredSentrySettings(stored); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to save sentry settings")
		return
	}
	if err := h.RefreshSentryRuntime(stored); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	h.handleGetSentryServerSettings(w, r)
}

func (h *Handler) handlePostSentryTestEvent(w http.ResponseWriter, r *http.Request) {
	if !h.canManageServerSentrySettings(r) {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
		return
	}
	if !observability.Enabled() {
		httputil.WriteError(w, http.StatusBadRequest, "error_tracking_is_not_active_save_a_dsn_", "error tracking is not active. Save a DSN with server tracking enabled first.")
		return
	}
	eventID, err := observability.SendTestEvent()
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"eventId": eventID,
		"message": "Test event sent.",
	})
}

func (h *Handler) LoadSentryClientSettings(userID string) (store.SentryClientSettings, error) {
	raw, err := h.preferences.Get(userID, store.PrefKeySentryClient)
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

func (h *Handler) handleGetSentryClientSettings(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	settings, err := h.LoadSentryClientSettings(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load client sentry settings")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, settings)
}

func (h *Handler) handlePutSentryClientSettings(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	body, err := httputil.ReadJSONBytes(r)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	settings := store.DefaultSentryClientSettings()
	if err := json.Unmarshal(body, &settings); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_client_sentry_settings", "invalid client sentry settings")
		return
	}
	encoded, err := json.Marshal(settings)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "encode client sentry settings")
		return
	}
	if err := h.preferences.Set(userID, store.PrefKeySentryClient, string(encoded)); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, settings)
}

func (h *Handler) ClientSentryPayload(r *http.Request) map[string]any {
	effective := h.EffectiveSentryConfig()
	if !effective.Enabled() || !effective.ClientReportingAllowed {
		return nil
	}

	userID := apishared.ResolveProgressUserID(r.Context())
	clientSettings, err := h.LoadSentryClientSettings(userID)
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
