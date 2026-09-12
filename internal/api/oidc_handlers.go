// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"melovian/internal/httputil"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	oidcStateCookie    = "melovian_oidc_state"
	oidcVerifierCookie = "melovian_oidc_verifier"
)

type oidcRuntime struct {
	provider *oidc.Provider
	oauth2   *oauth2.Config
}

func (s *Server) registerOIDCRoutes() {
	if !s.cfg.OIDCEnabled() {
		return
	}
	s.mux.HandleFunc("GET /api/auth/oidc/login", s.handleOIDCLogin)
	s.mux.HandleFunc("GET /api/auth/oidc/callback", s.handleOIDCCallback)
}

func (s *Server) oidcRuntime() (*oidcRuntime, error) {
	s.oidcOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		provider, err := oidc.NewProvider(ctx, s.cfg.OIDC.Issuer)
		if err != nil {
			s.oidcInitErr = fmt.Errorf("discover oidc provider: %w", err)
			return
		}

		redirectURL := s.cfg.OIDC.RedirectURL
		if redirectURL == "" && s.cfg.PublicURL != "" {
			redirectURL = s.cfg.PublicURL + "/api/auth/oidc/callback"
		}
		if redirectURL == "" {
			s.oidcInitErr = fmt.Errorf("oidc redirect url is required")
			return
		}

		s.oidcRuntimeValue = &oidcRuntime{
			provider: provider,
			oauth2: &oauth2.Config{
				ClientID:     s.cfg.OIDC.ClientID,
				ClientSecret: s.cfg.OIDC.ClientSecret,
				RedirectURL:  redirectURL,
				Endpoint:     provider.Endpoint(),
				Scopes:       s.cfg.OIDC.Scopes,
			},
		}
	})
	if s.oidcInitErr != nil {
		return nil, s.oidcInitErr
	}
	return s.oidcRuntimeValue, nil
}

func (s *Server) handleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil || !s.auth.Enabled() || !s.cfg.OIDCEnabled() {
		httputil.WriteError(w, http.StatusNotFound, "oidc_disabled", "oidc disabled")
		return
	}

	runtime, err := s.oidcRuntime()
	if err != nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "service_unavailable", "oidc unavailable")
		return
	}

	state, err := randomOIDCState()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to start oidc login")
		return
	}
	verifier := oauth2.GenerateVerifier()

	setHTTPOnlyCookie(w, r, oidcStateCookie, state, "/api/auth/oidc", 600, time.Time{})
	setHTTPOnlyCookie(w, r, oidcVerifierCookie, verifier, "/api/auth/oidc", 600, time.Time{})

	authURL := runtime.oauth2.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (s *Server) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil || !s.auth.Enabled() || !s.cfg.OIDCEnabled() {
		httputil.WriteError(w, http.StatusNotFound, "oidc_disabled", "oidc disabled")
		return
	}

	runtime, err := s.oidcRuntime()
	if err != nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "service_unavailable", "oidc unavailable")
		return
	}

	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		httputil.WriteError(w, http.StatusBadRequest, "oidc_error", errMsg)
		return
	}

	stateCookie, err := r.Cookie(oidcStateCookie)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_oidc_state", "invalid oidc state")
		return
	}
	verifierCookie, err := r.Cookie(oidcVerifierCookie)
	if err != nil || verifierCookie.Value == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing_oidc_verifier", "missing oidc verifier")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing_authorization_code", "missing authorization code")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	token, err := runtime.oauth2.Exchange(ctx, code, oauth2.VerifierOption(verifierCookie.Value))
	if err != nil {
		httputil.WriteError(w, http.StatusBadGateway, "bad_gateway", "oidc exchange failed")
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		httputil.WriteError(w, http.StatusBadGateway, "bad_gateway", "missing id token")
		return
	}

	verifier := runtime.provider.Verifier(&oidc.Config{ClientID: s.cfg.OIDC.ClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "invalid_id_token", "invalid id token")
		return
	}

	var claims struct {
		Subject           string `json:"sub"`
		PreferredUsername string `json:"preferred_username"`
		Email             string `json:"email"`
		Name              string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		httputil.WriteError(w, http.StatusBadGateway, "bad_gateway", "invalid id token claims")
		return
	}

	username := claims.PreferredUsername
	if username == "" {
		username = claims.Email
	}
	if username == "" {
		username = claims.Name
	}

	user, err := s.auth.FindOrCreateOIDCUser(idToken.Issuer, claims.Subject, username)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to provision user")
		return
	}

	clearOIDCCookies(w, r)
	if err := s.writeSession(w, r, user.ID); err != nil {
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func clearOIDCCookies(w http.ResponseWriter, r *http.Request) {
	for _, name := range []string{oidcStateCookie, oidcVerifierCookie} {
		clearHTTPOnlyCookie(w, r, name, "/api/auth/oidc")
	}
}

func randomOIDCState() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
