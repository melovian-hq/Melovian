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

func TestLocalVideosListExcludesAudio(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "song.mp3"), []byte("audio-bytes"), 0o644); err != nil {
		t.Fatalf("write audio: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "clip.mp4"), []byte("video-bytes"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
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
		t.Fatalf("view-mode status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(
		http.MethodPut,
		"/api/video/settings",
		bytes.NewReader([]byte(`{"enabled":true,"searchProvider":"invidious"}`)),
	)
	req.Header.Set("Content-Type", "application/json")
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("enable videos status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/local-music/videos", nil)
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("videos status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Videos []struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			Format string `json:"format"`
		} `json:"videos"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Videos) != 1 {
		t.Fatalf("expected 1 video, got %d (%+v)", len(payload.Videos), payload.Videos)
	}
	if payload.Videos[0].Format != "mp4" {
		t.Fatalf("format=%q", payload.Videos[0].Format)
	}

	artistsRec := httptest.NewRecorder()
	artistsReq := httptest.NewRequest(http.MethodGet, "/api/local-music/artists", nil)
	server.Handler().ServeHTTP(artistsRec, artistsReq)
	if artistsRec.Code != http.StatusOK {
		t.Fatalf("artists status=%d", artistsRec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/local-music/tracks/"+payload.Videos[0].ID+"/stream", nil)
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("stream status=%d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "video/mp4" {
		t.Fatalf("content-type=%q", ct)
	}
}

func TestVideoSettingsAndLinkRoundTrip(t *testing.T) {
	db := store.OpenTestDB(t)
	cfg := appconfig.Config{DataDir: t.TempDir()}
	server := api.NewServer(cfg, db)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/api/video/settings",
		bytes.NewReader([]byte(`{
			"enabled":true,
			"searchProvider":"invidious",
			"invidiousBaseUrl":"https://invidious.example.com/",
			"youtubeApiKey":""
		}`)),
	)
	req.Header.Set("Content-Type", "application/json")
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put settings status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/video/settings", nil)
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get settings status=%d", rec.Code)
	}
	var settings api.VideoSettings
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if !settings.Enabled {
		t.Fatal("expected enabled")
	}
	if settings.InvidiousBaseURL != "https://invidious.example.com" {
		t.Fatalf("url=%q", settings.InvidiousBaseURL)
	}
	if settings.SearchProvider != "invidious" {
		t.Fatalf("provider=%q", settings.SearchProvider)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(
		http.MethodPut,
		"/api/video/links/trk_demo",
		bytes.NewReader([]byte(`{"source":"invidious","videoId":"abc","title":"Clip"}`)),
	)
	req.Header.Set("Content-Type", "application/json")
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put link status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/video/resolve?source=invidious&id=abc", nil)
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("resolve status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resolved struct {
		EmbedURL string `json:"embedUrl"`
		PlayID   string `json:"playId"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resolved); err != nil {
		t.Fatalf("decode resolve: %v", err)
	}
	if resolved.EmbedURL != "https://invidious.example.com/embed/abc" {
		t.Fatalf("embed=%q", resolved.EmbedURL)
	}
	if resolved.PlayID != "ext:invidious:abc" {
		t.Fatalf("playId=%q", resolved.PlayID)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/video/resolve?source=youtube&id=yt123", nil)
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("youtube resolve status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resolved); err != nil {
		t.Fatalf("decode youtube resolve: %v", err)
	}
	if resolved.EmbedURL != "https://www.youtube.com/embed/yt123" {
		t.Fatalf("youtube embed=%q", resolved.EmbedURL)
	}
	if resolved.PlayID != "ext:youtube:yt123" {
		t.Fatalf("youtube playId=%q", resolved.PlayID)
	}
}

func TestVideosDisabledByDefault(t *testing.T) {
	db := store.OpenTestDB(t)
	cfg := appconfig.Config{DataDir: t.TempDir()}
	server := api.NewServer(cfg, db)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/video/settings", nil)
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	var settings api.VideoSettings
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if settings.Enabled {
		t.Fatal("videos should be disabled by default")
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/local-music/videos", nil)
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when disabled, got %d", rec.Code)
	}
}
