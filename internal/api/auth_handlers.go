// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"maps"
	"net/http"
	"strings"
	"time"

	"database/sql"
	"errors"

	"melovian/internal/compat"
	"melovian/internal/democatalog"
	"melovian/internal/httputil"
	"melovian/internal/store"
)

func (s *Server) registerAuthRoutes() {
	s.mux.HandleFunc("GET /api/auth/status", s.handleAuthStatus)
	s.mux.HandleFunc("POST /api/auth/setup", s.handleAuthSetup)
	s.mux.HandleFunc("POST /api/auth/login", s.handleAuthLogin)
	s.mux.HandleFunc("POST /api/auth/logout", s.handleAuthLogout)
	s.mux.HandleFunc("GET /api/auth/sessions", s.handleListSessions)
	s.mux.HandleFunc("DELETE /api/auth/sessions/{id}", s.handleRevokeSession)
	s.mux.HandleFunc("POST /api/auth/change-password", s.handleChangePassword)
	s.mux.HandleFunc("POST /api/auth/username", s.handleChangeUsername)
	s.mux.HandleFunc("POST /api/auth/delete-account", s.handleDeleteAccount)
	s.mux.HandleFunc("GET /health", s.handleHealth)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	payload := map[string]any{"status": "ok"}
	maps.Copy(payload, compat.ConfigFields())
	httputil.WriteJSON(w, http.StatusOK, payload)
}

func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	demo := s.cfg.DemoModeEffective()
	enabled := !demo && s.auth != nil && s.auth.Enabled()
	payload := map[string]any{
		"enabled":       enabled,
		"authenticated": false,
		"setupRequired": false,
		"oidcEnabled":   s.cfg.OIDCEnabled(),
		"oidcLoginUrl":  "/api/auth/oidc/login",
		"demoMode":      demo,
		"fakeCatalog":   democatalog.UsesFakeCatalog(s.cfg),
	}
	if !enabled {
		httputil.WriteJSON(w, http.StatusOK, payload)
		return
	}

	count, err := s.auth.CountUsers()
	if err != nil {
		http.Error(w, "failed to read auth status", http.StatusInternalServerError)
		return
	}
	payload["setupRequired"] = count == 0 && !s.cfg.OIDCEnabled()
	payload["localLoginEnabled"] = !s.cfg.OIDCEnabled() || count > 0

	userID := UserIDFromContext(r.Context())
	if userID == "" {
		if token := sessionTokenFromRequest(r); token != "" {
			userID, _ = s.auth.UserIDFromToken(token)
		}
	}
	if userID == "" {
		httputil.WriteJSON(w, http.StatusOK, payload)
		return
	}

	user, err := s.auth.GetUser(userID)
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

func (s *Server) handleAuthSetup(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil || !s.auth.Enabled() {
		http.Error(w, "auth disabled", http.StatusNotFound)
		return
	}
	if s.cfg.OIDCEnabled() {
		http.Error(w, "local account setup disabled when oidc is enabled", http.StatusForbidden)
		return
	}

	count, err := s.auth.CountUsers()
	if err != nil {
		http.Error(w, "failed to read users", http.StatusInternalServerError)
		return
	}
	if count > 0 {
		http.Error(w, "setup already completed", http.StatusConflict)
		return
	}

	var req authCredentialsRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := s.auth.CreateUser(req.Username, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.writeSession(w, r, user.ID); err != nil {
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, map[string]any{
		"user": map[string]any{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil || !s.auth.Enabled() {
		http.Error(w, "auth disabled", http.StatusNotFound)
		return
	}

	var req authCredentialsRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := s.auth.Authenticate(req.Username, req.Password)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := s.writeSession(w, r, user.ID); err != nil {
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if s.auth != nil && s.auth.Enabled() {
		if cookie, err := r.Cookie(store.SessionCookieName()); err == nil {
			_ = s.auth.DeleteSession(cookie.Value)
		}
	}
	clearSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil || !s.auth.Enabled() {
		httputil.WriteError(w, http.StatusNotFound, "auth_disabled", "auth disabled")
		return
	}
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	sessions, err := s.auth.ListSessions(userID, sessionTokenFromRequest(r))
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

func (s *Server) handleRevokeSession(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil || !s.auth.Enabled() {
		httputil.WriteError(w, http.StatusNotFound, "auth_disabled", "auth disabled")
		return
	}
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	sessionID := r.PathValue("id")
	if err := s.auth.RevokeSession(userID, sessionID, sessionTokenFromRequest(r)); err != nil {
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

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil || !s.auth.Enabled() {
		httputil.WriteError(w, http.StatusNotFound, "auth_disabled", "auth disabled")
		return
	}
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req changePasswordRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := s.auth.ChangePassword(userID, req.CurrentPassword, req.NewPassword); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "change_password_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type changeUsernameRequest struct {
	Username string `json:"username"`
}

func (s *Server) handleChangeUsername(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil || !s.auth.Enabled() {
		httputil.WriteError(w, http.StatusNotFound, "auth_disabled", "auth disabled")
		return
	}
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req changeUsernameRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	user, err := s.auth.UpdateUsername(userID, req.Username)
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

func (s *Server) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil || !s.auth.Enabled() {
		httputil.WriteError(w, http.StatusNotFound, "auth_disabled", "auth disabled")
		return
	}
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	var req deleteAccountRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := s.auth.DeleteUser(userID, req.Password); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "delete_account_failed", err.Error())
		return
	}
	if cookie, err := r.Cookie(store.SessionCookieName()); err == nil {
		_ = s.auth.DeleteSession(cookie.Value)
	}
	clearSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) writeSession(w http.ResponseWriter, r *http.Request, userID string) error {
	token, expires, err := s.auth.CreateSession(userID)
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return err
	}
	setHTTPOnlyCookie(w, r, store.SessionCookieName(), token, "/", int(time.Until(expires).Seconds()), expires)
	return nil
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	clearHTTPOnlyCookie(w, r, store.SessionCookieName(), "/")
}

func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if hdr := r.Header.Get("X-Forwarded-Proto"); strings.EqualFold(hdr, "https") {
		return true
	}
	return false
}

func sessionTokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(store.SessionCookieName())
	if err != nil {
		return ""
	}
	return cookie.Value
}

func isPublicAPIPath(path string) bool {
	switch path {
	case "/health", "/metrics", "/api/client-log", "/api/ws", "/api/config", "/api/auth/status", "/api/auth/setup", "/api/auth/login", "/api/auth/logout", "/api/auth/oidc/login", "/api/auth/oidc/callback":
		return true
	default:
		return false
	}
}
