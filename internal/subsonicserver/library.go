// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonicserver

import (
	"context"
	"strings"
	"time"

	"melovian/internal/localmusic"
	"melovian/internal/store"
)

type Genre struct {
	Name      string
	SongCount int
}

type PlaylistSummary struct {
	ID        string
	Name      string
	SongCount int
	Duration  int
	Public    bool
	Created   time.Time
	Changed   time.Time
	Kind      string
}

type LibraryProvider interface {
	CatalogProvider
	LibraryUserID(melovianUserID string) string
	ListPlaylists(ctx context.Context, melovianUserID string) ([]PlaylistSummary, error)
	GetPlaylist(ctx context.Context, melovianUserID, playlistID string) (PlaylistSummary, []localmusic.Song, error)
	ListStarredIDs(ctx context.Context, melovianUserID string) ([]string, error)
	Star(ctx context.Context, melovianUserID, trackID string, track store.LocalTrack) error
	Unstar(ctx context.Context, melovianUserID, trackID string) error
	Scrobble(ctx context.Context, melovianUserID string, track store.LocalTrack, playedAt time.Time) error
	ListGenres(ctx context.Context, melovianUserID string) ([]Genre, error)
}

func SubsonicLibraryUserID(melovianUserID string) string {
	if strings.TrimSpace(melovianUserID) == "" {
		return "subsonic"
	}
	return "subsonic:" + melovianUserID
}
