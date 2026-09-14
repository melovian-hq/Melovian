// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLibraryPathAllowsDotDotInDirectoryNameOracle(t *testing.T) {
	srv, _ := newTestServerWithLocalLibrary(t)
	// Guarantee: after Clean, a directory whose name contains ".." is still valid.
	// A substring ban on ".." falsely rejects real folders like "artist..band".
	dir := filepath.Join(t.TempDir(), "artist..band")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	got, err := srv.resolveLibraryPath(dir)
	if err != nil {
		t.Fatalf("resolveLibraryPath(%q): %v", dir, err)
	}
	// EvalSymlinks matches the resolver: on macOS t.TempDir() lives under
	// /var which is a symlink to /private/var.
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", dir, err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveLibraryPathRejectsRelativeOracle(t *testing.T) {
	srv, _ := newTestServerWithLocalLibrary(t)
	if _, err := srv.resolveLibraryPath("relative/music"); err == nil {
		t.Fatal("expected relative path rejection")
	}
}
