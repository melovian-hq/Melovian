// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package democatalog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCatalogHasContent(t *testing.T) {
	c := Get()
	if len(c.Artists) < 12 {
		t.Fatalf("expected artists, got %d", len(c.Artists))
	}
	if len(c.Albums) < 16 {
		t.Fatalf("expected albums, got %d", len(c.Albums))
	}
	if len(c.Songs) < 80 {
		t.Fatalf("expected songs, got %d", len(c.Songs))
	}
	if len(c.Playlists) < 4 {
		t.Fatalf("expected playlists, got %d", len(c.Playlists))
	}
	songs, albums, artists := Stats()
	if songs != len(c.Songs) || albums != len(c.Albums) || artists != len(c.Artists) {
		t.Fatalf("stats mismatch")
	}
}

func TestHandlerPingAndAlbumList(t *testing.T) {
	h := Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/subsonic/rest/ping.view?f=json", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ping status %d", rec.Code)
	}
	var ping map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &ping); err != nil {
		t.Fatalf("decode ping: %v", err)
	}
	resp := ping["subsonic-response"].(map[string]any)
	if resp["status"] != "ok" {
		t.Fatalf("ping not ok: %+v", resp)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/subsonic/rest/getAlbumList2.view?type=newest&size=5&f=json", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("album list status %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/subsonic/rest/getCoverArt.view?id=al-001-01", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("cover status %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		t.Fatalf("cover content-type %q", ct)
	}
	if len(rec.Body.Bytes()) < 100 {
		t.Fatalf("cover body too small")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/subsonic/rest/getCoverArt.view?id=pl-hits", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("playlist cover status %d", rec.Code)
	}
	ct = rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		t.Fatalf("playlist cover content-type %q", ct)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/subsonic/rest/stream.view?id=tr-0001", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("stream status %d", rec.Code)
	}
	if len(rec.Body.Bytes()) < 44 {
		t.Fatalf("stream too short")
	}
}

func TestIsFakeURL(t *testing.T) {
	if !IsFakeURL("fake://melovian-demo") {
		t.Fatal("expected fake url")
	}
	if IsFakeURL("https://music.example") {
		t.Fatal("real url should not be fake")
	}
}
