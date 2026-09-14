// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"melovian/internal/api/apishared"
	"melovian/internal/consts"
	"melovian/internal/httputil"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	oidcStateCookie    = "melovian_oidc_state"
	oidcVerifierCookie = "melovian_oidc_verifier"
	// oidcInitRetryCooldown bounds how long a failed provider init is
	// cached so a transient discovery outage does not disable SSO for the
	// process lifetime.
	oidcInitRetryCooldown = 60 * time.Second
	// oidcUserinfoMaxBody caps the userinfo response the decoder accepts.
	oidcUserinfoMaxBody = 1 << 20
)

type oidcRuntime struct {
	provider *oidc.Provider
	oauth2   *oauth2.Config
	// issuer identifies the provider when provisioning users. For generic
	// OAuth2 providers it is a synthetic value derived from the userinfo
	// endpoint so identities do not collide across providers.
	issuer string
}

func (h *Handler) registerOIDCRoutes(mux *http.ServeMux) {
	if !h.cfg.OIDCEnabled() {
		return
	}
	mux.HandleFunc("GET /api/auth/oidc/login", h.handleOIDCLogin)
	mux.HandleFunc("GET /api/auth/oidc/callback", h.handleOIDCCallback)
}

func (h *Handler) oidcRuntime() (*oidcRuntime, error) {
	h.oidcMu.Lock()
	defer h.oidcMu.Unlock()
	if h.oidcRuntimeValue != nil {
		return h.oidcRuntimeValue, nil
	}
	if h.oidcInitErr != nil && time.Since(h.oidcInitErrAt) < oidcInitRetryCooldown {
		return nil, h.oidcInitErr
	}
	runtime, err := h.buildOIDCRuntime()
	if err != nil {
		h.oidcInitErr = err
		h.oidcInitErrAt = time.Now()
		return nil, err
	}
	h.oidcRuntimeValue = runtime
	h.oidcInitErr = nil
	return runtime, nil
}

// buildOIDCRuntime performs provider discovery or endpoint validation. It
// runs under oidcMu and its result is only cached on success.
func (h *Handler) buildOIDCRuntime() (*oidcRuntime, error) {
	redirectURL := h.cfg.OIDC.RedirectURL
	if redirectURL == "" && h.cfg.PublicURL != "" {
		redirectURL = h.cfg.PublicURL + "/api/auth/oidc/callback"
	}
	if redirectURL == "" {
		return nil, fmt.Errorf("oidc redirect url is required")
	}

	conf := &oauth2.Config{
		ClientID:     h.cfg.OIDC.ClientID,
		ClientSecret: h.cfg.OIDC.ClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       h.cfg.OIDC.Scopes,
	}

	issuer := strings.TrimSpace(h.cfg.OIDC.Issuer)
	if issuer == "" {
		return h.genericOAuth2Runtime(conf)
	}
	if _, err := checkOIDCEndpointURL(issuer); err != nil {
		return nil, fmt.Errorf("oidc issuer: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), consts.UpstreamTimeout)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("discover oidc provider: %w", err)
	}
	conf.Endpoint = provider.Endpoint()
	return &oidcRuntime{provider: provider, oauth2: conf}, nil
}

// genericOAuth2Runtime builds the flow for providers that expose plain
// OAuth2 endpoints but no OIDC discovery document. Identity then comes from
// the userinfo endpoint rather than a verified id_token.
func (h *Handler) genericOAuth2Runtime(conf *oauth2.Config) (*oidcRuntime, error) {
	endpoints := []struct {
		name string
		raw  string
	}{
		{"oidc auth url", h.cfg.OIDC.AuthURL},
		{"oidc token url", h.cfg.OIDC.TokenURL},
		{"oidc userinfo url", h.cfg.OIDC.UserInfoURL},
	}
	var userInfoURL *url.URL
	for _, ep := range endpoints {
		u, err := checkOIDCEndpointURL(ep.raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", ep.name, err)
		}
		if ep.name == "oidc userinfo url" {
			userInfoURL = u
		}
	}
	conf.Endpoint = oauth2.Endpoint{
		AuthURL:  h.cfg.OIDC.AuthURL,
		TokenURL: h.cfg.OIDC.TokenURL,
		// AuthStyleInParams sends the client credentials in the token
		// request body, which more providers accept than a basic auth
		// header.
		AuthStyle: oauth2.AuthStyleInParams,
	}
	return &oidcRuntime{
		oauth2: conf,
		issuer: "oauth2:" + userInfoURL.Host,
	}, nil
}

// checkOIDCEndpointURL requires provider endpoints to use https. Plain http
// is allowed only for loopback hosts so local development still works. It
// returns the parsed URL so callers can reuse it.
func checkOIDCEndpointURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid endpoint url")
	}
	if u.Scheme == "https" {
		return u, nil
	}
	if u.Scheme == "http" && isLoopbackHost(u.Hostname()) {
		return u, nil
	}
	return nil, fmt.Errorf("endpoint must use https")
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	addr, err := netip.ParseAddr(host)
	return err == nil && addr.IsLoopback()
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

	var issuer, subject, username string
	if runtime.provider != nil {
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

		issuer = idToken.Issuer
		subject = claims.Subject
		username = claims.PreferredUsername
		if username == "" {
			username = claims.Email
		}
		if username == "" {
			username = claims.Name
		}
	} else {
		issuer = runtime.issuer
		subject, username, err = h.fetchOAuth2UserInfo(ctx, runtime, token)
		if err != nil {
			httputil.WriteError(w, http.StatusBadGateway, "bad_gateway", "oauth2 userinfo lookup failed")
			return
		}
	}

	user, err := h.auth.FindOrCreateOIDCUser(issuer, subject, username)
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

// fetchOAuth2UserInfo loads the user profile from a generic OAuth2 userinfo
// endpoint using the access token. The subject falls back from "sub" to
// "id", and the display name walks the common username claims before
// settling on the subject itself.
func (h *Handler) fetchOAuth2UserInfo(ctx context.Context, runtime *oidcRuntime, token *oauth2.Token) (string, string, error) {
	client := runtime.oauth2.Client(ctx, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSpace(h.cfg.OIDC.UserInfoURL), nil)
	if err != nil {
		return "", "", fmt.Errorf("build userinfo request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("userinfo request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("userinfo returned status %d", resp.StatusCode)
	}

	var info struct {
		Sub               string          `json:"sub"`
		ID                json.RawMessage `json:"id"`
		PreferredUsername string          `json:"preferred_username"`
		Username          string          `json:"username"`
		Email             string          `json:"email"`
		Name              string          `json:"name"`
	}
	body, err := httputil.ReadLimited(resp.Body, oidcUserinfoMaxBody)
	if err != nil {
		return "", "", fmt.Errorf("read userinfo response: %w", err)
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return "", "", fmt.Errorf("decode userinfo response: %w", err)
	}

	subject := strings.TrimSpace(info.Sub)
	if subject == "" {
		subject = userinfoIDString(info.ID)
	}
	if subject == "" {
		return "", "", fmt.Errorf("userinfo response has no subject")
	}
	username := strings.TrimSpace(info.PreferredUsername)
	for _, candidate := range []string{info.Username, info.Email, info.Name} {
		if username != "" {
			break
		}
		username = strings.TrimSpace(candidate)
	}
	if username == "" {
		username = subject
	}
	return subject, username, nil
}

// userinfoIDString accepts both string and numeric "id" claims since some
// providers return the subject as a JSON number.
func userinfoIDString(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err == nil {
		return n.String()
	}
	return ""
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
