// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package localmusic

import (
	"fmt"
	"testing"

	"melovian/internal/store"
)

func benchmarkTracks(n int) []store.CatalogTrack {
	tracks := make([]store.CatalogTrack, n)
	for i := range n {
		artist := fmt.Sprintf("Artist %d", i%50)
		album := fmt.Sprintf("Album %d", i%200)
		tracks[i] = store.CatalogTrack{
			ID:         fmt.Sprintf("trk_%d", i),
			Artist:     artist,
			Album:      album,
			Title:      fmt.Sprintf("Song %d", i),
			TrackNum:   i%20 + 1,
			DurationMs: 180000,
			Format:     "mp3",
		}
	}
	return tracks
}

func BenchmarkBuildCatalog(b *testing.B) {
	tracks := benchmarkTracks(5000)
	b.ReportAllocs()
	for b.Loop() {
		BuildCatalog(tracks)
	}
}

func BenchmarkCatalogSearch(b *testing.B) {
	catalog := BuildCatalog(benchmarkTracks(5000))
	b.ReportAllocs()
	for b.Loop() {
		catalog.Search("song 42", 20)
	}
}

func BenchmarkCatalogArtistLookup(b *testing.B) {
	catalog := BuildCatalog(benchmarkTracks(5000))
	artistID := catalog.Artists[0].ID
	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = catalog.Artist(artistID)
	}
}
