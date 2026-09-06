// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metaloader_test

import (
	"os"
	"path/filepath"
	"testing"

	"melovian/internal/metaloader"
	"melovian/internal/store"
)

func TestScannerDetectsNewRemovedAndDuplicates(t *testing.T) {
	db := store.OpenTestDB(t)
	libraries := store.NewLocalLibraryStore(db)
	tracks := store.NewLocalTrackStore(db)
	scanner := metaloader.NewScanner(libraries, tracks)

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "one.mp3"), []byte("track-one"), 0o644); err != nil {
		t.Fatalf("write one: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "two.mp3"), []byte("track-two"), 0o644); err != nil {
		t.Fatalf("write two: %v", err)
	}

	lib, err := libraries.Create(store.CreateLocalLibraryInput{Name: "Test", Path: root})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	result, err := scanner.Scan(lib.ID)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.Added != 2 || result.Total != 2 {
		t.Fatalf("unexpected first scan: %+v", result)
	}

	if err := os.Remove(filepath.Join(root, "two.mp3")); err != nil {
		t.Fatalf("remove two: %v", err)
	}
	result, err = scanner.Scan(lib.ID)
	if err != nil {
		t.Fatalf("rescan: %v", err)
	}
	if result.Removed != 1 {
		t.Fatalf("expected 1 removed track, got %+v", result)
	}

	dupPath := filepath.Join(root, "dup.mp3")
	if err := os.WriteFile(dupPath, []byte("track-one"), 0o644); err != nil {
		t.Fatalf("write dup: %v", err)
	}
	result, err = scanner.Scan(lib.ID)
	if err != nil {
		t.Fatalf("dup scan: %v", err)
	}
	if result.Duplicates == 0 {
		t.Fatalf("expected duplicate detection, got %+v", result)
	}
}

func TestScannerSkipsSymlinkAudioFiles(t *testing.T) {
	db := store.OpenTestDB(t)
	libraries := store.NewLocalLibraryStore(db)
	tracks := store.NewLocalTrackStore(db)
	scanner := metaloader.NewScanner(libraries, tracks)

	root := t.TempDir()
	realFile := filepath.Join(root, "real.mp3")
	if err := os.WriteFile(realFile, []byte("track-real"), 0o644); err != nil {
		t.Fatalf("write real: %v", err)
	}
	outside := filepath.Join(t.TempDir(), "secret.mp3")
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link.mp3")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	lib, err := libraries.Create(store.CreateLocalLibraryInput{Name: "Test", Path: root})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	result, err := scanner.Scan(lib.ID)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.Added != 1 || result.Total != 1 {
		t.Fatalf("expected symlink skipped, got %+v", result)
	}
}
