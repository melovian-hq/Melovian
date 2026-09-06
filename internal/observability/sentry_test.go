// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package observability

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"melovian/internal/appconfig"
)

func TestInitSentryDisabled(t *testing.T) {
	if err := InitSentry(appconfig.SentryConfig{}); err != nil {
		t.Fatalf("InitSentry: %v", err)
	}
	if Enabled() {
		t.Fatal("expected disabled without DSN")
	}
}

func TestHTTPMiddlewareNoOpWhenDisabled(t *testing.T) {
	if err := InitSentry(appconfig.SentryConfig{}); err != nil {
		t.Fatalf("InitSentry: %v", err)
	}
	called := false
	handler := HTTPMiddleware()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if !called {
		t.Fatal("expected handler to run")
	}
}

func TestCaptureClientLogNoOpWhenDisabled(t *testing.T) {
	if err := InitSentry(appconfig.SentryConfig{}); err != nil {
		t.Fatalf("InitSentry: %v", err)
	}
	CaptureClientLog("error", "boom", "test", "stack", "http://localhost", "req-test")
}

func TestSendTestEventWhenDisabled(t *testing.T) {
	if err := InitSentry(appconfig.SentryConfig{}); err != nil {
		t.Fatalf("InitSentry: %v", err)
	}
	if _, err := SendTestEvent(); err == nil {
		t.Fatal("expected error when sentry disabled")
	}
}

func TestCaptureErrorNoOpWhenDisabled(t *testing.T) {
	if err := InitSentry(appconfig.SentryConfig{}); err != nil {
		t.Fatalf("InitSentry: %v", err)
	}
	CaptureError(fmt.Errorf("boom"), map[string]string{"kind": "test"})
}
