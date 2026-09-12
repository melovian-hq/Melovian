// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"melovian/internal/appconfig"
	"melovian/internal/store"
)

func newTestServer(t *testing.T) (*Server, *store.DB) {
	t.Helper()
	dir := t.TempDir()
	cfg := appconfig.Config{
		DataDir:      dir,
		DatabasePath: filepath.Join(dir, "test.db"),
		ListenAddr:   "127.0.0.1:0",
		CacheEnabled: false,
	}
	db, err := store.OpenDB(cfg.DatabasePath, cfg)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewServer(cfg, db), db
}

func newTestServerWithLocalLibrary(t *testing.T) (*Server, *store.DB) {
	t.Helper()
	dir := t.TempDir()
	cfg := appconfig.Config{
		DataDir:      dir,
		DatabasePath: filepath.Join(dir, "test.db"),
		ListenAddr:   "127.0.0.1:0",
		CacheEnabled: false,
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

func TestGetConfig(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["dataDir"] == nil {
		t.Fatal("expected dataDir in response")
	}
	if payload["listenAddr"] == nil {
		t.Fatal("expected listenAddr in response")
	}
	if _, ok := payload["navidromeServer"]; ok {
		t.Fatal("legacy navidromeServer should not be exposed")
	}
}

func TestUpdateInstance(t *testing.T) {
	srv, _ := newTestServer(t)
	instances := srv.instances

	inst, err := instances.Create(store.CreateInstanceInput{
		Name:      "Original",
		ServerURL: "http://subsonic.example.com",
		Username:  "user",
		Password:  "pass",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	body := bytes.NewBufferString(`{"name":"Renamed"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/instances/"+inst.ID, body)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status %d: %s", rec.Code, rec.Body.String())
	}

	updated, err := instances.Get(inst.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if updated.Name != "Renamed" {
		t.Fatalf("expected name Renamed, got %q", updated.Name)
	}
	if updated.ServerURL != inst.ServerURL {
		t.Fatalf("server url should be preserved, got %q", updated.ServerURL)
	}
	if updated.Password != "pass" {
		t.Fatalf("password should be preserved when omitted")
	}
}

func TestPingInstance(t *testing.T) {
	subsonicSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"subsonic-response":{"status":"ok","version":"1.16.1","server":{"name":"Navidrome","version":"0.52.0"}}}`)
	}))
	defer subsonicSrv.Close()
	srv, _ := newTestServer(t)
	instances := srv.instances
	inst, err := instances.Create(store.CreateInstanceInput{
		Name:      "Ping Target",
		ServerURL: subsonicSrv.URL,
		Username:  "user",
		Password:  "pass",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/instances/"+inst.ID+"/ping", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ping status %d: %s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["online"] != true {
		t.Fatalf("expected online true, got %v", payload["online"])
	}
	if _, ok := payload["latencyMs"]; !ok {
		t.Fatal("expected latencyMs in response")
	}
}

func TestPingInstanceNotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/instances/missing/ping", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestInstanceRoutes(t *testing.T) {
	srv, db := newTestServer(t)
	instances := store.NewInstanceStore(db)

	inst, err := instances.Create(store.CreateInstanceInput{
		Name:      "Test",
		ServerURL: "http://subsonic.example.com",
		Username:  "user",
		Password:  "pass",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	_ = instances.SetActive(inst.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/instances", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/instances/active", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("active status %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/instances/"+inst.ID+"/activate", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("activate status %d", rec.Code)
	}
}

func TestResolveProgressUserIDFromHeader(t *testing.T) {
	srv, db := newTestServer(t)
	instances := store.NewInstanceStore(db)

	inst, err := instances.Create(store.CreateInstanceInput{
		Name:      "Scoped",
		ServerURL: "http://subsonic.example.com",
		Username:  "user",
		Password:  "pass",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	body := bytes.NewBufferString(`{"positionMs":5000,"trackTitle":"Test Track"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/music/items/track-42", body)
	req.Header.Set("X-Instance-Id", inst.ID)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("upsert status %d: %s", rec.Code, rec.Body.String())
	}

	listen := store.NewListenStore(db)
	item, err := listen.Get("instance:"+inst.ID, "track-42")
	if err != nil {
		t.Fatalf("Get scoped progress: %v", err)
	}
	if item.PositionMs != 5000 {
		t.Fatalf("expected position 5000, got %d", item.PositionMs)
	}
}

func TestUnknownInstanceHeader(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/music/history", nil)
	req.Header.Set("X-Instance-Id", "does-not-exist")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUnknownInstanceQueryParam(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/music/history?_instance=does-not-exist", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestMusicHistoryEmpty(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/music/history", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if !bytes.Contains(body, []byte(`"items"`)) {
		t.Fatalf("unexpected body: %s", body)
	}
}
