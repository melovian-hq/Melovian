// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package melog

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		raw  string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"WARN", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"", slog.LevelInfo},
		{"info", slog.LevelInfo},
	}
	for _, tc := range tests {
		if got := ParseLevel(tc.raw); got != tc.want {
			t.Fatalf("ParseLevel(%q) = %v, want %v", tc.raw, got, tc.want)
		}
	}
}

func TestDumpCrash(t *testing.T) {
	dir := t.TempDir()
	logDir = dir

	path, err := DumpCrash("test", "boom", []byte("stack line"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, filepath.Join(dir, "crashes")) {
		t.Fatalf("unexpected crash path %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "Kind: test") || !strings.Contains(content, "stack line") {
		t.Fatalf("unexpected crash file content: %q", content)
	}
}

func TestLogPaths(t *testing.T) {
	mainLog, clientLog, crashDir := LogPaths("/data/melovian")
	if !strings.HasSuffix(mainLog, "melovian.log") {
		t.Fatalf("unexpected main log path %q", mainLog)
	}
	if !strings.HasSuffix(clientLog, "client.log") {
		t.Fatalf("unexpected client log path %q", clientLog)
	}
	if !strings.HasSuffix(crashDir, "crashes") {
		t.Fatalf("unexpected crash dir %q", crashDir)
	}
}

func TestLogPathsRespectEnvOverride(t *testing.T) {
	t.Setenv("MELOVIAN_LOG_FILE", "/tmp/custom.log")
	mainLog, _, _ := LogPaths("/data/melovian")
	if mainLog != "/tmp/custom.log" {
		t.Fatalf("expected env override, got %q", mainLog)
	}
}

func TestInitCreatesLogDirsAndDefaultLogger(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MELOVIAN_LOG_LEVEL", "debug")
	t.Setenv("MELOVIAN_LOG_FILE", "")

	logger, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if logger == nil {
		t.Fatal("expected logger")
	}
	if Default() != logger {
		t.Fatal("expected Default() to return initialized logger")
	}
	if LogDir() != filepath.Join(dir, "logs") {
		t.Fatalf("unexpected log dir %q", LogDir())
	}
	if _, err := os.Stat(filepath.Join(dir, "logs", "crashes")); err != nil {
		t.Fatalf("expected crash dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "logs", "melovian.log")); err != nil {
		t.Fatalf("expected main log file: %v", err)
	}
}

func TestWriteClientLog(t *testing.T) {
	var buf strings.Builder
	entry := ClientLogEntry{
		Time:      "2026-01-01T00:00:00Z",
		Level:     "error",
		Source:    "frontend",
		URL:       "http://localhost/",
		Message:   "boom",
		Stack:     "Error: boom",
		RequestID: "req-abc",
	}
	if err := WriteClientLog(&buf, entry); err != nil {
		t.Fatalf("WriteClientLog: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"time=2026-01-01T00:00:00Z",
		"level=error",
		"source=frontend",
		"url=http://localhost/",
		"request_id=req-abc",
		`message="boom"`,
		"stack=Error: boom",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output %q", want, out)
		}
	}
}

func TestDumpCrashSanitizesKind(t *testing.T) {
	dir := t.TempDir()
	logDir = dir

	path, err := DumpCrash("panic: bad/file", "detail", []byte("stack"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, "panic--bad-file") {
		t.Fatalf("expected sanitized crash filename, got %q", path)
	}
}
