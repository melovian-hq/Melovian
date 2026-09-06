// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package democatalog

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"melovian/internal/store"
)

// SeedUserData fills local playlists and listen history for the demo instance scope.
// Safe to call repeatedly. Skips when playlists already exist for the user.
func SeedUserData(listen *store.ListenStore, userID string) error {
	if listen == nil || userID == "" {
		return nil
	}
	existing, err := listen.ListPlaylists(userID)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}

	c := Get()
	localPlaylists := []struct {
		name   string
		titles []string
	}{
		{"Chart Stack", []string{"Choosin' Texas", "Aperture", "drop dead", "SWIM", "Janice STFU", "DtMF"}},
		{"Bass Desk", []string{"Prophecy", "Contorted", "Delilah (Pull Me Out of This)", "Kammy (Like I Do)", "Rumble", "adore u"}},
		{"Weekend Hits", []string{"DtMF", "Be Her", "Opalite", "Abracadabra", "Birds of a Feather", "American Girls"}},
	}

	for _, spec := range localPlaylists {
		pl, err := listen.CreatePlaylist(userID, spec.name)
		if err != nil {
			return fmt.Errorf("create playlist %q: %w", spec.name, err)
		}
		tracks := make([]store.PlaylistTrack, 0, len(spec.titles))
		for _, sid := range pickByTitle(c, spec.titles) {
			s, ok := c.Song(sid)
			if !ok {
				continue
			}
			tracks = append(tracks, store.PlaylistTrack{
				TrackID:    s.ID,
				TrackTitle: s.Title,
				ArtistName: s.Artist,
				AlbumID:    s.AlbumID,
				AlbumTitle: s.Album,
				DurationMs: s.Duration * 1000,
				CoverArtID: s.CoverArt,
			})
		}
		if err := listen.SetPlaylistTracks(userID, pl.ID, tracks); err != nil {
			return fmt.Errorf("fill playlist %q: %w", spec.name, err)
		}
	}

	// Oldest first. Last entries win Jump back in / Recently played ordering.
	// Keep Bruno sparse so 2026 chart albums dominate the home shelves.
	historyTitles := []string{
		"24K Magic",
		"Blinding Lights",
		"Anti-Hero",
		"Birds of a Feather",
		"Die with a Smile",
		"Prophecy",
		"Delilah (Pull Me Out of This)",
		"yes, and?",
		"Abracadabra",
		"The Fate of Ophelia",
		"Opalite",
		"I Just Might",
		"hate that i made you love me",
		"Janice STFU",
		"drop dead",
		"SWIM",
		"Aperture",
		"Be Her",
		"Choosin' Texas",
		"Baile Inolvidable",
		"DtMF",
	}

	base := time.Now().Add(-48 * time.Hour).Unix()
	for i, title := range historyTitles {
		ids := pickByTitle(c, []string{title})
		if len(ids) == 0 {
			continue
		}
		s, ok := c.Song(ids[0])
		if !ok {
			continue
		}
		playedAt := base + int64(i*90)
		plays := 2 + (i % 4)
		for p := range plays {
			if err := listen.Upsert(userID, store.ListenUpsertInput{
				TrackID:       s.ID,
				PositionMs:    0,
				Played:        true,
				IncrementPlay: true,
				DeltaMs:       int64(s.Duration*1000) / 2,
				TrackTitle:    s.Title,
				ArtistName:    s.Artist,
				AlbumID:       s.AlbumID,
				AlbumTitle:    s.Album,
				DurationMs:    s.Duration * 1000,
				CoverArtID:    s.CoverArt,
				LastPlayedAt:  playedAt + int64(p),
			}); err != nil {
				return fmt.Errorf("seed listen %s: %w", s.ID, err)
			}
		}
	}

	// Unfinished Bad Bunny play so resume / Jump back in leads with DtMF.
	if ids := pickByTitle(c, []string{"DtMF"}); len(ids) > 0 {
		if s, ok := c.Song(ids[0]); ok {
			if err := listen.Upsert(userID, store.ListenUpsertInput{
				TrackID:       s.ID,
				PositionMs:    int64(s.Duration*1000) * 45 / 100,
				Played:        false,
				IncrementPlay: false,
				DeltaMs:       int64(s.Duration*1000) * 45 / 100,
				TrackTitle:    s.Title,
				ArtistName:    s.Artist,
				AlbumID:       s.AlbumID,
				AlbumTitle:    s.Album,
				DurationMs:    s.Duration * 1000,
				CoverArtID:    s.CoverArt,
				LastPlayedAt:  time.Now().Unix(),
			}); err != nil {
				return fmt.Errorf("seed resume %s: %w", s.ID, err)
			}
		}
	}

	slog.Info("demo catalog seeded local playlists and listen history", "user", userID)
	return nil
}

// EnsureInstance creates the fake demo Subsonic instance when missing.
func EnsureInstance(instances *store.InstanceStore) (store.SubsonicInstance, error) {
	items, err := instances.List()
	if err != nil {
		return store.SubsonicInstance{}, err
	}
	for _, inst := range items {
		if IsFakeURL(inst.ServerURL) {
			_ = instances.SetActive(inst.ID)
			return inst, nil
		}
	}
	if len(items) > 0 {
		active, err := instances.GetActive()
		if err == nil {
			return active, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return store.SubsonicInstance{}, err
		}
	}
	inst, err := instances.Create(store.CreateInstanceInput{
		Name:      "Home Library",
		ServerURL: ServerURL,
		Username:  Username,
		Password:  Password,
	})
	if err != nil {
		return store.SubsonicInstance{}, err
	}
	if err := instances.SetActive(inst.ID); err != nil {
		return store.SubsonicInstance{}, err
	}
	inst.ServerName = ServerName
	return inst, nil
}
