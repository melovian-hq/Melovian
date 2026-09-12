// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"crypto/md5" //#nosec G501 -- Subsonic token auth requires MD5 per protocol spec
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"melovian/internal/appconfig"
	"melovian/internal/store"
)

func newAuthedSubsonicServer(t *testing.T) (*Server, *store.DB) {
	t.Helper()
	dir := t.TempDir()
	cfg := appconfig.Config{
		DataDir:              dir,
		DatabasePath:         filepath.Join(dir, "test.db"),
		ListenAddr:           "127.0.0.1:0",
		CacheEnabled:         false,
		AuthSecret:           "test-auth-secret",
		ServerMode:           true,
		SubsonicServer:       true,
		LocalLibraryOverride: true,
		LocalLibrary: appconfig.LocalLibraryConfig{
			Enabled:         true,
			AllowCustomPath: true,
		},
	}
	db, err := store.OpenDB(cfg.DatabasePath, cfg)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewServer(cfg, db), db
}

func setupUser(t *testing.T, handler http.Handler, username, password string) *http.Cookie {
	t.Helper()
	body := bytes.NewBufferString(`{"username":"` + username + `","password":"` + password + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/setup", body)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("setup status %d: %s", rec.Code, rec.Body.String())
	}
	cookie := authCookie(rec)
	if cookie == nil {
		t.Fatal("expected session cookie after setup")
	}
	return cookie
}

func newDemoSubsonicTestServer(t *testing.T) (*Server, *store.DB) {
	t.Helper()
	dir := t.TempDir()
	cfg := appconfig.Config{
		DataDir:        dir,
		DatabasePath:   filepath.Join(dir, "test.db"),
		ListenAddr:     "127.0.0.1:0",
		CacheEnabled:   false,
		DemoMode:       true,
		ServerMode:     true,
		SubsonicServer: true,
	}
	db, err := store.OpenDB(cfg.DatabasePath, cfg)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewServer(cfg, db), db
}

