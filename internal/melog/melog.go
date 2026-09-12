// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package melog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"melovian/internal/brand"
)

var (
	mu     sync.RWMutex
	logger *slog.Logger
	logDir string
)

type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, h := range m.handlers {
		if !h.Enabled(ctx, r.Level) {
			continue
		}
		if err := h.Handle(ctx, r.Clone()); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		next[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: next}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		next[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: next}
}

// Sanitize strips CR and LF from a value before it is written to a
// line-oriented log so a user-controlled string cannot forge log lines.
func Sanitize(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	return strings.ReplaceAll(s, "\n", "")
}

func ParseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func Init(dataDir string) (*slog.Logger, error) {
	logDir = filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(filepath.Join(logDir, "crashes"), 0o750); err != nil {
		return nil, fmt.Errorf("create log dirs: %w", err)
	}

	level := ParseLevel(os.Getenv("MELOVIAN_LOG_LEVEL"))
	opts := &slog.HandlerOptions{Level: level, AddSource: true}

	logPath := envOr("MELOVIAN_LOG_FILE", filepath.Join(logDir, brand.Slug+".log"))
	handlers := []slog.Handler{slog.NewTextHandler(os.Stderr, opts)}
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600) //#nosec G304 -- log path is under configured data dir
	if err == nil {
		handlers = append(handlers, slog.NewTextHandler(file, opts))
	} else {
		fmt.Fprintf(os.Stderr, "%s: file logging disabled (%v), using stderr only\n", brand.Slug, err)
	}

	l := slog.New(&multiHandler{handlers: handlers})

	mu.Lock()
	logger = l
	mu.Unlock()

	slog.SetDefault(l)
	l.Info("logging initialized", "log_file", logPath, "level", level.String())
	mainLog, clientLog, crashDir := LogPaths(dataDir)
	l.Info("log paths", "main", mainLog, "client", clientLog, "crashes", crashDir)
	return l, nil
}

func Default() *slog.Logger {
	mu.RLock()
	l := logger
	mu.RUnlock()
	if l != nil {
		return l
	}
	return slog.Default()
}

func LogDir() string {
	return logDir
}

func LogPaths(dataDir string) (mainLog, clientLog, crashDir string) {
	dir := filepath.Join(dataDir, "logs")
	return envOr("MELOVIAN_LOG_FILE", filepath.Join(dir, "melovian.log")),
		filepath.Join(dir, "client.log"),
		filepath.Join(dir, "crashes")
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func WriteClientLog(w io.Writer, entry ClientLogEntry) error {
	requestID := entry.RequestID
	if requestID == "" {
		requestID = "-"
	}
	_, err := fmt.Fprintf(w,
		"time=%s level=%s source=%s url=%s request_id=%s message=%q\nstack=%s\n---\n",
		entry.Time,
		entry.Level,
		entry.Source,
		entry.URL,
		requestID,
		entry.Message,
		entry.Stack,
	)
	return err
}

type ClientLogEntry struct {
	Level     string
	Message   string
	Source    string
	Stack     string
	URL       string
	Time      string
	RequestID string
}
