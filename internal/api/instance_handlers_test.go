// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTestInstanceEmptyBody(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/instances/test", http.NoBody)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Connected bool   `json:"connected"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Connected {
		t.Fatal("expected connected false")
	}
	if !strings.Contains(payload.Error, "empty request body") {
		t.Fatalf("error %q", payload.Error)
	}
}

func TestTestInstancePingsServer(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/rest/ping.view") {
			t.Fatalf("path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"subsonic-response":{"status":"ok","version":"1.16.1","type":"navidrome","serverVersion":"0.54.0"}}`))
	}))
	defer upstream.Close()
	srv, _ := newTestServer(t)
	body := `{"name":"Home","serverUrl":"` + upstream.URL + `","username":"u","password":"p"}`
	req := httptest.NewRequest(http.MethodPost, "/api/instances/test", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Connected  bool   `json:"connected"`
		ServerName string `json:"serverName"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !payload.Connected {
		t.Fatalf("expected connected: %s", rec.Body.String())
	}
	if payload.ServerName != "navidrome" {
		t.Fatalf("serverName %q", payload.ServerName)
	}
}
