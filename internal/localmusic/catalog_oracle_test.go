// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package localmusic

import (
	"strings"
	"testing"
)

func TestNormalizeNameOracle(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "", want: ""},
		{in: "  ", want: ""},
		{in: "Artist", want: "artist"},
		{in: "  Album Name  ", want: "album name"},
		{in: "MiXeD CaSe", want: "mixed case"},
	}
	for _, tc := range cases {
		got := NormalizeName(tc.in)
		if got != tc.want {
			t.Fatalf("NormalizeName(%q) = %q, want %q", tc.in, got, tc.want)
		}
		if got != strings.ToLower(strings.TrimSpace(tc.in)) {
			t.Fatalf("NormalizeName(%q) not trim+lower: %q", tc.in, got)
		}
	}
}

func TestArtistAlbumNameOracle(t *testing.T) {
	if got := ArtistName(""); got != "Unknown Artist" {
		t.Fatalf("ArtistName empty = %q", got)
	}
	if got := ArtistName("  Band  "); got != "Band" {
		t.Fatalf("ArtistName trimmed = %q", got)
	}
	if got := AlbumName(""); got != "Unknown Album" {
		t.Fatalf("AlbumName empty = %q", got)
	}
	if got := AlbumName("  LP  "); got != "LP" {
		t.Fatalf("AlbumName trimmed = %q", got)
	}
}

func TestArtistIDOracleStable(t *testing.T) {
	a := ArtistID("Artist A")
	b := ArtistID("  artist a  ")
	if a != b {
		t.Fatalf("expected stable artist ids, got %q and %q", a, b)
	}
	if !strings.HasPrefix(a, ArtistIDPrefix) {
		t.Fatalf("expected prefix %q, got %q", ArtistIDPrefix, a)
	}
}
