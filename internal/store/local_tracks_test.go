// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestConcurrentTrackUpsert(t *testing.T) {
	db := OpenTestDB(t)
	libraries := NewLocalLibraryStore(db)
	tracks := NewLocalTrackStore(db)

	lib, err := libraries.Create(CreateLocalLibraryInput{
		Name: "Test",
		Path: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	done := make(chan struct{})
	var wg sync.WaitGroup

	for i := range 8 {
		wg.Go(func() {
			for j := range 25 {
				rel := fmt.Sprintf("album/track-%d-%d.mp3", i, j)
				input := UpsertLocalTrackInput{
					LibraryID:   lib.ID,
					RelPath:     rel,
					AbsPath:     "/music/" + rel,
					FileSig:     fmt.Sprintf("sig-%d-%d", i, j),
					ContentHash: fmt.Sprintf("hash-%d-%d", i, j),
					Size:        100,
					Mtime:       time.Now().Unix(),
					Title:       "Song",
					Artist:      "Artist",
					Album:       "Album",
					Format:      "mp3",
				}
				if _, err := tracks.Upsert(input); err != nil {
					t.Errorf("Upsert: %v", err)
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
	case <-time.After(10 * time.Second):
		t.Fatal("concurrent track upsert did not complete within timeout")
	}

	present, err := tracks.ListPresent(lib.ID)
	if err != nil {
		t.Fatalf("ListPresent: %v", err)
	}
	if len(present) != 200 {
		t.Fatalf("expected 200 tracks, got %d", len(present))
	}
}

func TestCountDistinctArtistsAlbums(t *testing.T) {
	db := OpenTestDB(t)
	libraries := NewLocalLibraryStore(db)
	tracks := NewLocalTrackStore(db)

	lib, err := libraries.Create(CreateLocalLibraryInput{Name: "Test", Path: t.TempDir()})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	for _, input := range []UpsertLocalTrackInput{
		{LibraryID: lib.ID, RelPath: "a1.mp3", AbsPath: "/a1.mp3", FileSig: "s1", ContentHash: "h1", Size: 1, Mtime: 1, Artist: "Artist A", Album: "Album 1", Format: "mp3"},
		{LibraryID: lib.ID, RelPath: "a2.mp3", AbsPath: "/a2.mp3", FileSig: "s2", ContentHash: "h2", Size: 1, Mtime: 1, Artist: "Artist A", Album: "Album 2", Format: "mp3"},
		{LibraryID: lib.ID, RelPath: "b1.mp3", AbsPath: "/b1.mp3", FileSig: "s3", ContentHash: "h3", Size: 1, Mtime: 1, Artist: "Artist B", Album: "Album 1", Format: "mp3"},
	} {
		if _, err := tracks.Upsert(input); err != nil {
			t.Fatalf("Upsert: %v", err)
		}
	}

	artists, albums, err := tracks.CountDistinctArtistsAlbums(lib.ID)
	if err != nil {
		t.Fatalf("CountDistinctArtistsAlbums: %v", err)
	}
	if artists != 2 || albums != 2 {
		t.Fatalf("expected 2 artists and 2 albums, got %d artists %d albums", artists, albums)
	}
}
