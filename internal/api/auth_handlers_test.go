// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"melovian/internal/appconfig"
	"melovian/internal/store"
)

func newAuthTestServer(t *testing.T) (*Server, *store.DB) {
	t.Helper()
	dir := t.TempDir()
	cfg := appconfig.Config{
		DataDir:      dir,
		DatabasePath: filepath.Join(dir, "test.db"),
		ListenAddr:   "127.0.0.1:0",
		CacheEnabled: false,
		AuthSecret:   "test-auth-secret",
	}
	db, err := store.OpenDB(cfg.DatabasePath, cfg)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewServer(cfg, db), db
}

func newOIDCAuthTestServer(t *testing.T) (*Server, *store.DB) {
	t.Helper()
	dir := t.TempDir()
	cfg := appconfig.Config{
		DataDir:      dir,
		DatabasePath: filepath.Join(dir, "test.db"),
		ListenAddr:   "127.0.0.1:0",
		CacheEnabled: false,
		AuthSecret:   "test-auth-secret",
		OIDC: appconfig.OIDCConfig{
			Issuer:       "https://issuer.example",
			ClientID:     "client-id",
			RedirectURL:  "http://localhost/api/auth/oidc/callback",
			ProviderName: "Pocket ID",
		},
	}
	db, err := store.OpenDB(cfg.DatabasePath, cfg)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewServer(cfg, db), db
}

// oidcUserCookie provisions a passwordless user and returns a valid session
// cookie for it.
func oidcUserCookie(t *testing.T, db *store.DB, username string) *http.Cookie {
	t.Helper()
	authStore := store.NewAuthStore(db, "test-auth-secret")
	user, err := authStore.FindOrCreateOIDCUser("https://issuer.example", "subject-"+username, username)
	if err != nil {
		t.Fatalf("FindOrCreateOIDCUser: %v", err)
	}
	token, _, err := authStore.CreateSession(user.ID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	return &http.Cookie{Name: store.SessionCookieName(), Value: token}
}

func newDemoTestServer(t *testing.T) (*Server, *store.DB) {
	t.Helper()
	dir := t.TempDir()
	cfg := appconfig.Config{
		DataDir:      dir,
		DatabasePath: filepath.Join(dir, "test.db"),
		ListenAddr:   "127.0.0.1:0",
		CacheEnabled: false,
		AuthSecret:   "test-auth-secret",
		DemoMode:     true,
		ServerMode:   true,
	}
	db, err := store.OpenDB(cfg.DatabasePath, cfg)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewServer(cfg, db), db
}

func authCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == store.SessionCookieName() {
			return c
		}
	}
	return nil
}

func TestAuthSetupLoginLogoutFlow(t *testing.T) {
	srv, _ := newAuthTestServer(t)
	handler := srv.Handler()

	statusReq := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	statusRec := httptest.NewRecorder()
	handler.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("status %d", statusRec.Code)
	}
	var status map[string]any
	if err := json.Unmarshal(statusRec.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status["enabled"] != true || status["setupRequired"] != true || status["authenticated"] != false {
		t.Fatalf("unexpected initial status: %+v", status)
	}

	setupBody := bytes.NewBufferString(`{"username":"admin","password":"password123"}`)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/auth/setup", setupBody)
	setupRec := httptest.NewRecorder()
	handler.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup status %d: %s", setupRec.Code, setupRec.Body.String())
	}
	setupCookie := authCookie(setupRec)
	if setupCookie == nil || setupCookie.Value == "" {
		t.Fatal("expected session cookie after setup")
	}

	authReq := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	authReq.AddCookie(setupCookie)
	authRec := httptest.NewRecorder()
	handler.ServeHTTP(authRec, authReq)
	if err := json.Unmarshal(authRec.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode auth status: %v", err)
	}
	if status["authenticated"] != true {
		t.Fatalf("expected authenticated after setup, got %+v", status)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutReq.AddCookie(setupCookie)
	logoutRec := httptest.NewRecorder()
	handler.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout status %d", logoutRec.Code)
	}

	loginBody := bytes.NewBufferString(`{"username":"admin","password":"password123"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", loginBody)
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status %d: %s", loginRec.Code, loginRec.Body.String())
	}
	loginCookie := authCookie(loginRec)
	if loginCookie == nil {
		t.Fatal("expected session cookie after login")
	}

	protectedReq := httptest.NewRequest(http.MethodGet, "/api/instances", nil)
	protectedReq.AddCookie(loginCookie)
	protectedRec := httptest.NewRecorder()
	handler.ServeHTTP(protectedRec, protectedReq)
	if protectedRec.Code != http.StatusOK {
		t.Fatalf("protected route status %d: %s", protectedRec.Code, protectedRec.Body.String())
	}
}

func TestAuthLoginInvalidCredentials(t *testing.T) {
	srv, _ := newAuthTestServer(t)
	handler := srv.Handler()

	setupBody := bytes.NewBufferString(`{"username":"admin","password":"password123"}`)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/auth/setup", setupBody)
	setupRec := httptest.NewRecorder()
	handler.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup status %d", setupRec.Code)
	}

	loginBody := bytes.NewBufferString(`{"username":"admin","password":"wrong-password"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", loginBody)
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", loginRec.Code)
	}
}

