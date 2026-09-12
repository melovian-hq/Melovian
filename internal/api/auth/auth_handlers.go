// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"maps"
	"net/http"
	"strings"
	"sync"
	"time"

	"database/sql"
	"errors"

	"melovian/internal/api/apishared"
	"melovian/internal/appconfig"
	"melovian/internal/compat"
	"melovian/internal/democatalog"
	"melovian/internal/httputil"
	"melovian/internal/store"
)

// Handler serves the auth, session, and OIDC routes.
type Handler struct {
	auth    *store.AuthStore
	cfg     appconfig.Config
	limiter *apishared.RateLimiter

	setupMu          sync.Mutex
	oidcOnce         sync.Once
	oidcInitErr      error
	oidcRuntimeValue *oidcRuntime
}

func New(auth *store.AuthStore, cfg appconfig.Config, limiter *apishared.RateLimiter) *Handler {
	return &Handler{auth: auth, cfg: cfg, limiter: limiter}
}

func (h *Handler) Register(mux *http.ServeMux) {
	h.registerAuthRoutes(mux)
	h.registerOIDCRoutes(mux)
}

func (h *Handler) registerAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth/status", h.handleAuthStatus)
	mux.HandleFunc("POST /api/auth/setup", h.handleAuthSetup)
	mux.HandleFunc("POST /api/auth/login", h.handleAuthLogin)
	mux.HandleFunc("POST /api/auth/logout", h.handleAuthLogout)
	mux.HandleFunc("GET /api/auth/sessions", h.handleListSessions)
	mux.HandleFunc("DELETE /api/auth/sessions/{id}", h.handleRevokeSession)
	mux.HandleFunc("POST /api/auth/change-password", h.handleChangePassword)
	mux.HandleFunc("POST /api/auth/username", h.handleChangeUsername)
	mux.HandleFunc("POST /api/auth/delete-account", h.handleDeleteAccount)
	mux.HandleFunc("GET /api/auth/subsonic-key", h.handleGetSubsonicKey)
	mux.HandleFunc("POST /api/auth/subsonic-key/rotate", h.handleRotateSubsonicKey)
	mux.HandleFunc("GET /health", h.handleHealth)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	payload := map[string]any{"status": "ok"}
	maps.Copy(payload, compat.ConfigFields())
	httputil.WriteJSON(w, http.StatusOK, payload)
}

func (h *Handler) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	demo := h.cfg.DemoModeEffective()
	enabled := !demo && h.auth != nil && h.auth.Enabled()
	payload := map[string]any{
		"enabled":       enabled,
		"authenticated": false,
		"setupRequired": false,
		"oidcEnabled":   h.cfg.OIDCEnabled(),
		"oidcLoginUrl":  "/api/auth/oidc/login",
		"demoMode":      demo,
		"fakeCatalog":   democatalog.UsesFakeCatalog(h.cfg),
	}
	if !enabled {
		httputil.WriteJSON(w, http.StatusOK, payload)
		return
	}

	count, err := h.auth.CountUsers()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to read auth status")
		return
	}
	payload["setupRequired"] = count == 0 && !h.cfg.OIDCEnabled()
	payload["localLoginEnabled"] = !h.cfg.OIDCEnabled() || count > 0

	userID := apishared.UserIDFromContext(r.Context())
	if userID == "" {
		if token := apishared.SessionTokenFromRequest(r); token != "" {
			userID, _ = h.auth.UserIDFromToken(token)
		}
	}
	if userID == "" {
		httputil.WriteJSON(w, http.StatusOK, payload)
		return
	}

	user, err := h.auth.GetUser(userID)
	if err != nil {
		httputil.WriteJSON(w, http.StatusOK, payload)
		return
	}

	payload["authenticated"] = true
	payload["user"] = map[string]any{
		"id":       user.ID,
		"username": user.Username,
	}
	httputil.WriteJSON(w, http.StatusOK, payload)
}

type authCredentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) handleAuthSetup(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil || !h.auth.Enabled() {
		httputil.WriteError(w, http.StatusNotFound, "auth_disabled", "auth disabled")
		return
	}
	if h.cfg.OIDCEnabled() {
		httputil.WriteError(w, http.StatusForbidden, "local_account_setup_disabled_when_oidc_i", "local account setup disabled when oidc is enabled")
		return
	}
	if h.loginKeysRateLimited(w, h.loginRateLimitKeys(r, "")) {
		return
	}

	// The user-count check and the insert must be atomic for this public
	// endpoint, otherwise two concurrent requests can both create accounts.
	h.setupMu.Lock()
	defer h.setupMu.Unlock()

	count, err := h.auth.CountUsers()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to read users")
		return
	}
	if count > 0 {
		httputil.WriteError(w, http.StatusConflict, "setup_already_completed", "setup already completed")
		return
	}

	var req authCredentialsRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	user, err := h.auth.CreateUser(req.Username, req.Password)
	if err != nil {
		if h.limiter != nil {
			h.limiter.Record(h.loginRateLimitKeys(r, "")...)
		}
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	if err := h.WriteSession(w, r, user.ID); err != nil {
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, map[string]any{
		"user": map[string]any{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (h *Handler) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil || !h.auth.Enabled() {
		httputil.WriteError(w, http.StatusNotFound, "auth_disabled", "auth disabled")
		return
	}

	var req authCredentialsRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	keys := h.loginRateLimitKeys(r, req.Username)
	if h.loginKeysRateLimited(w, keys) {
		return
	}

	user, err := h.auth.Authenticate(req.Username, req.Password)
	if err != nil {
		if h.limiter != nil {
			h.limiter.Record(keys...)
		}
		httputil.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
		return
	}
	if h.limiter != nil {
		h.limiter.Reset(keys...)
	}

	if err := h.WriteSession(w, r, user.ID); err != nil {
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (h *Handler) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if h.auth != nil && h.auth.Enabled() {
		if cookie, err := r.Cookie(store.SessionCookieName()); err == nil {
			_ = h.auth.DeleteSession(cookie.Value)
		}
	}
	clearSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleListSessions(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil || !h.auth.Enabled() {
		httputil.WriteError(w, http.StatusNotFound, "auth_disabled", "auth disabled")
		return
	}
	userID := apishared.UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	sessions, err := h.auth.ListSessions(userID, apishared.SessionTokenFromRequest(r))
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "session_list_failed", "failed to list sessions")
		return
	}
	out := make([]map[string]any, 0, len(sessions))
	for _, session := range sessions {
		out = append(out, map[string]any{
			"id":        session.ID,
			"expiresAt": session.ExpiresAt.UTC().Format(time.RFC3339),
			"current":   session.Current,
		})
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"sessions": out})
}

func (h *Handler) handleRevokeSession(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil || !h.auth.Enabled() {
		httputil.WriteError(w, http.StatusNotFound, "auth_disabled", "auth disabled")
		return
	}
	userID := apishared.UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	sessionID := r.PathValue("id")
	if err := h.auth.RevokeSession(userID, sessionID, apishared.SessionTokenFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "session not found")
			return
		}
		httputil.WriteError(w, http.StatusBadRequest, "revoke_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (h *Handler) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil || !h.auth.Enabled() {
		httputil.WriteError(w, http.StatusNotFound, "auth_disabled", "auth disabled")
		return
	}
	userID := apishared.UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req changePasswordRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := h.auth.ChangePassword(userID, req.CurrentPassword, req.NewPassword); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "change_password_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type changeUsernameRequest struct {
	Username string `json:"username"`
}

func (h *Handler) handleChangeUsername(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil || !h.auth.Enabled() {
		httputil.WriteError(w, http.StatusNotFound, "auth_disabled", "auth disabled")
		return
	}
	userID := apishared.UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req changeUsernameRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	user, err := h.auth.UpdateUsername(userID, req.Username)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "change_username_failed", err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"id":       user.ID,
		"username": user.Username,
	})
}

type deleteAccountRequest struct {
	Password string `json:"password"`
}

func (h *Handler) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil || !h.auth.Enabled() {
		httputil.WriteError(w, http.StatusNotFound, "auth_disabled", "auth disabled")
		return
	}
	userID := apishared.UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req deleteAccountRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := h.auth.DeleteUser(userID, req.Password); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "delete_account_failed", err.Error())
		return
	}
	if cookie, err := r.Cookie(store.SessionCookieName()); err == nil {
		_ = h.auth.DeleteSession(cookie.Value)
	}
	clearSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) WriteSession(w http.ResponseWriter, r *http.Request, userID string) error {
	token, expires, err := h.auth.CreateSession(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create session")
		return err
	}
	apishared.SetHTTPOnlyCookie(w, r, store.SessionCookieName(), token, "/", int(time.Until(expires).Seconds()), expires)
	return nil
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	apishared.ClearHTTPOnlyCookie(w, r, store.SessionCookieName(), "/")
}

// loginRateLimitKeys scopes throttling to the client address and, when known,
// the target account. A spoofed X-Forwarded-For can rotate the IP bucket but
// the username bucket still bounds guessing against a single account.
func (h *Handler) loginRateLimitKeys(r *http.Request, username string) []string {
	keys := []string{"login|ip|unknown"}
	if addr, ok := apishared.ClientIP(r, h.cfg.TrustProxy); ok {
		keys[0] = "login|ip|" + addr.String()
	}
	if name := strings.TrimSpace(username); name != "" {
		keys = append(keys, "login|user|"+name)
	}
	return keys
}

func (h *Handler) loginKeysRateLimited(w http.ResponseWriter, keys []string) bool {
	if h.limiter == nil || !h.limiter.Blocked(keys...) {
		return false
	}
	apishared.WriteRateLimited(w, h.limiter.RetryAfterSeconds(keys...))
	return true
}

// handleGetSubsonicKey returns the caller's Subsonic API key so it can be
// pasted into clients that use token auth or the "p" parameter.
func (h *Handler) handleGetSubsonicKey(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil || !h.auth.Enabled() {
		httputil.WriteError(w, http.StatusNotFound, "auth_disabled", "auth disabled")
		return
	}
	userID := apishared.UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	secret, err := h.auth.SubsonicAPISecretForUser(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "no_key", "no subsonic api key configured")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"apiKey":  secret,
		"restUrl": "/rest",
	})
}

// handleRotateSubsonicKey replaces the API key and returns the new value.
func (h *Handler) handleRotateSubsonicKey(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil || !h.auth.Enabled() {
		httputil.WriteError(w, http.StatusNotFound, "auth_disabled", "auth disabled")
		return
	}
	userID := apishared.UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	secret, err := h.auth.RegenerateSubsonicAPISecret(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "rotate_failed", "failed to rotate api key")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"apiKey":  secret,
		"restUrl": "/rest",
	})
}
