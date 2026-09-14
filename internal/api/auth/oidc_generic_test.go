// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"melovian/internal/appconfig"
	"melovian/internal/store"
)

// newGenericOAuth2Server returns an httptest server that plays the role of
// a plain OAuth2 provider: a token endpoint that always issues the same
// access token and a userinfo endpoint backed by the given handler.
func newGenericOAuth2Server(t *testing.T, userinfo http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("token request parse: %v", err)
		}
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Errorf("unexpected grant_type %q", r.Form.Get("grant_type"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-access-token",
			"token_type":   "Bearer",
		})
	})
	mux.Handle("/userinfo", userinfo)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func newGenericOAuth2Handler(t *testing.T, srv *httptest.Server) (*Handler, *store.AuthStore) {
	t.Helper()
	dir := t.TempDir()
	cfg := appconfig.Config{
		DataDir:      dir,
		DatabasePath: filepath.Join(dir, "test.db"),
		AuthSecret:   "test-secret",
		OIDC: appconfig.OIDCConfig{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			RedirectURL:  "http://localhost/api/auth/oidc/callback",
			Scopes:       []string{"profile"},
			AuthURL:      srv.URL + "/authorize",
			TokenURL:     srv.URL + "/token",
			UserInfoURL:  srv.URL + "/userinfo",
		},
	}
	db, err := store.OpenDB(cfg.DatabasePath, cfg)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	authStore := store.NewAuthStore(db, cfg.AuthSecret)
	return New(authStore, cfg, nil), authStore
}

func runGenericCallback(t *testing.T, h *Handler) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/callback?state=state-1&code=code-1", nil)
	req.AddCookie(&http.Cookie{Name: oidcStateCookie, Value: "state-1"})
	req.AddCookie(&http.Cookie{Name: oidcVerifierCookie, Value: "verifier-1"})
	rec := httptest.NewRecorder()
	h.handleOIDCCallback(rec, req)
	return rec
}

func genericIssuer(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse test server url: %v", err)
	}
	return "oauth2:" + u.Host
}

func sessionCookieValue(rec *httptest.ResponseRecorder) string {
	for _, c := range rec.Result().Cookies() {
		if c.Name == store.SessionCookieName() {
			return c.Value
		}
	}
	return ""
}

func TestGenericOAuth2CallbackProvisionsUser(t *testing.T) {
	var gotAuth string
	srv := newGenericOAuth2Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sub":                "subject-1",
			"preferred_username": "alice",
		})
	}))
	h, authStore := newGenericOAuth2Handler(t, srv)

	rec := runGenericCallback(t, h)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d: %s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Fatalf("expected redirect to /, got %q", loc)
	}
	if gotAuth != "Bearer test-access-token" {
		t.Fatalf("userinfo authorization header = %q", gotAuth)
	}
	if sessionCookieValue(rec) == "" {
		t.Fatal("expected session cookie after callback")
	}

	user, err := authStore.FindOrCreateOIDCUser(genericIssuer(t, srv), "subject-1", "")
	if err != nil {
		t.Fatalf("FindOrCreateOIDCUser: %v", err)
	}
	if user.Username != "alice" {
		t.Fatalf("expected provisioned username alice, got %q", user.Username)
	}
}

func TestGenericOAuth2UserinfoClaimFallbacks(t *testing.T) {
	cases := []struct {
		name         string
		body         map[string]any
		wantSubject  string
		wantUsername string
	}{
		{
			name:         "numeric id subject and username claim",
			body:         map[string]any{"id": 12345, "username": "bob"},
			wantSubject:  "12345",
			wantUsername: "bob",
		},
		{
			name:         "string id subject and email fallback",
			body:         map[string]any{"id": "id-9", "email": "carol@example.com"},
			wantSubject:  "id-9",
			wantUsername: "carol@example.com",
		},
		{
			name:         "name fallback before subject",
			body:         map[string]any{"sub": "subject-7", "name": "Dave"},
			wantSubject:  "subject-7",
			wantUsername: "Dave",
		},
		{
			name:         "username falls back to subject",
			body:         map[string]any{"sub": "subject-8"},
			wantSubject:  "subject-8",
			wantUsername: "subject-8",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newGenericOAuth2Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tc.body)
			}))
			h, authStore := newGenericOAuth2Handler(t, srv)

			rec := runGenericCallback(t, h)
			if rec.Code != http.StatusFound {
				t.Fatalf("expected 302, got %d: %s", rec.Code, rec.Body.String())
			}
			user, err := authStore.FindOrCreateOIDCUser(genericIssuer(t, srv), tc.wantSubject, "")
			if err != nil {
				t.Fatalf("FindOrCreateOIDCUser: %v", err)
			}
			if user.Username != tc.wantUsername {
				t.Fatalf("username = %q, want %q", user.Username, tc.wantUsername)
			}
		})
	}
}