func TestAuthMiddlewareBlocksProtectedRoutes(t *testing.T) {
	srv, _ := newAuthTestServer(t)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/instances", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", rec.Code)
	}
}

func TestAuthPublicPathsBypassMiddleware(t *testing.T) {
	srv, _ := newAuthTestServer(t)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health status %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("auth status %d", rec.Code)
	}
}

func TestSessionCookieSecureBehindProxy(t *testing.T) {
	srv, _ := newAuthTestServer(t)
	handler := srv.Handler()

	setupBody := bytes.NewBufferString(`{"username":"admin","password":"password123"}`)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/auth/setup", setupBody)
	setupReq.Header.Set("X-Forwarded-Proto", "https")
	setupRec := httptest.NewRecorder()
	handler.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup status %d", setupRec.Code)
	}

	cookie := authCookie(setupRec)
	if cookie == nil || !cookie.Secure {
		t.Fatal("expected secure session cookie behind https proxy")
	}
}

func TestAuthEnabledConfigIsPublic(t *testing.T) {
	srv, _ := newAuthTestServer(t)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected public /api/config with auth enabled, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/extensions", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected /api/extensions to require auth, got %d", rec.Code)
	}
}

func TestDemoModeAllowsReadWithoutSession(t *testing.T) {
	srv, _ := newDemoTestServer(t)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/instances", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 without session in demo mode, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected public /api/config in demo mode, got %d", rec.Code)
	}
}

