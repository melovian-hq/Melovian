// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package observability

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"

	"melovian/internal/appconfig"
	"melovian/internal/brand"
)

var (
	mu      sync.RWMutex
	enabled bool
)

func InitSentry(cfg appconfig.SentryConfig) error {
	mu.Lock()
	defer mu.Unlock()

	if !cfg.Enabled() {
		enabled = false
		return nil
	}

	opts := sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		TracesSampleRate: cfg.TracesSampleRate,
		AttachStacktrace: true,
	}
	if err := sentry.Init(opts); err != nil {
		enabled = false
		return fmt.Errorf("sentry init: %w", err)
	}
	enabled = true
	return nil
}

func ReinitSentry(cfg appconfig.SentryConfig) error {
	if Enabled() {
		sentry.Flush(2 * time.Second)
	}
	return InitSentry(cfg)
}

func Enabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return enabled
}

func Flush(timeout time.Duration) {
	if !Enabled() {
		return
	}
	sentry.Flush(timeout)
}

func HTTPMiddleware() func(http.Handler) http.Handler {
	sentryHandler := sentryhttp.New(sentryhttp.Options{Repanic: true})
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !Enabled() {
				next.ServeHTTP(w, r)
				return
			}
			sentryHandler.Handle(next).ServeHTTP(w, r)
		})
	}
}

func CapturePanic(recovered any, tags map[string]string) {
	if !Enabled() {
		return
	}
	hub := sentry.CurrentHub().Clone()
	for key, value := range tags {
		hub.Scope().SetTag(key, value)
	}
	hub.Recover(recovered)
}

// CaptureError reports a backend error to Sentry when tracking is enabled.
func CaptureError(err error, tags map[string]string) {
	if !Enabled() || err == nil {
		return
	}
	hub := sentry.CurrentHub().Clone()
	for key, value := range tags {
		hub.Scope().SetTag(key, value)
	}
	hub.CaptureException(err)
}

func CaptureClientLog(level, message, source, stack, url, requestID string) {
	if !Enabled() {
		return
	}
	if level != "error" && level != "warn" && level != "warning" {
		return
	}

	hub := sentry.CurrentHub().Clone()
	hub.Scope().SetTag("client_source", source)
	if url != "" {
		hub.Scope().SetTag("client_url", url)
	}
	if requestID != "" {
		hub.Scope().SetTag("request_id", requestID)
	}
	if stack != "" {
		hub.Scope().SetContext("client", sentry.Context{"stack": stack})
	}
	if level == "error" {
		hub.Scope().SetLevel(sentry.LevelError)
		hub.CaptureMessage(message)
		return
	}
	hub.Scope().SetLevel(sentry.LevelWarning)
	hub.CaptureMessage(message)
}

func CaptureHTTPError(status int, method, path, detail, requestID string) {
	if !Enabled() || status < 500 {
		return
	}
	hub := sentry.CurrentHub().Clone()
	hub.Scope().SetTag("http_method", method)
	hub.Scope().SetTag("http_path", path)
	hub.Scope().SetTag("http_status", fmt.Sprintf("%d", status))
	if requestID != "" {
		hub.Scope().SetTag("request_id", requestID)
	}
	if detail != "" {
		hub.Scope().SetContext("http", sentry.Context{"detail": detail})
	}
	hub.CaptureMessage(fmt.Sprintf("HTTP %d %s %s", status, method, path))
}

// SendTestEvent captures a settings probe message and flushes the transport.
// Returns the Sentry event id when the event was queued and flushed in time.
func SendTestEvent() (string, error) {
	if !Enabled() {
		return "", fmt.Errorf("error tracking is not active")
	}
	hub := sentry.CurrentHub().Clone()
	hub.Scope().SetTag("client_source", "settings-test")
	hub.Scope().SetLevel(sentry.LevelError)
	eventID := hub.CaptureMessage(brand.Name + " settings test event")
	if eventID == nil {
		return "", fmt.Errorf("failed to queue test event")
	}
	if !sentry.Flush(5 * time.Second) {
		return "", fmt.Errorf("timed out flushing test event to the tracker")
	}
	return string(*eventID), nil
}
