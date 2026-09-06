// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"melovian/internal/store"
)

func TestSmartPlaylistSupportAndCreate(t *testing.T) {
	var playlistBody map[string]any
	ndSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/login":
			_, _ = w.Write([]byte(`{"token":"jwt-token"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/playlist":
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &playlistBody)
			_, _ = w.Write([]byte(`{"id":"smart-1","name":"Evening jazz"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ndSrv.Close()
	subSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<?xml version="1.0"?><subsonic-response status="ok" version="1.16.1"><serverVersion>0.58.0</serverVersion></subsonic-response>`))
	}))
	defer subSrv.Close()
	srv, db := newTestServer(t)
	instances := store.NewInstanceStore(db)
	inst, err := instances.Create(store.CreateInstanceInput{
		Name:       "Navidrome",
		ServerURL:  ndSrv.URL,
		Username:   "user",
		Password:   "pass",
		ServerName: "Navidrome",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := instances.SetActive(inst.ID); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	if err := srv.reloadActiveSubsonic(); err != nil {
		t.Fatalf("reloadActiveSubsonic: %v", err)
	}
	_ = subSrv

	req := httptest.NewRequest(http.MethodGet, "/api/music/smart-playlists/support", nil)
	req.Header.Set("X-Instance-Id", inst.ID)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("support status %d: %s", rec.Code, rec.Body.String())
	}
	var support struct {
		Supported bool   `json:"supported"`
		Mode      string `json:"mode"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &support); err != nil {
		t.Fatalf("support json: %v", err)
	}
	if !support.Supported || support.Mode != "navidrome" {
		t.Fatalf("expected navidrome smart playlist support, got %#v", support)
	}

	payload := `{
		"name": "Evening jazz",
		"rules": {
			"all": [{"contains": {"genre": "Jazz"}}],
			"sort": "+random"
		}
	}`
	req = httptest.NewRequest(http.MethodPost, "/api/music/smart-playlists", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Instance-Id", inst.ID)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status %d: %s", rec.Code, rec.Body.String())
	}
	if playlistBody["name"] != "Evening jazz" {
		t.Fatalf("unexpected playlist payload: %#v", playlistBody)
	}
}

func TestSmartPlaylistCreateRejectsEmptyName(t *testing.T) {
	srv, db := newTestServer(t)
	instances := store.NewInstanceStore(db)
	inst, err := instances.Create(store.CreateInstanceInput{
		Name:       "Navidrome",
		ServerURL:  "http://example.com",
		Username:   "user",
		Password:   "pass",
		ServerName: "Navidrome",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := instances.SetActive(inst.ID); err != nil {
		t.Fatalf("SetActive: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/music/smart-playlists",
		strings.NewReader(`{"name":"  ","rules":{"all":[{"contains":{"genre":"Jazz"}}]}}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Instance-Id", inst.ID)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
