// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package osutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPathEscapesRootOracle(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "data", "melovian")
	inside := filepath.Join(root, "lyrics", "cache")
	if err := PathEscapesRoot(root, inside); err != nil {
		t.Fatalf("inside path should stay in jail: %v", err)
	}
	escaped := filepath.Join(root, "..", "..", "tmp", "pwned")
	if err := PathEscapesRoot(root, escaped); err == nil {
		t.Fatal("relative escape must be rejected")
	}
	if err := PathEscapesRoot(root, filepath.Join(string(filepath.Separator), "etc", "passwd")); err == nil {
		t.Fatal("absolute escape must be rejected")
	}
}

func TestResolveInside(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "sub")
	if err := os.MkdirAll(inside, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	got, err := ResolveInside(inside, root)
	if err != nil {
		t.Fatalf("in-root path rejected: %v", err)
	}
	if got != inside {
		t.Fatalf("resolved = %q, want %q", got, inside)
	}

	outside := t.TempDir()
	if _, err := ResolveInside(outside, root); !errors.Is(err, ErrPathOutsideRoots) {
		t.Fatalf("outside path should fail with ErrPathOutsideRoots, got %v", err)
	}

	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if _, err := ResolveInside(link, root); !errors.Is(err, ErrPathOutsideRoots) {
		t.Fatalf("symlink escape should fail with ErrPathOutsideRoots, got %v", err)
	}

	if _, err := ResolveInside(filepath.Join(root, "missing"), root); err == nil {
		t.Fatal("nonexistent path must fail resolution")
	}
	if _, err := ResolveInside(inside, filepath.Join(t.TempDir(), "no-root")); !errors.Is(err, ErrPathOutsideRoots) {
		t.Fatal("path under no root must be rejected")
	}
}
