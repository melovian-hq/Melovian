// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package observability

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"melovian/internal/brand"
	"melovian/internal/compat"
)

// OTEL support is opt-in and config-free to leave off: tracing starts only
// when an OTLP endpoint is configured or MELOVIAN_OTEL_ENABLED=true asks for
// the default localhost collector. Standard OTEL_* environment variables do
// the configuration, so deployments can point at any OTLP collector without
// new app settings:
//
//	OTEL_EXPORTER_OTLP_ENDPOINT       base endpoint, for example http://collector:4318
//	OTEL_EXPORTER_OTLP_TRACES_ENDPOINT overrides the traces path
//	OTEL_EXPORTER_OTLP_HEADERS        extra headers, key=value comma list
//	OTEL_TRACES_SAMPLER_ARG           traceidratio in [0,1], default 0.05
//	OTEL_SDK_DISABLED                 true disables everything
//	MELOVIAN_OTEL_ENABLED             true forces on with the default endpoint
//
// Only HTTP server spans are emitted. No user data is attached.

var (
	otelMu    sync.RWMutex
	otelState struct {
		enabled bool
		tp      *sdktrace.TracerProvider
	}
)

func envTrue(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func otelConfigured() bool {
	if envTrue("OTEL_SDK_DISABLED") {
		return false
	}
	if envTrue("MELOVIAN_OTEL_ENABLED") {
		return true
	}
	return strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")) != "" ||
		strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")) != ""
}

func otelSampleRatio() float64 {
	raw := strings.TrimSpace(os.Getenv("OTEL_TRACES_SAMPLER_ARG"))
	if raw == "" {
		return 0.05
	}
	ratio, err := strconv.ParseFloat(raw, 64)
	if err != nil || ratio < 0 {
		return 0
	}
	if ratio > 1 {
		return 1
	}
	return ratio
}

// InitOTel installs a global tracer provider when an OTLP endpoint is
// configured. With no endpoint it leaves the global no-op provider in place
// and returns nil.
func InitOTel(ctx context.Context) error {
	otelMu.Lock()
	defer otelMu.Unlock()
	if !otelConfigured() {
		otelState.enabled = false
		return nil
	}

	opts := []otlptracehttp.Option{}
	// MELOVIAN_OTEL_ENABLED alone targets the conventional local collector.
	if strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")) == "" &&
		strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")) == "" {
		opts = append(opts,
			otlptracehttp.WithEndpoint("localhost:4318"),
			otlptracehttp.WithInsecure(),
		)
	}
	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return fmt.Errorf("otel exporter: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(brand.Slug),
			semconv.ServiceVersion(compat.Version),
		),
	)
	if err != nil {
		res = resource.Default()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(
			sdktrace.ParentBased(sdktrace.TraceIDRatioBased(otelSampleRatio())),
		),
	)
	otel.SetTracerProvider(tp)
	otelState.tp = tp
	otelState.enabled = true
	return nil
}

// OTelEnabled reports whether the tracer provider is exporting.
func OTelEnabled() bool {
	otelMu.RLock()
	defer otelMu.RUnlock()
	return otelState.enabled
}

// OTelHTTPMiddleware wraps next with server spans when OTEL is enabled.
// Spans carry route, method, and status only, no headers or bodies.
func OTelHTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// otelhttp resolves the global provider per request, so building the
		// traced handler once still picks up a provider enabled later.
		traced := otelhttp.NewHandler(next, "http.request",
			otelhttp.WithSpanNameFormatter(func(_ string, req *http.Request) string {
				return req.Method + " " + req.URL.Path
			}),
		)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !OTelEnabled() {
				next.ServeHTTP(w, r)
				return
			}
			traced.ServeHTTP(w, r)
		})
	}
}

// ShutdownOTel flushes and stops the provider. Safe when never initialized.
func ShutdownOTel(ctx context.Context) {
	otelMu.Lock()
	tp := otelState.tp
	otelState.tp = nil
	otelState.enabled = false
	otelMu.Unlock()
	if tp == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_ = tp.Shutdown(ctx)
}