func TestGenericOAuth2UserinfoMissingSubject(t *testing.T) {
	srv := newGenericOAuth2Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"name": "No Subject"})
	}))
	h, _ := newGenericOAuth2Handler(t, srv)

	rec := runGenericCallback(t, h)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGenericOAuth2UserinfoFailure(t *testing.T) {
	srv := newGenericOAuth2Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	h, _ := newGenericOAuth2Handler(t, srv)

	rec := runGenericCallback(t, h)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGenericOAuth2RejectsInsecureEndpoints(t *testing.T) {
	dir := t.TempDir()
	cfg := appconfig.Config{
		DataDir:      dir,
		DatabasePath: filepath.Join(dir, "test.db"),
		AuthSecret:   "test-secret",
		OIDC: appconfig.OIDCConfig{
			ClientID:    "client-id",
			RedirectURL: "http://localhost/api/auth/oidc/callback",
			AuthURL:     "http://idp.example.com/authorize",
			TokenURL:    "http://idp.example.com/token",
			UserInfoURL: "http://idp.example.com/userinfo",
		},
	}
	db, err := store.OpenDB(cfg.DatabasePath, cfg)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	h := New(store.NewAuthStore(db, cfg.AuthSecret), cfg, nil)

	if _, err := h.oidcRuntime(); err == nil {
		t.Fatal("expected error for plain http endpoints on a remote host")
	} else if !strings.Contains(err.Error(), "https") {
		t.Fatalf("expected https error, got %v", err)
	}
}

// newIssuerHandler builds a Handler whose OIDC config uses discovery with
// the given issuer URL.
func newIssuerHandler(t *testing.T, issuer string) *Handler {
	t.Helper()
	dir := t.TempDir()
	cfg := appconfig.Config{
		DataDir:      dir,
		DatabasePath: filepath.Join(dir, "test.db"),
		AuthSecret:   "test-secret",
		OIDC: appconfig.OIDCConfig{
			Issuer:      issuer,
			ClientID:    "client-id",
			RedirectURL: "http://localhost/api/auth/oidc/callback",
		},
	}
	db, err := store.OpenDB(cfg.DatabasePath, cfg)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return New(store.NewAuthStore(db, cfg.AuthSecret), cfg, nil)
}

func TestOIDCIssuerRequiresHTTPSOrLoopback(t *testing.T) {
	h := newIssuerHandler(t, "http://idp.example.com")

	if _, err := h.oidcRuntime(); err == nil {
		t.Fatal("expected error for plain http issuer on a remote host")
	} else if !strings.Contains(err.Error(), "https") {
		t.Fatalf("expected https error, got %v", err)
	}
}

func TestOIDCInitRetriesAfterCooldown(t *testing.T) {
	// The issuer fails validation, so init errors fast without network.
	h := newIssuerHandler(t, "http://idp.example.com")

	if _, err := h.oidcRuntime(); err == nil {
		t.Fatal("expected init error")
	}
	firstErrAt := h.oidcInitErrAt

	// Within the cooldown the cached error is returned without retrying.
	if _, err := h.oidcRuntime(); err == nil {
		t.Fatal("expected cached init error")
	}
	if !h.oidcInitErrAt.Equal(firstErrAt) {
		t.Fatal("init should not retry inside the cooldown window")
	}

	// After the cooldown the build runs again and the error is re-stamped.
	h.oidcInitErrAt = time.Now().Add(-2 * oidcInitRetryCooldown)
	if _, err := h.oidcRuntime(); err == nil {
		t.Fatal("expected init error after retry")
	}
	if !h.oidcInitErrAt.After(firstErrAt) {
		t.Fatal("expected init to retry after the cooldown")
	}
}

func TestGenericOAuth2LoginRedirectsToAuthURL(t *testing.T) {
	srv := newGenericOAuth2Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	h, _ := newGenericOAuth2Handler(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/login", nil)
	rec := httptest.NewRecorder()
	h.handleOIDCLogin(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d: %s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, srv.URL+"/authorize") {
		t.Fatalf("expected redirect to auth url, got %q", loc)
	}
	if !strings.Contains(loc, "client_id=client-id") {
		t.Fatalf("expected client_id in auth url, got %q", loc)
	}
	var stateCookie, verifierCookie bool
	for _, c := range rec.Result().Cookies() {
		switch c.Name {
		case oidcStateCookie:
			stateCookie = c.Value != ""
		case oidcVerifierCookie:
			verifierCookie = c.Value != ""
		}
	}
	if !stateCookie || !verifierCookie {
		t.Fatal("expected state and verifier cookies on login redirect")
	}
}

func TestGenericOAuth2CallbackRejectsBadState(t *testing.T) {
	srv := newGenericOAuth2Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	h, _ := newGenericOAuth2Handler(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/callback?state=wrong&code=code-1", nil)
	req.AddCookie(&http.Cookie{Name: oidcStateCookie, Value: "state-1"})
	req.AddCookie(&http.Cookie{Name: oidcVerifierCookie, Value: "verifier-1"})
	rec := httptest.NewRecorder()
	h.handleOIDCCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
