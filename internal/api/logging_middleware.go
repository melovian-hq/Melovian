// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"melovian/internal/httputil"
	"melovian/internal/melog"
	"melovian/internal/observability"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

func (r *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return delegateHijack(r.ResponseWriter)
}

func (r *responseRecorder) Flush() {
	_ = delegateFlush(r.ResponseWriter)
}

func (r *responseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		level := httpStatusLogLevel(rec.status)

		// Health checks and static assets are noisy at the default server log level.
		path := melog.Sanitize(metricPath(r.URL.Path))
		if level == slog.LevelDebug && (path == "/health" || strings.HasPrefix(path, "/assets/")) {
			return
		}

		attrs := []any{
			"method", r.Method,
			"path", path,
			"status", rec.status,
			"bytes", rec.bytes,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", melog.Sanitize(r.RemoteAddr),
		}
		if requestID := httputil.RequestIDFromContext(r.Context()); requestID != "" {
			attrs = append(attrs, "request_id", requestID)
		}
		melog.Default().Log(r.Context(), level, "http request", attrs...)
		if rec.status >= 500 {
			observability.CaptureHTTPError(
				rec.status,
				r.Method,
				path,
				"",
				httputil.RequestIDFromContext(r.Context()),
			)
		}
	})
}

// httpStatusLogLevel maps response codes to log levels.
// Auth challenges and forbidden responses are expected during login flows
// so they stay at debug. Other client errors are warnings. Server faults are errors.
func httpStatusLogLevel(status int) slog.Level {
	switch {
	case status >= 500:
		return slog.LevelError
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return slog.LevelDebug
	case status >= 400:
		return slog.LevelWarn
	default:
		return slog.LevelDebug
	}
}

func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				stack := debug.Stack()
				path, dumpErr := melog.DumpCrash("http-panic", fmtPanic(recovered, r), stack)
				requestID := httputil.RequestIDFromContext(r.Context())
				attrs := []any{
					"panic", recovered,
					"method", r.Method,
					"path", melog.Sanitize(r.URL.Path),
					"crash_file", path,
					"dump_err", dumpErr,
				}
				if requestID != "" {
					attrs = append(attrs, "request_id", requestID)
				}
				melog.Default().Error("handler panic", attrs...)
				tags := map[string]string{
					"kind":   "http-panic",
					"method": r.Method,
					"path":   metricPath(r.URL.Path),
				}
				if requestID != "" {
					tags["request_id"] = requestID
				}
				observability.CapturePanic(recovered, tags)
				observability.Flush(2 * time.Second)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func fmtPanic(recovered any, r *http.Request) string {
	return fmt.Sprintf("%v method=%s path=%s", recovered, r.Method, r.URL.Path)
}
