// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package navidrome

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsNavidromeServer(t *testing.T) {
	if !IsNavidromeServer("Navidrome", "0.58.0") {
		t.Fatal("expected navidrome detection")
	}
	if IsNavidromeServer("Subsonic", "1.16.1") {
		t.Fatal("expected non-navidrome server")
	}
}

func TestClientLoginAndCreateSmartPlaylist(t *testing.T) {
	var gotAuth string
	var gotPlaylistBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/login":
			body, _ := io.ReadAll(r.Body)
			var payload map[string]string
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("login body: %v", err)
			}
			if payload["username"] != "user" || payload["password"] != "pass" {
				t.Fatalf("unexpected login payload: %#v", payload)
			}
			_, _ = w.Write([]byte(`{"token":"jwt-token"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/playlist":
			gotAuth = r.Header.Get(authHeader)
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, &gotPlaylistBody); err != nil {
				t.Fatalf("playlist body: %v", err)
			}
			_, _ = w.Write([]byte(`{"id":"pl-smart","name":"Jazz mix"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	client := NewClient(srv.URL, "user", "pass")
	playlist, err := client.CreateSmartPlaylist(t.Context(), CreateSmartPlaylistRequest{
		Name: "Jazz mix",
		Rules: map[string]any{
			"all": []any{
				map[string]any{"contains": map[string]any{"genre": "Jazz"}},
			},
			"sort": "+random",
		},
	})
	if err != nil {
		t.Fatalf("CreateSmartPlaylist: %v", err)
	}
	if playlist.ID != "pl-smart" {
		t.Fatalf("unexpected playlist id: %q", playlist.ID)
	}
	if !strings.Contains(gotAuth, "Bearer jwt-token") {
		t.Fatalf("missing auth header: %q", gotAuth)
	}
	if gotPlaylistBody["name"] != "Jazz mix" {
		t.Fatalf("unexpected playlist name: %#v", gotPlaylistBody["name"])
	}
}

func TestClientLoginUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer srv.Close()
	client := NewClient(srv.URL, "user", "bad")
	_, err := client.CreateSmartPlaylist(t.Context(), CreateSmartPlaylistRequest{
		Name:  "Test",
		Rules: map[string]any{"all": []any{}},
	})
	if err == nil || !strings.Contains(err.Error(), "authentication") {
		t.Fatalf("expected auth error, got %v", err)
	}
}
