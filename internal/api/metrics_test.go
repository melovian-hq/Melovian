// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"melovian/internal/api/apishared"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsEndpoint(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "melovian_http_requests_total") {
		t.Fatal("expected melovian_http_requests_total metric")
	}
	if !strings.Contains(body, "go_goroutines") {
		t.Fatal("expected default prometheus metrics")
	}
}

func TestMetricsPathPublic(t *testing.T) {
	if !apishared.IsPublicAPIPath("/metrics") {
		t.Fatal("expected /metrics to be public")
	}
	if !isAPIPath("/metrics") {
		t.Fatal("expected /metrics to be routed to API handler")
	}
}
