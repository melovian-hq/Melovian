// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonic

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestInjectAuthStripsExistingAuthParams(t *testing.T) {
	client := NewClient("http://example.com", "alice", "secret")
	query := url.Values{
		"u":   {"olduser"},
		"t":   {"oldtoken"},
		"s":   {"oldsalt"},
		"p":   {"oldpass"},
		"jwt": {"oldjwt"},
		"id":  {"track-1"},
	}
	merged := client.InjectAuth(query)

	if merged.Get("u") != "alice" {
		t.Fatalf("expected username alice, got %q", merged.Get("u"))
	}
	if merged.Get("id") != "track-1" {
		t.Fatalf("expected track id preserved, got %q", merged.Get("id"))
	}
	if merged.Get("t") == "oldtoken" || merged.Get("s") == "oldsalt" {
		t.Fatal("expected fresh auth token and salt")
	}
	if merged.Get("c") != ClientName || merged.Get("v") != Version {
		t.Fatalf("expected client metadata, got c=%q v=%q", merged.Get("c"), merged.Get("v"))
	}
}

func TestClientEnabled(t *testing.T) {
	if NewClient("", "user", "pass").Enabled() {
		t.Fatal("empty server url should disable client")
	}
	if !NewClient("http://example.com", "user", "pass").Enabled() {
		t.Fatal("expected enabled client")
	}
}

func TestPingJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/rest/ping.view") {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if r.URL.Query().Get("u") != "alice" {
			t.Fatalf("expected auth username, got %q", r.URL.Query().Get("u"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"subsonic-response": map[string]any{
				"status":  "ok",
				"version": "1.16.1",
				"server": map[string]any{
					"name":    "Navidrome",
					"version": "0.54.0",
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "alice", "secret")
	name, version, err := client.Ping()
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if name != "Navidrome" || version != "0.54.0" {
		t.Fatalf("unexpected ping result %q %q", name, version)
	}
}

func TestPingRejectsOversizeBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("x", (4<<20)+64))
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "alice", "secret")
	if _, _, err := client.Ping(); err == nil {
		t.Fatal("expected oversize ping body to fail")
	}
}

func TestPingJSONOpenSubsonic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/rest/ping.view") {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"subsonic-response": map[string]any{
				"status":        "ok",
				"version":       "1.16.1",
				"type":          "navidrome",
				"serverVersion": "0.58.0",
				"openSubsonic":  true,
			},
		})
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "alice", "secret")
	name, version, err := client.Ping()
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if name != "navidrome" || version != "0.58.0" {
		t.Fatalf("unexpected ping result %q %q", name, version)
	}
}

func TestPingXMLError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<subsonic-response status="failed"><error code="40" message="Wrong password"/></subsonic-response>`))
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "alice", "secret")
	_, _, err := client.Ping()
	if err == nil || !strings.Contains(err.Error(), "Wrong password") {
		t.Fatalf("expected password error, got %v", err)
	}
}

func TestStreamSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/rest/stream.view") {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if r.URL.Query().Get("id") != "track-1" {
			t.Fatalf("expected track id, got %q", r.URL.Query().Get("id"))
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("audio-bytes"))
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "alice", "secret")
	body, contentType, err := client.Stream("track-1")
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	defer func() { _ = body.Close() }()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(data) != "audio-bytes" || contentType != "audio/mpeg" {
		t.Fatalf("unexpected stream result %q %q", string(data), contentType)
	}
}

func TestStreamNonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "alice", "secret")
	_, _, err := client.Stream("missing")
	if err == nil || !strings.Contains(err.Error(), "status 404") {
		t.Fatalf("expected stream error, got %v", err)
	}
}

func TestLibraryStatsIncludesArtistAlbumCounts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/rest/getScanStatus.view"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"subsonic-response": map[string]any{
					"status": "ok",
					"scanStatus": map[string]any{
						"scanning":    false,
						"count":       42,
						"folderCount": 2,
						"lastScan":    "2026-07-01",
					},
				},
			})
		case strings.Contains(r.URL.Path, "/rest/getArtists.view"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"subsonic-response": map[string]any{
					"status": "ok",
					"artists": map[string]any{
						"index": []map[string]any{
							{
								"name": "A",
								"artist": []map[string]any{
									{"id": "1", "name": "Artist One", "albumCount": 3},
									{"id": "2", "name": "Artist Two", "albumCount": 5},
								},
							},
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "alice", "secret")
	stats, err := client.LibraryStats()
	if err != nil {
		t.Fatalf("LibraryStats: %v", err)
	}
	if stats.SongCount != 42 || stats.FolderCount != 2 {
		t.Fatalf("unexpected scan stats %+v", stats)
	}
	if stats.ArtistCount != 2 || stats.AlbumCount != 8 {
		t.Fatalf("unexpected artist/album counts %+v", stats)
	}
}

func TestShouldCacheRequestSubsonic(t *testing.T) {
	if ShouldCacheRequest(http.MethodGet, "getAlbum.view") != true {
		t.Fatal("expected album view to be cacheable")
	}
	if ShouldCacheRequest(http.MethodGet, "stream.view") != false {
		t.Fatal("expected stream to bypass cache")
	}
	if ShouldCacheRequest(http.MethodGet, "getCoverArt.view") != false {
		t.Fatal("expected cover art to bypass response cache")
	}
	if ShouldCacheRequest(http.MethodGet, "getPlaylist.view") != false {
		t.Fatal("expected playlist detail to bypass cache")
	}
	if ShouldCacheRequest(http.MethodGet, "getPlaylists.view") != true {
		t.Fatal("expected playlist list to remain cacheable")
	}
	if ShouldCacheRequest(http.MethodPost, "getAlbum.view") != false {
		t.Fatal("expected non-GET to bypass cache")
	}
}
