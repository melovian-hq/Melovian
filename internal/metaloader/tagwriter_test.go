// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metaloader

import (
	"os"
	"path/filepath"
	"testing"
)

func writeMinimalWAV(t *testing.T, path string) {
	t.Helper()
	hdr := []byte("RIFF$\x00\x00\x00WAVEfmt \x10\x00\x00\x00\x01\x00\x01\x00\x44\xac\x00\x00\x88X\x01\x00\x02\x00\x10\x00data\x00\x00\x00\x00")
	if err := os.WriteFile(path, hdr, 0o644); err != nil {
		t.Fatalf("write wav: %v", err)
	}
}

func TestReadWriteTrackMetadataWAV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "track.wav")
	writeMinimalWAV(t, path)

	meta := TrackMetadata{
		Title:       "Alison",
		Artist:      "Slowdive",
		Album:       "Souvlaki",
		AlbumArtist: "Slowdive",
		TrackNum:    3,
		Year:        1993,
		Genre:       "Shoegaze",
	}
	if err := WriteTrackMetadata(path, meta); err != nil {
		t.Fatalf("WriteTrackMetadata: %v", err)
	}

	read, err := ReadTrackMetadata(path)
	if err != nil {
		t.Fatalf("ReadTrackMetadata: %v", err)
	}
	if read.Title != meta.Title || read.Artist != meta.Artist || read.Album != meta.Album {
		t.Fatalf("unexpected read metadata: %+v", read)
	}
	if read.Year != meta.Year || read.Genre != meta.Genre {
		t.Fatalf("unexpected year/genre: %+v", read)
	}
}
