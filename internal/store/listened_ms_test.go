// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"math"
	"testing"
)

func TestClampListenDeltaUnit(t *testing.T) {
	cases := []struct {
		name       string
		delta      int64
		durationMs int
		want       int64
	}{
		{name: "zero", delta: 0, durationMs: 180_000, want: 0},
		{name: "negative", delta: -5000, durationMs: 180_000, want: 0},
		{name: "normal", delta: 5000, durationMs: 180_000, want: 5000},
		{name: "chunk cap", delta: 60_000, durationMs: 180_000, want: 15_000},
		{name: "duration cap", delta: 20_000, durationMs: 8_000, want: 8_000},
		{name: "no duration", delta: 20_000, durationMs: 0, want: 15_000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := clampListenDelta(tc.delta, tc.durationMs)
			if got != tc.want {
				t.Fatalf("clampListenDelta(%d, %d) = %d, want %d", tc.delta, tc.durationMs, got, tc.want)
			}
		})
	}
}

func TestListenedMsAccumulatesOnDisk(t *testing.T) {
	db := OpenTestDB(t)
	listen := NewListenStore(db)
	userID := "instance:listen-ms"

	if err := listen.Upsert(userID, ListenUpsertInput{
		TrackID:    "t1",
		PositionMs: 5000,
		DeltaMs:    5000,
		DurationMs: 180_000,
		TrackTitle: "Song",
		ArtistName: "Artist",
	}); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if err := listen.Upsert(userID, ListenUpsertInput{
		TrackID:    "t1",
		PositionMs: 12000,
		DeltaMs:    7000,
		DurationMs: 180_000,
		TrackTitle: "Song",
		ArtistName: "Artist",
	}); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	item, err := listen.Get(userID, "t1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if item.ListenedMs != 12_000 {
		t.Fatalf("expected listened_ms 12000, got %d", item.ListenedMs)
	}
	if item.JSON().ListenedMs != 12_000 {
		t.Fatalf("JSON listenedMs mismatch: %d", item.JSON().ListenedMs)
	}
}

func TestListenedMsAdversarialInflationRejected(t *testing.T) {
	db := OpenTestDB(t)
	listen := NewListenStore(db)
	userID := "instance:listen-adv"

	if err := listen.Upsert(userID, ListenUpsertInput{
		TrackID:    "t1",
		PositionMs: 1,
		DeltaMs:    math.MaxInt64,
		DurationMs: 60_000,
		TrackTitle: "Song",
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	item, err := listen.Get(userID, "t1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if item.ListenedMs > 15_000 {
		t.Fatalf("adversarial delta should clamp to <=15000, got %d", item.ListenedMs)
	}
	if item.ListenedMs != 15_000 {
		t.Fatalf("expected chunk clamp 15000, got %d", item.ListenedMs)
	}
}

func TestStatsPreferListenedMs(t *testing.T) {
	db := OpenTestDB(t)
	listen := NewListenStore(db)
	userID := "instance:stats-listen"

	if err := listen.Upsert(userID, ListenUpsertInput{
		TrackID:       "t1",
		IncrementPlay: true,
		DeltaMs:       12_000,
		DurationMs:    180_000,
		TrackTitle:    "Heard",
		ArtistName:    "Alpha",
		AlbumID:       "al1",
		AlbumTitle:    "Album",
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	stats, err := listen.Stats(userID, 5)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.TotalListeningMs != 12_000 {
		t.Fatalf("expected totalListeningMs 12000 from listened_ms, got %d", stats.TotalListeningMs)
	}
	if len(stats.TopArtists) == 0 || stats.TopArtists[0].Label != "Alpha" {
		t.Fatalf("expected Alpha top artist, got %+v", stats.TopArtists)
	}
}

func TestListenedMsFallbackWhenZero(t *testing.T) {
	db := OpenTestDB(t)
	listen := NewListenStore(db)
	userID := "instance:stats-fallback"

	if err := listen.Upsert(userID, ListenUpsertInput{
		TrackID:       "t1",
		IncrementPlay: true,
		DeltaMs:       0,
		DurationMs:    1000,
		TrackTitle:    "Legacy",
		ArtistName:    "Beta",
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	stats, err := listen.Stats(userID, 5)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.TotalListeningMs != 1000 {
		t.Fatalf("expected duration*playCount fallback 1000, got %d", stats.TotalListeningMs)
	}
}
