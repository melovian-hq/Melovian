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
