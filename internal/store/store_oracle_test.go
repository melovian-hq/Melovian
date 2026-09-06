// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import "testing"

func TestInstanceRoundtripOracle(t *testing.T) {
	db := OpenTestDB(t)
	instances := NewInstanceStore(db)

	created, err := instances.Create(CreateInstanceInput{
		Name:      "Oracle Home",
		ServerURL: "http://music.example.com",
		Username:  "listener",
		Password:  "secret",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty id")
	}
	if created.Name != "Oracle Home" {
		t.Fatalf("name = %q, want Oracle Home", created.Name)
	}
	if created.ServerURL != "http://music.example.com" {
		t.Fatalf("server url = %q", created.ServerURL)
	}

	got, err := instances.Get(created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != created.ID || got.Name != created.Name || got.ServerURL != created.ServerURL {
		t.Fatalf("roundtrip mismatch: %+v vs %+v", got, created)
	}
}

func TestPlaylistInsertGetOracle(t *testing.T) {
	db := OpenTestDB(t)
	listen := NewListenStore(db)
	userID := "instance:oracle"

	pl, err := listen.CreatePlaylist(userID, "Oracle Mix")
	if err != nil {
		t.Fatalf("CreatePlaylist: %v", err)
	}
	if pl.ID == "" || pl.Name != "Oracle Mix" {
		t.Fatalf("unexpected playlist: %+v", pl)
	}

	track := PlaylistTrack{
		TrackID:    "oracle-t1",
		TrackTitle: "Song One",
		ArtistName: "Artist",
	}
	if err := listen.AddPlaylistTrack(userID, pl.ID, track); err != nil {
		t.Fatalf("AddPlaylistTrack: %v", err)
	}

	got, err := listen.GetPlaylist(userID, pl.ID)
	if err != nil {
		t.Fatalf("GetPlaylist: %v", err)
	}
	if got.Name != "Oracle Mix" {
		t.Fatalf("name = %q", got.Name)
	}
	if len(got.Tracks) != 1 || got.Tracks[0].TrackID != "oracle-t1" {
		t.Fatalf("unexpected tracks: %+v", got.Tracks)
	}
}

func TestSmartPlaylistRulesOracle(t *testing.T) {
	db := OpenTestDB(t)
	listen := NewListenStore(db)
	userID := "instance:smart"

	rules := `{"name":"Jazz","root":{"logic":"all","rules":[{"field":"genre","operator":"contains","value":"Jazz"}]}}`
	pl, err := listen.CreatePlaylistOpts(userID, CreatePlaylistOpts{
		Name:      "Jazz Smart",
		Kind:      "smart",
		RulesJSON: rules,
	})
	if err != nil {
		t.Fatalf("CreatePlaylistOpts: %v", err)
	}
	if pl.Kind != "smart" || pl.RulesJSON != rules {
		t.Fatalf("unexpected create: %+v", pl)
	}

	got, err := listen.GetPlaylist(userID, pl.ID)
	if err != nil {
		t.Fatalf("GetPlaylist: %v", err)
	}
	if got.Kind != "smart" || got.RulesJSON != rules {
		t.Fatalf("rules roundtrip failed: %+v", got)
	}

	listed, err := listen.ListPlaylists(userID)
	if err != nil {
		t.Fatalf("ListPlaylists: %v", err)
	}
	if len(listed) != 1 || listed[0].Kind != "smart" || listed[0].RulesJSON != rules {
		t.Fatalf("list missing smart rules: %+v", listed)
	}
}

func TestListenQueryLimitsOracle(t *testing.T) {
	db := OpenTestDB(t)
	listen := NewListenStore(db)
	if _, err := listen.History("user", 1_000_000); err != nil {
		t.Fatalf("History: %v", err)
	}
	ids := make([]string, 250)
	for i := range ids {
		ids[i] = "track-" + string(rune('a'+i%26))
	}
	batch, err := listen.Batch("user", ids)
	if err != nil {
		t.Fatalf("Batch: %v", err)
	}
	if batch == nil {
		t.Fatal("expected map from Batch")
	}
}
