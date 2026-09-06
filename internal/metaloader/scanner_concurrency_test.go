// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metaloader

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"melovian/internal/store"
)

func TestConcurrentScanSameLibrary(t *testing.T) {
	db := store.OpenTestDB(t)
	libraries := store.NewLocalLibraryStore(db)
	tracks := store.NewLocalTrackStore(db)
	scanner := NewScanner(libraries, tracks)

	root := t.TempDir()
	for i := range 10 {
		name := fmt.Sprintf("track-%02d.mp3", i)
		if err := os.WriteFile(filepath.Join(root, name), []byte("data-"+name), 0o644); err != nil {
			t.Fatalf("write track: %v", err)
		}
	}

	lib, err := libraries.Create(store.CreateLocalLibraryInput{Name: "Test", Path: root})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	done := make(chan struct{})
	var wg sync.WaitGroup

	for range 6 {
		wg.Go(func() {
			for range 5 {
				result, err := scanner.Scan(lib.ID)
				if err != nil {
					t.Errorf("Scan: %v", err)
					return
				}
				if result.Total != 10 {
					t.Errorf("expected 10 tracks, got %d", result.Total)
				}
			}
		})
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("concurrent scans did not complete within timeout")
	}

	loaded, err := libraries.Get(lib.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if loaded.ScanStatus != "idle" {
		t.Fatalf("expected idle scan status, got %q (error: %q)", loaded.ScanStatus, loaded.ScanError)
	}
	if loaded.TrackCount != 10 {
		t.Fatalf("expected 10 tracks, got %d", loaded.TrackCount)
	}
}