func TestDemoModeBlocksRestMutations(t *testing.T) {
	srv, _ := newDemoSubsonicTestServer(t)
	handler := srv.Handler()

	for _, endpoint := range []string{"star", "scrobble", "createPlaylist", "createShare", "jukeboxControl"} {
		req := httptest.NewRequest(http.MethodGet, "/rest/"+endpoint+".view?f=json", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if !strings.Contains(rec.Body.String(), "read-only") {
			t.Fatalf("expected read-only error for %s, got %d %s", endpoint, rec.Code, rec.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/rest/ping.view?f=json", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("expected ping to stay readable in demo mode, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestDemoModeBlocksShareWriteMethods(t *testing.T) {
	srv, _ := newDemoTestServer(t)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodPost, "/s/sometoken", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for POST /s/ in demo mode, got %d", rec.Code)
	}

	// The share unlock POST is a read-side flow and stays available.
	req = httptest.NewRequest(http.MethodPost, "/s/sometoken/unlock", strings.NewReader(`{"password":"x"}`))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code == http.StatusForbidden {
		t.Fatalf("unlock POST should pass the demo gate, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/s/sometoken", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for DELETE /s/ in demo mode, got %d", rec.Code)
	}
}

func TestSubsonicAPIKeyIsNotPassword(t *testing.T) {
	srv, _ := newAuthedSubsonicServer(t)
	handler := srv.Handler()
	cookie := setupUser(t, handler, "admin", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/auth/subsonic-key", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("subsonic key status %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	apiKey, _ := payload["apiKey"].(string)
	if apiKey == "" || apiKey == "password123" {
		t.Fatalf("api key must be a generated secret, got %q", apiKey)
	}

	// The account password still works for the "p" parameter.
	req = httptest.NewRequest(http.MethodGet, "/rest/ping.view?f=json&u=admin&p=password123", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("password auth should still work: %s", rec.Body.String())
	}

	// The API key works as a "p" credential too.
	req = httptest.NewRequest(http.MethodGet, "/rest/ping.view?f=json&u=admin&p="+apiKey, nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("api key auth failed: %s", rec.Body.String())
	}

	// Token auth binds to the API key, not the account password.
	salt := "abc123"
	sum := md5.Sum([]byte(apiKey + salt)) //#nosec G401 -- Subsonic protocol token
	token := hex.EncodeToString(sum[:])
	req = httptest.NewRequest(http.MethodGet, "/rest/ping.view?f=json&u=admin&t="+token+"&s="+salt, nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("token auth with api key failed: %s", rec.Body.String())
	}

	// md5(password + salt) must no longer authenticate: the stored secret is
	// not the account password anymore.
	sum = md5.Sum([]byte("password123" + salt)) //#nosec G401 -- Subsonic protocol token
	token = hex.EncodeToString(sum[:])
	req = httptest.NewRequest(http.MethodGet, "/rest/ping.view?f=json&u=admin&t="+token+"&s="+salt, nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatal("token derived from account password must not authenticate")
	}
}

func TestSubsonicKeyRotateRevokesOld(t *testing.T) {
	srv, _ := newAuthedSubsonicServer(t)
	handler := srv.Handler()
	cookie := setupUser(t, handler, "admin", "password123")

	getKey := func() string {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/subsonic-key", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		var payload map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &payload)
		key, _ := payload["apiKey"].(string)
		return key
	}

	oldKey := getKey()
	if oldKey == "" {
		t.Fatal("expected api key")
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/subsonic-key/rotate", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("rotate status %d: %s", rec.Code, rec.Body.String())
	}
	newKey := getKey()
	if newKey == "" || newKey == oldKey {
		t.Fatal("rotate did not produce a new key")
	}

	req = httptest.NewRequest(http.MethodGet, "/rest/ping.view?f=json&u=admin&p="+oldKey, nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatal("rotated key still authenticates")
	}
}

func TestSubsonicKeyRequiresSession(t *testing.T) {
	srv, _ := newAuthedSubsonicServer(t)
	handler := srv.Handler()
	setupUser(t, handler, "admin", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/auth/subsonic-key", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", rec.Code)
	}
}

func TestAuthLoginRateLimited(t *testing.T) {
	srv, _ := newAuthedSubsonicServer(t)
	handler := srv.Handler()
	setupUser(t, handler, "admin", "password123")

	var lastCode int
	for range 12 {
		body := bytes.NewBufferString(`{"username":"admin","password":"bad"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		lastCode = rec.Code
	}
	if lastCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after repeated failures, got %d", lastCode)
	}
}

func TestShareUnlockRateLimited(t *testing.T) {
	srv, _ := newAuthedSubsonicServer(t)
	handler := srv.Handler()
	setupUser(t, handler, "admin", "password123")

	share, err := srv.shares.Create(store.CreateShareInput{
		UserID:       "admin",
		ResourceType: "song",
		ResourceID:   "song-1",
		AccessMode:   store.ShareAccessPassword,
		Password:     "sharepass",
	})
	if err != nil {
		t.Fatalf("create share: %v", err)
	}

	var lastCode int
	for range 12 {
		req := httptest.NewRequest(http.MethodPost, "/s/"+share.Token+"/unlock", strings.NewReader(`{"password":"nope"}`))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		lastCode = rec.Code
	}
	if lastCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after repeated bad unlocks, got %d", lastCode)
	}
}

func TestClientLogRateLimited(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Handler()

	var lastCode int
	for range 65 {
		req := httptest.NewRequest(http.MethodPost, "/api/client-log", strings.NewReader(`{"level":"info","message":"x"}`))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		lastCode = rec.Code
	}
	if lastCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after flooding client-log, got %d", lastCode)
	}
}

func TestConfigHidesDataDirFromAnonymous(t *testing.T) {
	srv, _ := newAuthedSubsonicServer(t)
	handler := srv.Handler()
	cookie := setupUser(t, handler, "admin", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("config status %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := payload["dataDir"]; ok {
		t.Fatal("dataDir must not be exposed to anonymous callers when auth is on")
	}
	ext, _ := payload["extensions"].(map[string]any)
	if dir, ok := ext["dir"]; ok && dir != "" {
		t.Fatal("extensions dir must not be exposed to anonymous callers")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/config", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["dataDir"] == "" {
		t.Fatal("authenticated caller should still see dataDir")
	}
}

func TestConfigShowsDataDirWhenAuthDisabled(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["dataDir"] == "" {
		t.Fatal("dataDir should be present when auth is disabled")
	}
}

func TestFilesystemBrowseRejectsSymlinkEscape(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Handler()

	outside := t.TempDir()
	link := filepath.Join(srv.cfg.DataDir, "escape-link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/filesystem/directories?path="+link, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("symlink escape outside browse roots must be rejected, got 200: %s", rec.Body.String())
	}

	inside := filepath.Join(srv.cfg.DataDir, "real-dir")
	if err := os.MkdirAll(inside, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/filesystem/directories?path="+inside, nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("in-root path should browse fine, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLocalLibraryPathRestrictedInServerMode(t *testing.T) {
	srv, _ := newAuthedSubsonicServer(t)
	handler := srv.Handler()
	cookie := setupUser(t, handler, "admin", "password123")

	outside := t.TempDir()
	body := bytes.NewBufferString(`{"name":"escape","path":"` + outside + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/local-libraries", body)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for out-of-roots library path, got %d: %s", rec.Code, rec.Body.String())
	}

	inside := filepath.Join(srv.cfg.DataDir, "music")
	if err := os.MkdirAll(inside, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body = bytes.NewBufferString(`{"name":"ok","path":"` + inside + `"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/local-libraries", body)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for in-roots library path, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestInstanceServerURLValidation(t *testing.T) {
	srv, _ := newAuthedSubsonicServer(t)
	handler := srv.Handler()
	cookie := setupUser(t, handler, "admin", "password123")

	for _, raw := range []string{
		"file:///etc/passwd",
		"gopher://internal",
		"https://user:pass@example.com",
		"not a url",
	} {
		body := bytes.NewBufferString(`{"serverUrl":"` + raw + `","username":"u","password":"p"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/instances/test", body)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %q, got %d: %s", raw, rec.Code, rec.Body.String())
		}
	}
}

func TestWebSocketOriginRejected(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Handler()

	mkReq := func(origin string) (*httptest.ResponseRecorder, *http.Request) {
		req := httptest.NewRequest(http.MethodGet, "http://player.example/api/ws", nil)
		req.Host = "player.example"
		// The handshake headers must be valid so the request reaches the
		// origin check inside websocket.Accept.
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Sec-WebSocket-Version", "13")
		req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		return httptest.NewRecorder(), req
	}

	// A browser page on a foreign origin must not open the control socket.
	rec, req := mkReq("http://evil.example")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for foreign origin, got %d", rec.Code)
	}

	// Same origin passes the origin gate. The request then fails later at the
	// websocket handshake or hijack stage with a different status.
	rec, req = mkReq("http://player.example")
	handler.ServeHTTP(rec, req)
	if rec.Code == http.StatusForbidden {
		t.Fatal("same-origin upgrade must pass the origin check")
	}

	// The configured CORS allowlist covers the Wails desktop origin.
	rec, req = mkReq("https://wails.localhost")
	handler.ServeHTTP(rec, req)
	if rec.Code == http.StatusForbidden {
		t.Fatal("wails.localhost origin must be allowed")
	}

	// Non browser clients send no Origin and stay allowed.
	rec, req = mkReq("")
	handler.ServeHTTP(rec, req)
	if rec.Code == http.StatusForbidden {
		t.Fatal("requests without an Origin header must be allowed")
	}
}
