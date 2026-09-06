// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metaloader_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"melovian/internal/metaloader"
	"melovian/internal/store"
)

func newWatchFixture(t *testing.T) (*store.LocalLibraryStore, *store.LocalTrackStore, *metaloader.Scanner, store.LocalLibrary, string) {
	t.Helper()
	db := store.OpenTestDB(t)
	libraries := store.NewLocalLibraryStore(db)
	tracks := store.NewLocalTrackStore(db)
	scanner := metaloader.NewScanner(libraries, tracks)

	root := t.TempDir()
	lib, err := libraries.Create(store.CreateLocalLibraryInput{Name: "Watch", Path: root})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return libraries, tracks, scanner, lib, root
}

func waitForTrackStatus(t *testing.T, tracks *store.LocalTrackStore, libraryID, relPath string, wantPresent bool) store.LocalTrack {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		track, err := tracks.GetByRelPath(libraryID, relPath)
		if err == nil && (track.Status == store.TrackStatusPresent) == wantPresent {
			return track
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s present=%v", relPath, wantPresent)
	return store.LocalTrack{}
}

func TestApplyChangesAddsRemovesAndAdoptsRenames(t *testing.T) {
	_, tracks, scanner, lib, root := newWatchFixture(t)

	if err := os.WriteFile(filepath.Join(root, "one.mp3"), []byte("track-one"), 0o644); err != nil {
		t.Fatalf("write one: %v", err)
	}
	if _, err := scanner.Scan(lib.ID); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	original, err := tracks.GetByRelPath(lib.ID, "one.mp3")
	if err != nil {
		t.Fatalf("GetByRelPath: %v", err)
	}

	// Rename within the library. The missing row is adopted by content hash so
	// the track keeps its identity.
	if err := os.Rename(filepath.Join(root, "one.mp3"), filepath.Join(root, "renamed.mp3")); err != nil {
		t.Fatalf("rename: %v", err)
	}
	err = scanner.ApplyChanges(lib.ID, []metaloader.FileChange{
		{RelPath: "one.mp3", Removed: true},
		{RelPath: "renamed.mp3"},
	})
	if err != nil {
		t.Fatalf("ApplyChanges: %v", err)
	}
	moved, err := tracks.GetByRelPath(lib.ID, "renamed.mp3")
	if err != nil {
		t.Fatalf("renamed track missing: %v", err)
	}
	if moved.ID != original.ID {
		t.Fatalf("rename should keep track id %q, got %q", original.ID, moved.ID)
	}
	if _, err := tracks.GetByRelPath(lib.ID, "one.mp3"); err == nil {
		t.Fatal("old rel path should be gone after adoption")
	}

	// Removal marks the row missing.
	if err := os.Remove(filepath.Join(root, "renamed.mp3")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := scanner.ApplyChanges(lib.ID, []metaloader.FileChange{
		{RelPath: "renamed.mp3", Removed: true},
	}); err != nil {
		t.Fatalf("ApplyChanges remove: %v", err)
	}
	missing, err := tracks.GetByRelPath(lib.ID, "renamed.mp3")
	if err != nil {
		t.Fatalf("GetByRelPath after remove: %v", err)
	}
	if missing.Status != store.TrackStatusMissing {
		t.Fatalf("expected missing status, got %q", missing.Status)
	}

	// A removed directory marks every track under it missing.
	sub := filepath.Join(root, "album")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, name := range []string{"a.mp3", "b.mp3"} {
		if err := os.WriteFile(filepath.Join(sub, name), []byte("dir-"+name), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := scanner.ApplyChanges(lib.ID, []metaloader.FileChange{
		{RelPath: "album/a.mp3"},
		{RelPath: "album/b.mp3"},
	}); err != nil {
		t.Fatalf("ApplyChanges add dir files: %v", err)
	}
	if err := scanner.ApplyChanges(lib.ID, []metaloader.FileChange{
		{RelPath: "album", Dir: true, Removed: true},
	}); err != nil {
		t.Fatalf("ApplyChanges dir remove: %v", err)
	}
	for _, rel := range []string{"album/a.mp3", "album/b.mp3"} {
		track, err := tracks.GetByRelPath(lib.ID, rel)
		if err != nil || track.Status != store.TrackStatusMissing {
			t.Fatalf("expected %s missing, got err=%v status=%q", rel, err, track.Status)
		}
	}
}

func TestLibraryWatcherDetectsChanges(t *testing.T) {
	libraries, tracks, scanner, lib, root := newWatchFixture(t)

	changed := make(chan string, 8)
	watcher, err := metaloader.NewLibraryWatcher(scanner, func(libraryID string) {
		changed <- libraryID
	})
	if err != nil {
		t.Fatalf("NewLibraryWatcher: %v", err)
	}
	defer watcher.Close()

	if err := watcher.WatchLibrary(lib.ID, root); err != nil {
		t.Fatalf("WatchLibrary: %v", err)
	}
	if !watcher.Watching(lib.ID) {
		t.Fatal("expected library to be watched")
	}

	if err := os.WriteFile(filepath.Join(root, "fresh.mp3"), []byte("fresh-track"), 0o644); err != nil {
		t.Fatalf("write fresh: %v", err)
	}
	track := waitForTrackStatus(t, tracks, lib.ID, "fresh.mp3", true)
	if track.ID == "" {
		t.Fatal("watched file was not added to the catalog")
	}
	select {
	case <-changed:
	case <-time.After(5 * time.Second):
		t.Fatal("expected onChange callback")
	}

	// A new file inside a newly created directory is picked up too.
	sub := filepath.Join(root, "newdir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "nested.mp3"), []byte("nested-track"), 0o644); err != nil {
		t.Fatalf("write nested: %v", err)
	}
	waitForTrackStatus(t, tracks, lib.ID, "newdir/nested.mp3", true)

	// Deleting a file marks it missing without a rescan.
	if err := os.Remove(filepath.Join(root, "fresh.mp3")); err != nil {
		t.Fatalf("remove fresh: %v", err)
	}
	waitForTrackStatus(t, tracks, lib.ID, "fresh.mp3", false)

	watcher.UnwatchLibrary(lib.ID)
	if watcher.Watching(lib.ID) {
		t.Fatal("expected watch removed")
	}

	// Changes after unwatching are ignored.
	if err := os.WriteFile(filepath.Join(root, "late.mp3"), []byte("late"), 0o644); err != nil {
		t.Fatalf("write late: %v", err)
	}
	time.Sleep(3 * metaloader.WatchDebounce)
	if _, err := tracks.GetByRelPath(lib.ID, "late.mp3"); err == nil {
		t.Fatal("unwatched file should not be tracked")
	}

	if _, err := libraries.Get(lib.ID); err != nil {
		t.Fatalf("library row should survive watching: %v", err)
	}
}
