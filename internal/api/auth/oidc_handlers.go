// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"melovian/internal/api/apishared"
	"melovian/internal/consts"
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

func (h *Handler) registerOIDCRoutes(mux *http.ServeMux) {
	if !h.cfg.OIDCEnabled() {
		return
	}
	mux.HandleFunc("GET /api/auth/oidc/login", h.handleOIDCLogin)
	mux.HandleFunc("GET /api/auth/oidc/callback", h.handleOIDCCallback)
}

func (h *Handler) oidcRuntime() (*oidcRuntime, error) {
	h.oidcOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), consts.UpstreamTimeout)
		defer cancel()

		provider, err := oidc.NewProvider(ctx, h.cfg.OIDC.Issuer)
		if err != nil {
			h.oidcInitErr = fmt.Errorf("discover oidc provider: %w", err)
			return
		}

		redirectURL := h.cfg.OIDC.RedirectURL
		if redirectURL == "" && h.cfg.PublicURL != "" {
			redirectURL = h.cfg.PublicURL + "/api/auth/oidc/callback"
		}
		if redirectURL == "" {
			h.oidcInitErr = fmt.Errorf("oidc redirect url is required")
			return
		}

		h.oidcRuntimeValue = &oidcRuntime{
			provider: provider,
			oauth2: &oauth2.Config{
				ClientID:     h.cfg.OIDC.ClientID,
				ClientSecret: h.cfg.OIDC.ClientSecret,
				RedirectURL:  redirectURL,
				Endpoint:     provider.Endpoint(),
				Scopes:       h.cfg.OIDC.Scopes,
			},
		}
	})
	if h.oidcInitErr != nil {
		return nil, h.oidcInitErr
	}
	return h.oidcRuntimeValue, nil
}

func (h *Handler) handleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil || !h.auth.Enabled() || !h.cfg.OIDCEnabled() {
		httputil.WriteError(w, http.StatusNotFound, "oidc_disabled", "oidc disabled")
		return
	}

	runtime, err := h.oidcRuntime()
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

	apishared.SetHTTPOnlyCookie(w, r, oidcStateCookie, state, "/api/auth/oidc", 600, time.Time{})
	apishared.SetHTTPOnlyCookie(w, r, oidcVerifierCookie, verifier, "/api/auth/oidc", 600, time.Time{})

	authURL := runtime.oauth2.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (h *Handler) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil || !h.auth.Enabled() || !h.cfg.OIDCEnabled() {
		httputil.WriteError(w, http.StatusNotFound, "oidc_disabled", "oidc disabled")
		return
	}

	runtime, err := h.oidcRuntime()
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

	ctx, cancel := context.WithTimeout(r.Context(), consts.UpstreamTimeout)
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

	verifier := runtime.provider.Verifier(&oidc.Config{ClientID: h.cfg.OIDC.ClientID})
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

	user, err := h.auth.FindOrCreateOIDCUser(idToken.Issuer, claims.Subject, username)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to provision user")
		return
	}

	clearOIDCCookies(w, r)
	if err := h.WriteSession(w, r, user.ID); err != nil {
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func clearOIDCCookies(w http.ResponseWriter, r *http.Request) {
	for _, name := range []string{oidcStateCookie, oidcVerifierCookie} {
		apishared.ClearHTTPOnlyCookie(w, r, name, "/api/auth/oidc")
	}
}

func randomOIDCState() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
