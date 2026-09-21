// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package melog

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"melovian/internal/brand"
	"melovian/internal/observability"
)

// maxCrashDumps bounds the crashes directory. Client crash reports are
// accepted from unauthenticated callers, so without a cap repeated
// submissions could fill the disk. Timestamped names sort oldest first.
const maxCrashDumps = 50

func DumpCrash(kind string, detail string, stack []byte) (string, error) {
	if logDir == "" {
		return "", fmt.Errorf("logging not initialized")
	}

	dir := filepath.Join(logDir, "crashes")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}

	ts := time.Now().UTC().Format("20060102-150405")
	name := fmt.Sprintf("crash-%s-%s.log", ts, sanitizeFilename(kind))
	path := filepath.Join(dir, name)

	body := fmt.Sprintf(
		"=== "+brand.Name+" crash report ===\nTime: %s\nKind: %s\nDetail: %s\n\nStack:\n%s\n",
		time.Now().UTC().Format(time.RFC3339Nano),
		kind,
		detail,
		string(stack),
	)

	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return "", err
	}
	pruneCrashDumps(dir)
	return path, nil
}

// pruneCrashDumps evicts the oldest crash reports beyond maxCrashDumps.
// Errors are ignored: pruning is best effort housekeeping.
func pruneCrashDumps(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "crash-") {
			names = append(names, e.Name())
		}
	}
	if len(names) <= maxCrashDumps {
		return
	}
	slices.Sort(names)
	for _, name := range names[:len(names)-maxCrashDumps] {
		_ = os.Remove(filepath.Join(dir, name))
	}
}

func DeferredPanicHandler() {
	if recovered := recover(); recovered != nil {
		stack := debug.Stack()
		observability.CapturePanic(recovered, map[string]string{"kind": "panic"})
		path, err := DumpCrash("panic", fmt.Sprint(recovered), stack)
		Default().Error("fatal panic recovered",
			"panic", recovered,
			"crash_file", path,
			"dump_err", err,
		)
		observability.Flush(2 * time.Second)
		os.Exit(1)
	}
}

func sanitizeFilename(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			out = append(out, c)
		default:
			out = append(out, '-')
		}
	}
	if len(out) == 0 {
		return "unknown"
	}
	return string(out)
}