func TestDemoModeBlocksMutations(t *testing.T) {
	srv, _ := newDemoTestServer(t)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodPost, "/api/instances", bytes.NewBufferString(`{"name":"x","serverUrl":"https://x","username":"u","password":"p"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for POST in demo mode, got %d", rec.Code)
	}
}

func TestDemoModeAuthStatus(t *testing.T) {
	srv, _ := newDemoTestServer(t)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("auth status %d", rec.Code)
	}
	var status map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status["demoMode"] != true || status["enabled"] != false {
		t.Fatalf("unexpected demo auth status: %+v", status)
	}
}

func TestAuthStatusReportsProviderNameAndHasPassword(t *testing.T) {
	srv, db := newOIDCAuthTestServer(t)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("auth status %d", rec.Code)
	}
	var status map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status["oidcEnabled"] != true {
		t.Fatalf("expected oidcEnabled, got %+v", status)
	}
	if status["oidcProviderName"] != "Pocket ID" {
		t.Fatalf("expected provider name, got %+v", status)
	}

	cookie := oidcUserCookie(t, db, "oidcuser")
	authReq := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	authReq.AddCookie(cookie)
	authRec := httptest.NewRecorder()
	handler.ServeHTTP(authRec, authReq)
	if err := json.Unmarshal(authRec.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode authed status: %v", err)
	}
	user, ok := status["user"].(map[string]any)
	if !ok {
		t.Fatalf("expected user payload, got %+v", status)
	}
	if user["hasPassword"] != false {
		t.Fatalf("expected hasPassword false for oidc user, got %+v", user)
	}
}

func TestAuthStatusHasPasswordForLocalUser(t *testing.T) {
	srv, _ := newAuthTestServer(t)
	handler := srv.Handler()

	setupBody := bytes.NewBufferString(`{"username":"admin","password":"password123"}`)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/auth/setup", setupBody)
	setupRec := httptest.NewRecorder()
	handler.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup status %d", setupRec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	req.AddCookie(authCookie(setupRec))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var status map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	user, ok := status["user"].(map[string]any)
	if !ok || user["hasPassword"] != true {
		t.Fatalf("expected hasPassword true for local user, got %+v", status)
	}
}

func TestSubsonicKeyLazyGeneratedForOIDCUser(t *testing.T) {
	srv, db := newAuthTestServer(t)
	handler := srv.Handler()

	cookie := oidcUserCookie(t, db, "oidcuser")
	req := httptest.NewRequest(http.MethodPost, "/api/auth/subsonic-key", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for lazy key, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode key payload: %v", err)
	}
	key, _ := payload["apiKey"].(string)
	if len(key) != 64 {
		t.Fatalf("expected generated api key, got %+v", payload)
	}

	again := httptest.NewRequest(http.MethodPost, "/api/auth/subsonic-key", nil)
	again.AddCookie(cookie)
	againRec := httptest.NewRecorder()
	handler.ServeHTTP(againRec, again)
	var second map[string]any
	if err := json.Unmarshal(againRec.Body.Bytes(), &second); err != nil {
		t.Fatalf("decode second key payload: %v", err)
	}
	if second["apiKey"] != key {
		t.Fatal("expected the same key on repeat reads")
	}
}

func TestDeleteAccountRequiresConfirmForPasswordlessUser(t *testing.T) {
	srv, db := newAuthTestServer(t)
	handler := srv.Handler()

	cookie := oidcUserCookie(t, db, "oidcuser")

	denyReq := httptest.NewRequest(http.MethodPost, "/api/auth/delete-account", bytes.NewBufferString(`{}`))
	denyReq.AddCookie(cookie)
	denyRec := httptest.NewRecorder()
	handler.ServeHTTP(denyRec, denyReq)
	if denyRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without confirm, got %d: %s", denyRec.Code, denyRec.Body.String())
	}

	confirmReq := httptest.NewRequest(http.MethodPost, "/api/auth/delete-account", bytes.NewBufferString(`{"confirm":true}`))
	confirmReq.AddCookie(cookie)
	confirmRec := httptest.NewRecorder()
	handler.ServeHTTP(confirmRec, confirmReq)
	if confirmRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 with confirm, got %d: %s", confirmRec.Code, confirmRec.Body.String())
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	statusReq.AddCookie(cookie)
	statusRec := httptest.NewRecorder()
	handler.ServeHTTP(statusRec, statusReq)
	var status map[string]any
	if err := json.Unmarshal(statusRec.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status["authenticated"] != false {
		t.Fatalf("expected session cleared after delete, got %+v", status)
	}
}

func TestDeleteAccountLocalUserStillNeedsPassword(t *testing.T) {
	srv, _ := newAuthTestServer(t)
	handler := srv.Handler()

	setupBody := bytes.NewBufferString(`{"username":"admin","password":"password123"}`)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/auth/setup", setupBody)
	setupRec := httptest.NewRecorder()
	handler.ServeHTTP(setupRec, setupReq)
	if setupRec.Code != http.StatusCreated {
		t.Fatalf("setup status %d", setupRec.Code)
	}
	cookie := authCookie(setupRec)

	wrongReq := httptest.NewRequest(http.MethodPost, "/api/auth/delete-account", bytes.NewBufferString(`{"password":"wrong","confirm":true}`))
	wrongReq.AddCookie(cookie)
	wrongRec := httptest.NewRecorder()
	handler.ServeHTTP(wrongRec, wrongReq)
	if wrongRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for wrong password, got %d", wrongRec.Code)
	}

	okReq := httptest.NewRequest(http.MethodPost, "/api/auth/delete-account", bytes.NewBufferString(`{"password":"password123"}`))
	okReq.AddCookie(cookie)
	okRec := httptest.NewRecorder()
	handler.ServeHTTP(okRec, okReq)
	if okRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for correct password, got %d: %s", okRec.Code, okRec.Body.String())
	}
}
