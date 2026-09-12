// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"melovian/internal/appconfig"
	"melovian/internal/store"
)

func newTestServerWithPprof(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	cfg := appconfig.Config{
		DataDir:      dir,
		DatabasePath: filepath.Join(dir, "test.db"),
		ListenAddr:   "127.0.0.1:0",
		CacheEnabled: true,
		DebugPprof:   true,
	}
	db, err := store.OpenDB(cfg.DatabasePath, cfg)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewServer(cfg, db)
}

func TestDebugPprofDisabledByDefault(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when pprof disabled, got %d", rec.Code)
	}
}

func TestDebugPprofEnabledServesIndex(t *testing.T) {
	srv := newTestServerWithPprof(t)
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for pprof index, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDebugMemoryReportsCacheFields(t *testing.T) {
	srv := newTestServerWithPprof(t)
	req := httptest.NewRequest(http.MethodGet, "/api/debug/memory", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}

	var payload struct {
		DebugPprof bool     `json:"debugPprof"`
		Notes      []string `json:"notes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !payload.DebugPprof {
		t.Fatal("expected debugPprof true")
	}
	if len(payload.Notes) == 0 {
		t.Fatal("expected profiling notes")
	}
}

func TestMetricsExposeCacheGauges(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	body := rec.Body.String()
	for _, needle := range []string{
		"melovian_cache_entries",
		"melovian_cache_bytes",
		"melovian_ws_clients",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("metrics missing %q", needle)
		}
	}
}
