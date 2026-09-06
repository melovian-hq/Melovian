// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build linux && !android

package libmpv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSystemLibmpvWhenPresent(t *testing.T) {
	if _, err := os.Stat("/usr/lib/libmpv.so.2"); err != nil {
		t.Skip("system libmpv not installed")
	}
	if !Available() {
		t.Fatalf("expected libmpv to load, got: %v", LoadError())
	}
	player, err := NewPlayer()
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	player.Close()
}

func TestLibmpvCandidatesIncludeBundledAppDir(t *testing.T) {
	appDir := t.TempDir()
	libDir := filepath.Join(appDir, "usr", "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	bundled := filepath.Join(libDir, "libmpv.so.2")
	if err := os.WriteFile(bundled, []byte("stub"), 0o644); err != nil {
		t.Fatalf("write stub: %v", err)
	}

	t.Setenv("APPDIR", appDir)

	candidates := libmpvCandidates()
	if !containsPath(candidates, bundled) {
		t.Fatalf("expected candidates to include bundled %q, got %v", bundled, candidates)
	}
	if idx := indexOfPath(candidates, bundled); idx != 0 {
		t.Fatalf("expected bundled libmpv to be preferred first, found at index %d in %v", idx, candidates)
	}
}

func containsPath(paths []string, target string) bool {
	return indexOfPath(paths, target) >= 0
}

func indexOfPath(paths []string, target string) int {
	resolved := target
	if r, err := filepath.EvalSymlinks(target); err == nil && r != "" {
		resolved = r
	}
	for i, p := range paths {
		if p == target || p == resolved {
			return i
		}
	}
	return -1
}
