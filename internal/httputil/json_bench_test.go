// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"net/http/httptest"
	"testing"
	"time"

	"melovian/internal/store"
)

func BenchmarkWriteJSONListenProgress(b *testing.B) {
	item := store.ListenProgressJSON{
		TrackID:      "trk_1",
		TrackTitle:   "Example Song",
		ArtistName:   "Example Artist",
		AlbumID:      "alb_1",
		AlbumTitle:   "Example Album",
		PositionMs:   120_000,
		DurationMs:   240_000,
		Played:       false,
		PlayCount:    3,
		LastPlayedAt: time.Now().UTC().Format(time.RFC3339),
		CoverArtID:   "alb_1",
	}

	b.ReportAllocs()
	for b.Loop() {
		rec := httptest.NewRecorder()
		WriteJSON(rec, 200, item)
	}
}
