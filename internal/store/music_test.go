// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"testing"
)

func TestListenProgressScopedByUser(t *testing.T) {
	db := OpenTestDB(t)
	listen := NewListenStore(db)

	userA := "instance:aaa"
	userB := "instance:bbb"

	if err := listen.Upsert(userA, ListenUpsertInput{
		TrackID:    "track-1",
		PositionMs: 1000,
		TrackTitle: "Song A",
	}); err != nil {
		t.Fatalf("Upsert A: %v", err)
	}
	if err := listen.Upsert(userB, ListenUpsertInput{
		TrackID:    "track-1",
		PositionMs: 2000,
		TrackTitle: "Song B",
	}); err != nil {
		t.Fatalf("Upsert B: %v", err)
	}

	itemA, err := listen.Get(userA, "track-1")
	if err != nil {
		t.Fatalf("Get A: %v", err)
	}
	if itemA.PositionMs != 1000 {
		t.Fatalf("expected position 1000, got %d", itemA.PositionMs)
	}

	itemB, err := listen.Get(userB, "track-1")
	if err != nil {
		t.Fatalf("Get B: %v", err)
	}
	if itemB.PositionMs != 2000 {
		t.Fatalf("expected position 2000, got %d", itemB.PositionMs)
	}
}

func TestListenHistoryAndStats(t *testing.T) {
	db := OpenTestDB(t)
	listen := NewListenStore(db)
	userID := "instance:test"

	if err := listen.Upsert(userID, ListenUpsertInput{
		TrackID:       "t1",
		IncrementPlay: true,
		TrackTitle:    "Track One",
		ArtistName:    "Artist",
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	history, err := listen.History(userID, 10)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history item, got %d", len(history))
	}

	stats, err := listen.Stats(userID, 5)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.TotalPlays < 1 {
		t.Fatalf("expected at least 1 play, got %d", stats.TotalPlays)
	}
}

func TestResumeTracksFiltersIncompletePlayback(t *testing.T) {
	db := OpenTestDB(t)
	listen := NewListenStore(db)
	userID := "instance:test"

	cases := []struct {
		trackID    string
		positionMs int
		played     bool
		wantResume bool
	}{
		{"resume-me", 6000, false, true},
		{"too-short", 1000, false, false},
		{"already-played", 9000, true, false},
	}

	for _, tc := range cases {
		if err := listen.Upsert(userID, ListenUpsertInput{
			TrackID:    tc.trackID,
			PositionMs: int64(tc.positionMs),
			Played:     tc.played,
			TrackTitle: tc.trackID,
		}); err != nil {
			t.Fatalf("Upsert %s: %v", tc.trackID, err)
		}
	}

	resume, err := listen.ResumeTracks(userID, 10)
	if err != nil {
		t.Fatalf("ResumeTracks: %v", err)
	}
	if len(resume) != 1 || resume[0].TrackID != "resume-me" {
		t.Fatalf("expected only resume-me, got %+v", resume)
	}
}

func TestPlaylistLifecycle(t *testing.T) {
	db := OpenTestDB(t)
	listen := NewListenStore(db)
	userID := "instance:test"

	pl, err := listen.CreatePlaylist(userID, "Road Mix")
	if err != nil {
		t.Fatalf("CreatePlaylist: %v", err)
	}
	if pl.Name != "Road Mix" {
		t.Fatalf("unexpected playlist name %q", pl.Name)
	}

	track := PlaylistTrack{TrackID: "t1", TrackTitle: "Song One", ArtistName: "Artist"}
	if err := listen.AddPlaylistTrack(userID, pl.ID, track); err != nil {
		t.Fatalf("AddPlaylistTrack: %v", err)
	}

	got, err := listen.GetPlaylist(userID, pl.ID)
	if err != nil {
		t.Fatalf("GetPlaylist: %v", err)
	}
	if len(got.Tracks) != 1 || got.Tracks[0].TrackID != "t1" {
		t.Fatalf("unexpected tracks %+v", got.Tracks)
	}

	if err := listen.RenamePlaylist(userID, pl.ID, "Night Drive"); err != nil {
		t.Fatalf("RenamePlaylist: %v", err)
	}
	got, err = listen.GetPlaylist(userID, pl.ID)
	if err != nil {
		t.Fatalf("GetPlaylist after rename: %v", err)
	}
	if got.Name != "Night Drive" {
		t.Fatalf("expected renamed playlist, got %q", got.Name)
	}

	if err := listen.RemovePlaylistTrack(userID, pl.ID, "t1"); err != nil {
		t.Fatalf("RemovePlaylistTrack: %v", err)
	}
	got, err = listen.GetPlaylist(userID, pl.ID)
	if err != nil {
		t.Fatalf("GetPlaylist after remove: %v", err)
	}
	if len(got.Tracks) != 0 {
		t.Fatalf("expected empty playlist, got %+v", got.Tracks)
	}

	if err := listen.DeletePlaylist(userID, pl.ID); err != nil {
		t.Fatalf("DeletePlaylist: %v", err)
	}
	if _, err := listen.GetPlaylist(userID, pl.ID); err == nil {
		t.Fatal("expected deleted playlist to be missing")
	}
}

func TestFavoritesAddRemoveList(t *testing.T) {
	db := OpenTestDB(t)
	listen := NewListenStore(db)
	userID := "instance:test"

	track := FavoriteTrack{
		TrackID:    "fav-1",
		TrackTitle: "Favorite Song",
		ArtistName: "Artist",
	}
	if err := listen.AddFavorite(userID, track); err != nil {
		t.Fatalf("AddFavorite: %v", err)
	}

	items, err := listen.ListFavorites(userID, 10)
	if err != nil {
		t.Fatalf("ListFavorites: %v", err)
	}
	if len(items) != 1 || items[0].TrackID != "fav-1" {
		t.Fatalf("unexpected favorites %+v", items)
	}

	if err := listen.RemoveFavorite(userID, "fav-1"); err != nil {
		t.Fatalf("RemoveFavorite: %v", err)
	}
	items, err = listen.ListFavorites(userID, 10)
	if err != nil {
		t.Fatalf("ListFavorites after remove: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty favorites, got %+v", items)
	}
}
