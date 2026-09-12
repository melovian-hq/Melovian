// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"melovian/internal/httputil"
	"melovian/internal/melog"
	"melovian/internal/observability"
)

const maxClientLogBody = 64 << 10
const maxClientLogFile = 8 << 20

type clientLogRequest struct {
	Level     string `json:"level"`
	Message   string `json:"message"`
	Source    string `json:"source"`
	Stack     string `json:"stack"`
	URL       string `json:"url"`
	Time      string `json:"time"`
	RequestID string `json:"requestId"`
}

func (s *Server) registerClientLogRoutes() {
	s.mux.HandleFunc("POST /api/client-log", s.handleClientLog)
}

func (s *Server) handleClientLog(w http.ResponseWriter, r *http.Request) {
	// This endpoint is reachable before login and writes to disk, so it gets
	// a per-IP request budget to blunt unauthenticated log flooding.
	if s.clientLogLimiter != nil {
		key := "clientlog"
		if addr, ok := clientIP(r, s.cfg.TrustProxy); ok {
			key += "|" + addr.String()
		}
		if s.clientLogLimiter.blocked(key) {
			writeRateLimited(w, s.clientLogLimiter.retryAfterSeconds(key))
			return
		}
		s.clientLogLimiter.record(key)
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxClientLogBody)

	var req clientLogRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	req.Level = strings.ToLower(strings.TrimSpace(req.Level))
	if req.Level == "" {
		req.Level = "error"
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		httputil.WriteError(w, http.StatusBadRequest, "message_required", "message required")
		return
	}
	if req.Time == "" {
		req.Time = time.Now().UTC().Format(time.RFC3339)
	}
	requestID := strings.TrimSpace(req.RequestID)
	if requestID == "" {
		requestID = strings.TrimSpace(r.Header.Get(httputil.RequestIDHeader))
	}
	if requestID == "" {
		requestID = httputil.RequestIDFromContext(r.Context())
	}

	entry := melog.ClientLogEntry{
		Level:     req.Level,
		Message:   req.Message,
		Source:    truncate(req.Source, 256),
		Stack:     truncate(req.Stack, 32<<10),
		URL:       truncate(req.URL, 2048),
		Time:      req.Time,
		RequestID: requestID,
	}

	s.appendClientLog(entry)
	s.logClientEntry(entry, requestID)
	observability.CaptureClientLog(req.Level, req.Message, req.Source, req.Stack, req.URL, requestID)

	if req.Level == "error" && req.Stack != "" {
		stack := []byte(req.Stack)
		if path, err := melog.DumpCrash("client", req.Message, stack); err == nil {
			melog.Default().Info("client crash report saved", "crash_file", path)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) appendClientLog(entry melog.ClientLogEntry) {
	path := filepath.Join(s.cfg.DataDir, "logs", "client.log")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return
	}
	if info, err := os.Stat(path); err == nil && info.Size() > maxClientLogFile {
		_ = os.Rename(path, path+".old")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600) //#nosec G304 -- path is under configured data dir
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_ = melog.WriteClientLog(f, entry)
}

func (s *Server) logClientEntry(entry melog.ClientLogEntry, requestID string) {
	level := slog.LevelInfo
	switch entry.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	attrs := []any{
		"client_source", melog.Sanitize(entry.Source),
		"client_url", melog.Sanitize(entry.URL),
		"client_time", melog.Sanitize(entry.Time),
		"message", melog.Sanitize(entry.Message),
	}
	if requestID != "" {
		attrs = append(attrs, "request_id", melog.Sanitize(requestID))
	}
	melog.Default().Log(context.TODO(), level, "client log", attrs...)
	if entry.Stack != "" && level >= slog.LevelWarn {
		stackAttrs := []any{"stack", melog.Sanitize(entry.Stack)}
		if requestID != "" {
			stackAttrs = append(stackAttrs, "request_id", requestID)
		}
		melog.Default().Log(context.TODO(), level, "client stack", stackAttrs...)
	}
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
