// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"melovian/internal/api/music"
	"melovian/internal/store"
)

func TestLyricsSettingsDefaultsAndCacheClear(t *testing.T) {
	srv, db := newTestServer(t)
	instances := store.NewInstanceStore(db)
	inst, err := instances.Create(store.CreateInstanceInput{
		Name:      "Lyrics",
		ServerURL: "http://example.com",
		Username:  "user",
		Password:  "pass",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/music/settings/lyrics", nil)
	req.Header.Set("X-Instance-Id", inst.ID)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get lyrics settings status %d: %s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if payload["autoFetch"] != false {
		t.Fatalf("expected autoFetch=false, got %v", payload["autoFetch"])
	}
	providers, ok := payload["providers"].([]any)
	if !ok || len(providers) < 3 {
		t.Fatalf("expected default providers, got %v", payload["providers"])
	}

	body := `{"autoFetch":true,"providers":[{"id":"lrclib","enabled":true},{"id":"lyrics-ovh","enabled":false}]}`
	req = httptest.NewRequest(http.MethodPut, "/api/music/settings/lyrics", strings.NewReader(body))
	req.Header.Set("X-Instance-Id", inst.ID)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put lyrics settings status %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/music/lyrics/cache", nil)
	req.Header.Set("X-Instance-Id", inst.ID)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("clear lyrics cache status %d", rec.Code)
	}
}

func TestMergeLyricsSettingsAddsMissingBuiltins(t *testing.T) {
	settings, err := music.MergeLyricsSettings([]byte(`{"providers":[{"id":"custom-1","custom":true,"enabled":true,"url":"https://example.com?a={artist}"}]}`), "/data")
	if err != nil {
		t.Fatalf("mergeLyricsSettings: %v", err)
	}
	if len(settings.Providers) < 4 {
		t.Fatalf("expected custom + built-in providers, got %d", len(settings.Providers))
	}
}

func TestLyricsRootRejectsRelativeEscape(t *testing.T) {
	srv, _ := newTestServer(t)
	escaped := srv.lyricsRoot(music.LyricsSettings{StorageDir: "../../tmp/pwned"})
	want := music.DefaultLyricsStorageDir(srv.cfg.DataDir)
	if escaped != want {
		t.Fatalf("escaped relative dir = %q, want default %q", escaped, want)
	}
	nested := srv.lyricsRoot(music.LyricsSettings{StorageDir: "custom/cache"})
	if nested != filepath.Join(srv.cfg.DataDir, "custom", "cache") {
		t.Fatalf("nested relative dir = %q", nested)
	}
}

func TestLyricsRootKeepsAbsoluteCustomDir(t *testing.T) {
	srv, _ := newTestServer(t)
	custom := filepath.Join(t.TempDir(), "lyrics-cache")
	got := srv.lyricsRoot(music.LyricsSettings{StorageDir: custom})
	if got != filepath.Clean(custom) {
		t.Fatalf("absolute custom dir = %q, want %q", got, filepath.Clean(custom))
	}
}
