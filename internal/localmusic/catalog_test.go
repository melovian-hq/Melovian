// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package localmusic

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"melovian/internal/store"
)

func TestBuildCatalogGroupsTracks(t *testing.T) {
	tracks := []store.CatalogTrack{
		{ID: "trk_1", Artist: "Artist A", Album: "Album One", Title: "Song 1", TrackNum: 1, DurationMs: 180000, Format: "mp3"},
		{ID: "trk_2", Artist: "Artist A", Album: "Album One", Title: "Song 2", TrackNum: 2, DurationMs: 200000, Format: "mp3"},
		{ID: "trk_3", Artist: "Artist B", Album: "Album Two", Title: "Other", TrackNum: 1, DurationMs: 150000, Format: "flac"},
	}

	catalog := BuildCatalog(tracks)
	stats := catalog.Stats()
	if stats.SongCount != 3 || stats.AlbumCount != 2 || stats.ArtistCount != 2 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	albumID := AlbumID("Artist A", "Album One")
	album, songs, ok := catalog.Album(albumID)
	if !ok {
		t.Fatal("expected album")
	}
	if album.SongCount != 2 || len(songs) != 2 {
		t.Fatalf("unexpected album: %+v songs=%d", album, len(songs))
	}

	artists, albums, foundSongs := catalog.Search("song 2", 10)
	if len(foundSongs) != 1 || foundSongs[0].ID != "trk_2" {
		t.Fatalf("unexpected search songs: %+v", foundSongs)
	}
	if len(artists) != 0 && len(albums) != 0 {
		t.Fatalf("expected song-only search match, artists=%d albums=%d", len(artists), len(albums))
	}
}

func TestArtistAndAlbumIDsStable(t *testing.T) {
	a := ArtistID("Artist A")
	b := ArtistID("artist a")
	if a != b {
		t.Fatalf("expected stable artist id, got %q and %q", a, b)
	}
	albumA := AlbumID("Artist A", "Album")
	albumB := AlbumID("artist a", "album")
	if albumA != albumB {
		t.Fatalf("expected stable album id, got %q and %q", albumA, albumB)
	}
}

func TestRandomSongsShufflesSelection(t *testing.T) {
	tracks := make([]store.CatalogTrack, 20)
	for i := range tracks {
		tracks[i] = store.CatalogTrack{
			ID:         fmt.Sprintf("trk_%02d", i),
			Artist:     "Artist",
			Album:      "Album",
			Title:      "Song",
			TrackNum:   i + 1,
			DurationMs: 180000,
			Format:     "mp3",
		}
	}
	catalog := BuildCatalog(tracks)

	first := catalog.RandomSongs(10)
	second := catalog.RandomSongs(10)
	if len(first) != 10 || len(second) != 10 {
		t.Fatalf("expected 10 songs, got %d and %d", len(first), len(second))
	}

	sameOrder := true
	for i := range first {
		if first[i].ID != catalog.songsByIDAsc[i].ID {
			sameOrder = false
			break
		}
	}
	if sameOrder {
		t.Fatal("expected random songs to differ from sorted catalog order")
	}
}

func TestResolveTrackPath(t *testing.T) {
	root := t.TempDir()
	inside := root + "/song.mp3"
	if _, err := ResolveTrackPath(root, inside); err != nil {
		t.Fatalf("expected inside path to resolve, got %v", err)
	}
	if _, err := ResolveTrackPath(root, "/etc/passwd"); err == nil {
		t.Fatal("expected escape attempt to fail")
	}
}

func TestResolveTrackPathRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(outside, []byte("nope"), 0o644); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	link := filepath.Join(root, "song.mp3")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, err := ResolveTrackPath(root, link); err == nil {
		t.Fatal("expected symlink escape to fail")
	}
}
