// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"melovian/internal/metaloader"
	"melovian/internal/store"
)

func assertJSONKeys(t *testing.T, payload map[string]any, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if _, ok := payload[key]; !ok {
			t.Fatalf("missing required JSON key %q in %+v", key, payload)
		}
	}
}

func assertJSONArrayField(t *testing.T, payload map[string]any, field string) {
	t.Helper()
	raw, ok := payload[field]
	if !ok {
		t.Fatalf("missing array field %q", field)
	}
	if _, isSlice := raw.([]any); !isSlice {
		t.Fatalf("field %q should be an array, got %T", field, raw)
	}
}

func decodeJSONResponse(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("expected application/json, got %q body=%s", rec.Header().Get("Content-Type"), rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode JSON: %v body=%s", err, rec.Body.String())
	}
	return payload
}

func serveGET(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestConfigContract(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := serveGET(t, srv.Handler(), "/api/config")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	payload := decodeJSONResponse(t, rec)
	assertJSONKeys(t, payload,
		"dataDir", "listenAddr", "authEnabled", "oidcEnabled",
		"oidcLoginUrl", "serverMode", "connectionDefaults", "localLibrary",
		"version", "apiVersion", "minClientVersion", "minServerVersion", "capabilities",
	)
	lib, ok := payload["localLibrary"].(map[string]any)
	if !ok {
		t.Fatal("localLibrary should be an object")
	}
	assertJSONKeys(t, lib, "enabled", "defaultPath", "allowCustomPath")
}

func TestAuthStatusContract(t *testing.T) {
	srv, _ := newAuthTestServer(t)
	rec := serveGET(t, srv.Handler(), "/api/auth/status")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	payload := decodeJSONResponse(t, rec)
	assertJSONKeys(t, payload, "enabled", "setupRequired", "authenticated")
}

func TestInstancesListContract(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := serveGET(t, srv.Handler(), "/api/instances")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	payload := decodeJSONResponse(t, rec)
	assertJSONArrayField(t, payload, "instances")
}

func TestSourcesStatusContract(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := serveGET(t, srv.Handler(), "/api/sources/status")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	payload := decodeJSONResponse(t, rec)
	assertJSONKeys(t, payload, "mode", "activeInstanceId", "activeLocalId", "unifiedAvailable")
}

func TestMusicStatusContract(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := serveGET(t, srv.Handler(), "/api/music/status")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	payload := decodeJSONResponse(t, rec)
	assertJSONKeys(t, payload, "enabled", "connected")
	if payload["enabled"] != false || payload["connected"] != false {
		t.Fatalf("expected disconnected subsonic status, got %+v", payload)
	}
}

func TestMusicStatusLocalModeContract(t *testing.T) {
	srv, db := newTestServerWithLocalLibrary(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "track.mp3"), []byte("data"), 0o644); err != nil {
		t.Fatalf("write track: %v", err)
	}

	libraries := store.NewLocalLibraryStore(db)
	tracks := store.NewLocalTrackStore(db)
	scanner := metaloader.NewScanner(libraries, tracks)

	lib, err := libraries.Create(store.CreateLocalLibraryInput{Name: "Local", Path: root})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := scanner.Scan(lib.ID); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if err := libraries.SetActive(lib.ID); err != nil {
		t.Fatalf("SetActive: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/sources/view-mode", strings.NewReader(`{"mode":"local"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("set source view mode: %d body=%s", rec.Code, rec.Body.String())
	}

	rec = serveGET(t, srv.Handler(), "/api/music/status")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	payload := decodeJSONResponse(t, rec)
	assertJSONKeys(t, payload, "enabled", "connected", "source", "serverName", "version")
	if payload["source"] != "local" || payload["connected"] != true || payload["enabled"] != true {
		t.Fatalf("unexpected local status payload: %+v", payload)
	}
}

func TestMusicLibraryStatsContract(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := serveGET(t, srv.Handler(), "/api/music/library-stats")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	payload := decodeJSONResponse(t, rec)
	assertJSONKeys(t, payload, "songCount", "albumCount", "artistCount", "folderCount", "scanning")
}

func TestMusicSettingsContracts(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Handler()

	for _, path := range []string{
		"/api/music/settings/eq",
		"/api/music/settings/connection",
		"/api/music/settings/cache",
		"/api/music/settings/lyrics",
	} {
		t.Run(path, func(t *testing.T) {
			rec := serveGET(t, handler, path)
			if rec.Code != http.StatusOK {
				t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("expected application/json for %s", path)
			}
			var payload map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode %s: %v", path, err)
			}
		})
	}
}

func TestMusicBatchContract(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := serveGET(t, srv.Handler(), "/api/music/batch?ids=")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode batch map: %v", err)
	}
	if payload == nil {
		t.Fatal("expected batch object")
	}
}

func TestMusicStatsContract(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := serveGET(t, srv.Handler(), "/api/music/stats?limit=5")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	payload := decodeJSONResponse(t, rec)
	assertJSONKeys(t, payload, "totalPlays", "uniqueTracks", "totalListeningMs", "topArtists", "topTracks", "topAlbums")
}

func TestMusicListenEventYearsContract(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := serveGET(t, srv.Handler(), "/api/music/listen-events/years")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	payload := decodeJSONResponse(t, rec)
	assertJSONArrayField(t, payload, "years")
}

func TestDownloadsListContract(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := serveGET(t, srv.Handler(), "/api/downloads")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	payload := decodeJSONResponse(t, rec)
	assertJSONArrayField(t, payload, "downloads")
}

func TestHealthContract(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := serveGET(t, srv.Handler(), "/health")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	payload := decodeJSONResponse(t, rec)
	assertJSONKeys(t, payload, "status", "version", "apiVersion", "capabilities")
}

func TestAPIErrorShapeContract(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodPut, "/api/sources/view-mode", strings.NewReader("not-json"))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	payload := decodeJSONResponse(t, rec)
	assertJSONKeys(t, payload, "error", "code")
}

func TestLibraryRefreshContract(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/music/library/refresh", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	payload := decodeJSONResponse(t, rec)
	assertJSONKeys(t, payload, "invalidated")
}

func TestLocalLibrariesContract(t *testing.T) {
	srv, _ := newTestServerWithLocalLibrary(t)
	rec := serveGET(t, srv.Handler(), "/api/local-libraries")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	payload := decodeJSONResponse(t, rec)
	assertJSONArrayField(t, payload, "libraries")
}

func TestJSONFieldNamingConvention(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := serveGET(t, srv.Handler(), "/api/music/status")
	payload := decodeJSONResponse(t, rec)
	for key := range payload {
		if strings.Contains(key, "_") {
			t.Fatalf("JSON key %q uses snake_case. API responses should use camelCase", key)
		}
	}
}
