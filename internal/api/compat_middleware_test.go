// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"melovian/internal/compat"
)

func TestCompatMiddlewareAllowsCurrentClient(t *testing.T) {
	called := false
	handler := CompatMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	req.Header.Set(compat.HeaderClientVersion, compat.Version)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if !called {
		t.Fatal("expected next handler to run")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if got := rec.Header().Get(compat.HeaderServerVersion); got != compat.Version {
		t.Fatalf("server version header %q", got)
	}
}

func TestCompatMiddlewareRejectsOldClient(t *testing.T) {
	handler := CompatMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next should not run")
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	req.Header.Set(compat.HeaderClientVersion, "0.0.1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUpgradeRequired {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestCompatMiddlewareAllowsMissingClientVersion(t *testing.T) {
	called := false
	handler := CompatMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if !called {
		t.Fatal("expected next handler")
	}
}
