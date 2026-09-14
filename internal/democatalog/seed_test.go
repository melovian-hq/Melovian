// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package democatalog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"melovian/internal/appconfig"
	"melovian/internal/store"
)

// A fresh demo-mode database must come up with the fake catalog instance
// seeded, active, and set as the subsonic source so the frontend skips setup.
func TestFreshDemoDBProvisionsActiveInstance(t *testing.T) {
	cfg := appconfig.Config{ServerMode: true, DemoMode: true}
	ApplyDefaults(&cfg)
	if cfg.LegacyServer != ServerURL {
		t.Fatalf("ApplyDefaults server = %q, want %q", cfg.LegacyServer, ServerURL)
	}

	db, err := store.OpenDB(filepath.Join(t.TempDir(), "demo.db"), cfg)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer func() { _ = db.Close() }()

	instances := store.NewInstanceStore(db)
	active, err := instances.GetActive()
	if err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	if !IsFakeURL(active.ServerURL) {
		t.Fatalf("active instance url = %q, want fake catalog", active.ServerURL)
	}

	mode, err := store.NewPreferencesStore(db).GetSourceViewMode("")
	if err != nil {
		t.Fatalf("GetSourceViewMode: %v", err)
	}
	if mode != store.SourceViewSubsonic {
		t.Fatalf("source view mode = %q, want subsonic", mode)
	}

	inst, err := EnsureInstance(instances)
	if err != nil {
		t.Fatalf("EnsureInstance: %v", err)
	}
	if inst.ID != active.ID {
		t.Fatalf("EnsureInstance id = %q, want active %q", inst.ID, active.ID)
	}
	if inst.Name != ServerName || inst.ServerName != ServerName {
		t.Fatalf("demo instance name = %q/%q, want %q", inst.Name, inst.ServerName, ServerName)
	}

	scope := "instance:" + inst.ID
	listen := store.NewListenStore(db)
	if err := SeedUserData(listen, scope); err != nil {
		t.Fatalf("SeedUserData: %v", err)
	}
	playlists, err := listen.ListPlaylists(scope)
	if err != nil {
		t.Fatalf("ListPlaylists: %v", err)
	}
	if len(playlists) == 0 {
		t.Fatal("expected seeded playlists for demo scope")
	}
}

// The demo catalog handler must answer with a populated library so browse
// pages render content on first load.
func TestFreshDemoCatalogServesLibrary(t *testing.T) {
	h := Handler()

	get := func(path string) map[string]any {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status %d", path, rec.Code)
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s decode: %v", path, err)
		}
		resp, ok := body["subsonic-response"].(map[string]any)
		if !ok {
			t.Fatalf("%s missing subsonic-response: %v", path, body)
		}
		if resp["status"] != "ok" {
			t.Fatalf("%s not ok: %v", path, resp)
		}
		return resp
	}

	artists := get("/api/subsonic/rest/getArtists.view?f=json")
	index, ok := artists["artists"].(map[string]any)["index"].([]any)
	if !ok || len(index) == 0 {
		t.Fatalf("expected artist indexes, got %v", artists["artists"])
	}

	albums := get("/api/subsonic/rest/getAlbumList2.view?type=newest&size=10&f=json")
	list, ok := albums["albumList2"].(map[string]any)["album"].([]any)
	if !ok || len(list) == 0 {
		t.Fatalf("expected albums, got %v", albums["albumList2"])
	}

	songs := get("/api/subsonic/rest/getRandomSongs.view?size=10&f=json")
	random, ok := songs["randomSongs"].(map[string]any)["song"].([]any)
	if !ok || len(random) == 0 {
		t.Fatalf("expected songs, got %v", songs["randomSongs"])
	}
}
