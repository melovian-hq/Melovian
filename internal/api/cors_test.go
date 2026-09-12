// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"melovian/internal/api/apishared"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCORSMiddlewareAllowsWailsMobileOrigin(t *testing.T) {
	ConfigureCORSOrigins(nil)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CORSMiddleware(inner)

	req := httptest.NewRequest(http.MethodOptions, "/api/config", nil)
	req.Header.Set("Origin", "https://wails.localhost")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://wails.localhost" {
		t.Fatalf("ACAO = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("ACAC = %q", got)
	}
}

func TestCORSMiddlewareRejectsUnknownOrigin(t *testing.T) {
	ConfigureCORSOrigins(nil)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CORSMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected ACAO %q", got)
	}
}

func TestSessionCookieSameSiteNoneForMobileOrigin(t *testing.T) {
	ConfigureCORSOrigins(nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.Header.Set("Origin", "https://wails.localhost")
	req.Header.Set("X-Forwarded-Proto", "https")
	apishared.SetHTTPOnlyCookie(rec, req, "melovian_session", "tok", "/", 3600, time.Time{})
	setCookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "SameSite=None") {
		t.Fatalf("Set-Cookie missing SameSite=None: %q", setCookie)
	}
	if !strings.Contains(strings.ToLower(setCookie), "secure") {
		t.Fatalf("Set-Cookie missing Secure: %q", setCookie)
	}
}

func TestSessionCookieLaxOnHTTPEvenWithMobileOrigin(t *testing.T) {
	ConfigureCORSOrigins(nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.Header.Set("Origin", "https://wails.localhost")
	apishared.SetHTTPOnlyCookie(rec, req, "melovian_session", "tok", "/", 3600, time.Time{})
	setCookie := rec.Header().Get("Set-Cookie")
	if strings.Contains(setCookie, "SameSite=None") {
		t.Fatalf("HTTP responses must not force SameSite=None: %q", setCookie)
	}
}
