// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import "testing"

func TestSearchMetadataFilters(t *testing.T) {
	db := OpenTestDB(t)
	libraries := NewLocalLibraryStore(db)
	tracks := NewLocalTrackStore(db)

	lib, err := libraries.Create(CreateLocalLibraryInput{Name: "Test", Path: t.TempDir()})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	inputs := []UpsertLocalTrackInput{
		{LibraryID: lib.ID, RelPath: "good.mp3", AbsPath: "/good.mp3", FileSig: "s1", ContentHash: "h1", Size: 1, Mtime: 1, Title: "Good Song", Artist: "Artist", Album: "Album", Format: "mp3"},
		{LibraryID: lib.ID, RelPath: "bad.mp3", AbsPath: "/bad.mp3", FileSig: "s2", ContentHash: "h2", Size: 1, Mtime: 1, Title: "Mystery", Artist: "Unknown Artist", Album: "Unknown Album", Format: "mp3"},
		{LibraryID: lib.ID, RelPath: "untitled.mp3", AbsPath: "/untitled.mp3", FileSig: "s3", ContentHash: "h3", Size: 1, Mtime: 1, Title: "", Artist: "Artist", Album: "Album", Format: "mp3"},
	}
	for _, input := range inputs {
		if _, err := tracks.Upsert(input); err != nil {
			t.Fatalf("Upsert: %v", err)
		}
	}

	unknownArtist, unknownAlbum, missingTitle, any, err := tracks.CountMetadataIssues(lib.ID)
	if err != nil {
		t.Fatalf("CountMetadataIssues: %v", err)
	}
	if unknownArtist < 1 || unknownAlbum < 1 || missingTitle < 1 || any < 2 {
		t.Fatalf("unexpected counts: artist=%d album=%d title=%d any=%d", unknownArtist, unknownAlbum, missingTitle, any)
	}

	found, total, err := tracks.SearchMetadata(MetadataSearchQuery{
		LibraryID: lib.ID,
		Query:     "mystery",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("SearchMetadata: %v", err)
	}
	if total != 1 || len(found) != 1 || found[0].Title != "Mystery" {
		t.Fatalf("unexpected search result: total=%d found=%+v", total, found)
	}

	albumPeers, err := tracks.ListByAlbumArtist(lib.ID, "Album", "Artist")
	if err != nil {
		t.Fatalf("ListByAlbumArtist: %v", err)
	}
	if len(albumPeers) != 2 {
		t.Fatalf("expected 2 album peers, got %d", len(albumPeers))
	}

	filtered, total, err := tracks.SearchMetadata(MetadataSearchQuery{
		LibraryID: lib.ID,
		Issue:     "unknown-artist",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("SearchMetadata issue: %v", err)
	}
	if total != 1 || len(filtered) != 1 {
		t.Fatalf("unexpected issue filter: total=%d len=%d", total, len(filtered))
	}
}
