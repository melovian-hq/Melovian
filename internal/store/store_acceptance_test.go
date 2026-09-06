// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import "testing"

func TestPlaylistCreateAddReorderAcceptance(t *testing.T) {
	db := OpenTestDB(t)
	listen := NewListenStore(db)
	userID := "instance:acceptance"

	pl, err := listen.CreatePlaylist(userID, "Acceptance Mix")
	if err != nil {
		t.Fatalf("CreatePlaylist: %v", err)
	}

	tracks := []PlaylistTrack{
		{TrackID: "a1", TrackTitle: "First", ArtistName: "A"},
		{TrackID: "a2", TrackTitle: "Second", ArtistName: "A"},
		{TrackID: "a3", TrackTitle: "Third", ArtistName: "A"},
	}
	for _, track := range tracks {
		if err := listen.AddPlaylistTrack(userID, pl.ID, track); err != nil {
			t.Fatalf("AddPlaylistTrack %s: %v", track.TrackID, err)
		}
	}

	got, err := listen.GetPlaylist(userID, pl.ID)
	if err != nil {
		t.Fatalf("GetPlaylist: %v", err)
	}
	if len(got.Tracks) != 3 {
		t.Fatalf("expected 3 tracks, got %d", len(got.Tracks))
	}
	if got.Tracks[0].TrackID != "a1" || got.Tracks[2].TrackID != "a3" {
		t.Fatalf("unexpected initial order: %+v", got.Tracks)
	}

	reordered := []PlaylistTrack{tracks[2], tracks[0], tracks[1]}
	if err := listen.SetPlaylistTracks(userID, pl.ID, reordered); err != nil {
		t.Fatalf("SetPlaylistTracks: %v", err)
	}

	got, err = listen.GetPlaylist(userID, pl.ID)
	if err != nil {
		t.Fatalf("GetPlaylist after reorder: %v", err)
	}
	if len(got.Tracks) != 3 {
		t.Fatalf("expected 3 tracks after reorder, got %d", len(got.Tracks))
	}
	wantOrder := []string{"a3", "a1", "a2"}
	for i, id := range wantOrder {
		if got.Tracks[i].TrackID != id {
			t.Fatalf("order[%d]=%q want %q (tracks=%+v)", i, got.Tracks[i].TrackID, id, got.Tracks)
		}
	}

	listed, err := listen.ListPlaylists(userID)
	if err != nil {
		t.Fatalf("ListPlaylists: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != pl.ID {
		t.Fatalf("unexpected playlist list: %+v", listed)
	}
}
