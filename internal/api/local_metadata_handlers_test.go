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

func writeTestWAV(t *testing.T, path string) {
	t.Helper()
	hdr := []byte("RIFF$\x00\x00\x00WAVEfmt \x10\x00\x00\x00\x01\x00\x01\x00\x44\xac\x00\x00\x88X\x01\x00\x02\x00\x10\x00data\x00\x00\x00\x00")
	if err := os.WriteFile(path, hdr, 0o644); err != nil {
		t.Fatalf("write wav: %v", err)
	}
}

func TestLocalMetadataWriteAndSuggestions(t *testing.T) {
	root := t.TempDir()
	rel := "Slowdive/Souvlaki/03 Alison.wav"
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeTestWAV(t, abs)

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

	present, err := tracks.ListPresent(lib.ID)
	if err != nil || len(present) != 1 {
		t.Fatalf("ListPresent: %v len=%d", err, len(present))
	}
	trackID := present[0].ID

	cfg := appconfig.Config{DataDir: t.TempDir(), LocalLibrary: appconfig.LocalLibraryConfig{Enabled: true}}
	server := api.NewServer(cfg, db)

	t.Run("suggestions", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/local-music/metadata/tracks/"+trackID+"/suggestions", nil)
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status: %d body=%s", rec.Code, rec.Body.String())
		}
		var payload struct {
			Filename struct {
				Artist   string `json:"artist"`
				Album    string `json:"album"`
				Title    string `json:"title"`
				TrackNum int    `json:"trackNum"`
			} `json:"filename"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if payload.Filename.Artist != "Slowdive" || payload.Filename.Album != "Souvlaki" || payload.Filename.Title != "Alison" {
			t.Fatalf("unexpected filename suggestion: %+v", payload.Filename)
		}
	})

	t.Run("patch metadata", func(t *testing.T) {
		body := []byte(`{"title":"Alison","artist":"Slowdive","album":"Souvlaki","year":1993,"genre":"Shoegaze"}`)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, "/api/local-music/metadata/tracks/"+trackID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("patch status: %d body=%s", rec.Code, rec.Body.String())
		}
		var payload struct {
			Title  string `json:"title"`
			Artist string `json:"artist"`
			Genre  string `json:"genre"`
			Year   int    `json:"year"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if payload.Title != "Alison" || payload.Artist != "Slowdive" || payload.Genre != "Shoegaze" || payload.Year != 1993 {
			t.Fatalf("unexpected payload: %+v", payload)
		}
	})

	t.Run("batch filename autofix", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{
			"trackIds": []string{trackID},
			"source":   "filename",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/local-music/metadata/autofix-batch", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("batch status: %d body=%s", rec.Code, rec.Body.String())
		}
		var payload struct {
			Results []struct {
				OK bool `json:"ok"`
			} `json:"results"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(payload.Results) != 1 || !payload.Results[0].OK {
			t.Fatalf("unexpected batch payload: %+v", payload)
		}
	})
}

func TestLocalMetadataSearchAndSummary(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "broken.mp3"), []byte("not-a-real-mp3"), 0o644); err != nil {
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

	present, err := tracks.ListPresent(lib.ID)
	if err != nil || len(present) == 0 {
		t.Fatalf("ListPresent: %v len=%d", err, len(present))
	}
	track := present[0]
	if _, err := tracks.Upsert(store.UpsertLocalTrackInput{
		LibraryID:   lib.ID,
		RelPath:     track.RelPath,
		AbsPath:     track.AbsPath,
		FileSig:     track.FileSig,
		ContentHash: track.ContentHash,
		Size:        track.Size,
		Mtime:       track.Mtime,
		Title:       "Mystery Song",
		Artist:      "Unknown Artist",
		Album:       "Unknown Album",
		AlbumArtist: "",
		TrackNum:    1,
		DiscNum:     1,
		DurationMs:  0,
		Genre:       "",
		Year:        0,
		Format:      track.Format,
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	cfg := appconfig.Config{DataDir: t.TempDir(), LocalLibrary: appconfig.LocalLibraryConfig{Enabled: true}}
	server := api.NewServer(cfg, db)

	t.Run("summary", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/local-music/metadata/summary", nil)
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("summary status: %d body=%s", rec.Code, rec.Body.String())
		}
		var payload struct {
			Any int `json:"any"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if payload.Any < 1 {
			t.Fatalf("expected metadata issues, got %+v", payload)
		}
	})

	t.Run("search by title", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/local-music/metadata/tracks?q=mystery", nil)
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("search status: %d body=%s", rec.Code, rec.Body.String())
		}
		var payload struct {
			Tracks []map[string]any `json:"tracks"`
			Total  int              `json:"total"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if payload.Total != 1 || len(payload.Tracks) != 1 {
			t.Fatalf("unexpected search payload: %+v", payload)
		}
	})

	t.Run("filter unknown artist", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/local-music/metadata/tracks?issue=unknown-artist", nil)
		server.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("filter status: %d body=%s", rec.Code, rec.Body.String())
		}
	})
}
