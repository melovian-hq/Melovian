// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"testing"
)

func TestListForCatalog(t *testing.T) {
	db := OpenTestDB(t)
	libraries := NewLocalLibraryStore(db)
	tracks := NewLocalTrackStore(db)

	lib, err := libraries.Create(CreateLocalLibraryInput{Name: "Test", Path: t.TempDir()})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := tracks.Upsert(UpsertLocalTrackInput{
		LibraryID:   lib.ID,
		RelPath:     "song.mp3",
		AbsPath:     "/music/song.mp3",
		FileSig:     "sig",
		ContentHash: "hash",
		Size:        1,
		Mtime:       1,
		Title:       "My Song",
		Artist:      "Artist",
		Album:       "Album",
		Format:      "mp3",
		MediaKind:   MediaKindAudio,
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if _, err := tracks.Upsert(UpsertLocalTrackInput{
		LibraryID:   lib.ID,
		RelPath:     "clip.mp4",
		AbsPath:     "/music/clip.mp4",
		FileSig:     "sig2",
		ContentHash: "hash2",
		Size:        2,
		Mtime:       2,
		Title:       "My Video",
		Artist:      "Artist",
		Album:       "Album",
		Format:      "mp4",
		MediaKind:   MediaKindVideo,
	}); err != nil {
		t.Fatalf("Upsert video: %v", err)
	}

	items, err := tracks.ListForCatalog(lib.ID)
	if err != nil {
		t.Fatalf("ListForCatalog: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 audio track, got %d", len(items))
	}
	if items[0].Title != "My Song" || items[0].RelPath != "song.mp3" {
		t.Fatalf("unexpected catalog track: %+v", items[0])
	}

	videos, err := tracks.ListVideos(lib.ID)
	if err != nil {
		t.Fatalf("ListVideos: %v", err)
	}
	if len(videos) != 1 || videos[0].Title != "My Video" {
		t.Fatalf("unexpected videos: %+v", videos)
	}
}
