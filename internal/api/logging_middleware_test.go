// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"melovian/internal/melog"
)

func TestRecoverMiddleware(t *testing.T) {
	if _, err := melog.Init(t.TempDir()); err != nil {
		t.Fatal(err)
	}

	handler := RecoverMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "internal server error") {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestHandleClientLog(t *testing.T) {
	srv, _ := newTestServer(t)
	if _, err := melog.Init(srv.cfg.DataDir); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/client-log", strings.NewReader(`{
		"level":"error",
		"message":"test failure",
		"source":"unit-test",
		"stack":"Error: test\n    at foo",
		"url":"http://localhost/music",
		"time":"2026-06-23T12:00:00Z"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", rec.Code, rec.Body.String())
	}

	data, err := os.ReadFile(srv.cfg.DataDir + "/logs/client.log")
	if err != nil {
		t.Fatalf("expected client.log: %v", err)
	}
	if !strings.Contains(string(data), "test failure") {
		t.Fatalf("client.log missing message: %q", data)
	}
}

func TestNormalizePublicBaseURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"http://0.0.0.0:8080/", "http://127.0.0.1:8080"},
		{"http://[::]:8080", "http://127.0.0.1:8080"},
		{"0.0.0.0:8080", "http://127.0.0.1:8080"},
		{"[::]:8080", "http://127.0.0.1:8080"},
		{"https://music.example/", "https://music.example"},
		{"", ""},
	}
	for _, tc := range cases {
		got := normalizePublicBaseURL(tc.in)
		if got != tc.want {
			t.Fatalf("normalizePublicBaseURL(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestHTTPStatusLogLevel(t *testing.T) {
	cases := []struct {
		status int
		want   slog.Level
	}{
		{http.StatusOK, slog.LevelDebug},
		{http.StatusUnauthorized, slog.LevelDebug},
		{http.StatusForbidden, slog.LevelDebug},
		{http.StatusNotFound, slog.LevelWarn},
		{http.StatusBadRequest, slog.LevelWarn},
		{http.StatusInternalServerError, slog.LevelError},
	}
	for _, tc := range cases {
		got := httpStatusLogLevel(tc.status)
		if got != tc.want {
			t.Fatalf("status %d: got %v want %v", tc.status, got, tc.want)
		}
	}
}
