// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func clearOTelEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"OTEL_SDK_DISABLED",
		"MELOVIAN_OTEL_ENABLED",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		"OTEL_TRACES_SAMPLER_ARG",
	} {
		t.Setenv(key, "")
	}
}

func TestInitOTelDisabledByDefault(t *testing.T) {
	clearOTelEnv(t)
	if err := InitOTel(context.Background()); err != nil {
		t.Fatalf("InitOTel: %v", err)
	}
	if OTelEnabled() {
		t.Fatal("expected otel disabled without an endpoint")
	}
}

func TestInitOTelSDKDisabledWins(t *testing.T) {
	clearOTelEnv(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	t.Setenv("OTEL_SDK_DISABLED", "true")
	if err := InitOTel(context.Background()); err != nil {
		t.Fatalf("InitOTel: %v", err)
	}
	if OTelEnabled() {
		t.Fatal("expected otel disabled when OTEL_SDK_DISABLED=true")
	}
}

func TestInitOTelEnabledWithEndpoint(t *testing.T) {
	clearOTelEnv(t)
	// otlptracehttp does not connect eagerly, so init succeeds without a
	// running collector.
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	if err := InitOTel(context.Background()); err != nil {
		t.Fatalf("InitOTel: %v", err)
	}
	if !OTelEnabled() {
		t.Fatal("expected otel enabled with an endpoint")
	}
	t.Cleanup(func() {
		otelMu.Lock()
		otelState.enabled = false
		otelState.tp = nil
		otelMu.Unlock()
	})
	ShutdownOTel(context.Background())
	if OTelEnabled() {
		t.Fatal("expected otel disabled after shutdown")
	}
}

func TestOTelSampleRatioBounds(t *testing.T) {
	clearOTelEnv(t)
	if got := otelSampleRatio(); got != 0.05 {
		t.Fatalf("default ratio %v", got)
	}
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "0.5")
	if got := otelSampleRatio(); got != 0.5 {
		t.Fatalf("ratio %v", got)
	}
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "7")
	if got := otelSampleRatio(); got != 1 {
		t.Fatalf("ratio %v", got)
	}
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "garbage")
	if got := otelSampleRatio(); got != 0 {
		t.Fatalf("ratio %v", got)
	}
}

func TestOTelMiddlewarePassesThroughWhenDisabled(t *testing.T) {
	clearOTelEnv(t)
	if err := InitOTel(context.Background()); err != nil {
		t.Fatalf("InitOTel: %v", err)
	}
	called := false
	handler := OTelHTTPMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if !called {
		t.Fatal("handler not called")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d", rec.Code)
	}
}
