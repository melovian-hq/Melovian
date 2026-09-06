// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"melovian/internal/httputil"
)

func TestRequestIDMiddlewarePreservesIncomingID(t *testing.T) {
	var got string
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = httputil.RequestIDFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(httputil.RequestIDHeader, "client-trace-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got != "client-trace-123" {
		t.Fatalf("context request id = %q, want client-trace-123", got)
	}
	if rec.Header().Get(httputil.RequestIDHeader) != "client-trace-123" {
		t.Fatalf("response header = %q, want client-trace-123", rec.Header().Get(httputil.RequestIDHeader))
	}
}

func TestRequestIDMiddlewareGeneratesID(t *testing.T) {
	var got string
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = httputil.RequestIDFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got == "" {
		t.Fatal("expected generated request id in context")
	}
	if rec.Header().Get(httputil.RequestIDHeader) != got {
		t.Fatalf("response header = %q, context id = %q", rec.Header().Get(httputil.RequestIDHeader), got)
	}
}
