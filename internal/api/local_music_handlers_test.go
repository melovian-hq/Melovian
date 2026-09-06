// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"melovian/internal/api"
	"melovian/internal/appconfig"
	"melovian/internal/metaloader"
	"melovian/internal/store"
)

func TestLocalMusicBrowseAndStream(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "alpha.mp3"), []byte("alpha-track-data"), 0o644); err != nil {
		t.Fatalf("write track: %v", err)
	}

	db := store.OpenTestDB(t)
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

	cfg := appconfig.Config{DataDir: t.TempDir(), LocalLibrary: appconfig.LocalLibraryConfig{Enabled: true}}
	server := api.NewServer(cfg, db)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/sources/view-mode", bytes.NewReader([]byte(`{"mode":"local"}`)))
	req.Header.Set("Content-Type", "application/json")
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("set source view mode: %d body=%s", rec.Code, rec.Body.String())
	}

	t.Run("status", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/music/status", nil)
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status code: %d body=%s", rec.Code, rec.Body.String())
		}
		var payload map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if payload["source"] != "local" || payload["connected"] != true {
			t.Fatalf("unexpected status payload: %+v", payload)
		}
	})

	t.Run("artists", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/local-music/artists", nil)
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status code: %d body=%s", rec.Code, rec.Body.String())
		}
		var payload struct {
			Artists []map[string]any `json:"artists"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(payload.Artists) != 1 {
			t.Fatalf("expected one artist, got %+v", payload.Artists)
		}
	})

	t.Run("stream", func(t *testing.T) {
		items, err := tracks.ListPresent(lib.ID)
		if err != nil || len(items) == 0 {
			t.Fatalf("ListPresent: %v len=%d", err, len(items))
		}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/local-music/tracks/"+items[0].ID+"/stream", nil)
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("stream status: %d body=%s", rec.Code, rec.Body.String())
		}
		if rec.Body.String() != "alpha-track-data" {
			t.Fatalf("unexpected stream body: %q", rec.Body.String())
		}
	})

	t.Run("search", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/local-music/search?q=alpha&limit=10", nil)
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("search status: %d body=%s", rec.Code, rec.Body.String())
		}
	})

	if _, err := tracks.Upsert(store.UpsertLocalTrackInput{
		LibraryID: lib.ID,
		RelPath:   "jazz-one.mp3",
		AbsPath:   filepath.Join(root, "jazz-one.mp3"),
		Title:     "Jazz One",
		Artist:    "Jazz Artist",
		Album:     "Jazz Album",
		Genre:     "Jazz",
		Format:    "mp3",
	}); err != nil {
		t.Fatalf("seed genre track: %v", err)
	}

	t.Run("genres", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/local-music/genres", nil)
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("genres status: %d body=%s", rec.Code, rec.Body.String())
		}
		var payload struct {
			Genres []struct {
				Name      string `json:"name"`
				SongCount int    `json:"songCount"`
			} `json:"genres"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(payload.Genres) != 1 || payload.Genres[0].Name != "Jazz" || payload.Genres[0].SongCount != 1 {
			t.Fatalf("unexpected genres: %+v", payload.Genres)
		}
	})

	t.Run("songsByGenre", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/local-music/songsByGenre?genre=jazz", nil)
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("songsByGenre status: %d body=%s", rec.Code, rec.Body.String())
		}
		var payload struct {
			Songs []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"songs"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(payload.Songs) != 1 || payload.Songs[0].Title != "Jazz One" {
			t.Fatalf("unexpected songsByGenre: %+v", payload.Songs)
		}
	})

	t.Run("played count decorates song", func(t *testing.T) {
		items, err := tracks.ListPresent(lib.ID)
		if err != nil || len(items) == 0 {
			t.Fatalf("ListPresent: %v len=%d", err, len(items))
		}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/music/items/"+items[0].ID+"/played", nil)
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent && rec.Code != http.StatusOK {
			t.Fatalf("played status: %d body=%s", rec.Code, rec.Body.String())
		}

		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, "/api/local-music/songs/"+items[0].ID, nil)
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("song status: %d body=%s", rec.Code, rec.Body.String())
		}
		var song map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &song); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if count, _ := song["playCount"].(float64); count < 1 {
			t.Fatalf("expected playCount >= 1, got %+v", song)
		}
	})

	t.Run("starred lists favorites", func(t *testing.T) {
		items, err := tracks.ListPresent(lib.ID)
		if err != nil || len(items) == 0 {
			t.Fatalf("ListPresent: %v len=%d", err, len(items))
		}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/music/favorites/"+items[0].ID,
			bytes.NewBufferString(`{"trackTitle":"Fav","artistName":"A"}`))
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent && rec.Code != http.StatusOK {
			t.Fatalf("favorite status: %d body=%s", rec.Code, rec.Body.String())
		}

		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, "/api/local-music/starred", nil)
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("starred status: %d body=%s", rec.Code, rec.Body.String())
		}
		var payload struct {
			Songs []struct {
				ID      string `json:"id"`
				Starred string `json:"starred"`
			} `json:"songs"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(payload.Songs) != 1 || payload.Songs[0].ID != items[0].ID || payload.Songs[0].Starred == "" {
			t.Fatalf("unexpected starred payload: %+v", payload.Songs)
		}
	})

	t.Run("similar", func(t *testing.T) {
		jazz, err := tracks.GetByRelPath(lib.ID, "jazz-one.mp3")
		if err != nil {
			t.Fatalf("GetByRelPath: %v", err)
		}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/local-music/songs/"+jazz.ID+"/similar?count=5", nil)
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("similar status: %d body=%s", rec.Code, rec.Body.String())
		}
	})
}
