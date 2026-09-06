// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDemoAcceptanceAuthStatusFlags(t *testing.T) {
	srv, _ := newDemoTestServer(t)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("auth status %d", rec.Code)
	}
	var status map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status["demoMode"] != true {
		t.Fatalf("expected demoMode true, got %+v", status)
	}
	if status["enabled"] != false {
		t.Fatalf("expected enabled false in demo mode, got %+v", status)
	}
}

func TestDemoAcceptanceBlocksPlaylistAndFavoriteMutations(t *testing.T) {
	srv, _ := newDemoTestServer(t)
	handler := srv.Handler()

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"create playlist", http.MethodPost, "/api/music/playlists", `{"name":"Demo Blocked"}`},
		{"add favorite", http.MethodPost, "/api/music/favorites/fav-1", `{"trackTitle":"Song","artistName":"Artist"}`},
		{"remove favorite", http.MethodDelete, "/api/music/favorites/fav-1", ``},
		{"delete playlist", http.MethodDelete, "/api/music/playlists/pl-1", ``},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var body *bytes.Buffer
			if tc.body != "" {
				body = bytes.NewBufferString(tc.body)
			} else {
				body = bytes.NewBuffer(nil)
			}
			req := httptest.NewRequest(tc.method, tc.path, body)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
			}
			var payload map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode error body: %v", err)
			}
			if payload["error"] == nil {
				t.Fatalf("expected error field in response: %+v", payload)
			}
		})
	}
}
