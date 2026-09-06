// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metadata

import "testing"

func TestParseFilenameSuggestionArtistTitle(t *testing.T) {
	got := ParseFilenameSuggestion("Slowdive - When the Sun Hits.mp3")
	if got.Artist != "Slowdive" || got.Title != "When the Sun Hits" {
		t.Fatalf("unexpected suggestion: %+v", got)
	}
}

func TestParseFilenameSuggestionTrackPrefix(t *testing.T) {
	got := ParseFilenameSuggestion("01 Cherry-coloured Funk.flac")
	if got.TrackNum != 1 || got.Title != "Cherry-coloured Funk" {
		t.Fatalf("unexpected suggestion: %+v", got)
	}
}

func TestParseFilenameSuggestionFolderLayout(t *testing.T) {
	got := ParseFilenameSuggestion("Slowdive/Souvlaki/03 Alison.mp3")
	if got.Artist != "Slowdive" || got.Album != "Souvlaki" || got.TrackNum != 3 || got.Title != "Alison" {
		t.Fatalf("unexpected suggestion: %+v", got)
	}
}

func TestParseFilenameSuggestionDiscTrack(t *testing.T) {
	got := ParseFilenameSuggestion("2-05 Bonus Track.mp3")
	if got.DiscNum != 2 || got.TrackNum != 5 || got.Title != "Bonus Track" {
		t.Fatalf("unexpected suggestion: %+v", got)
	}
}
