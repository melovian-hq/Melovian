// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"melovian/internal/dlna"
	"melovian/internal/jukebox"
	"melovian/internal/store"
	"melovian/internal/subsonicserver"
)

type dlnaCatalog struct {
	server *Server
}

func (s *Server) dlnaCatalogAdapter() *dlnaCatalog {
	return &dlnaCatalog{server: s}
}

func (c *dlnaCatalog) Entries() []dlna.CatalogEntry {
	catalog, err := c.server.localCatalogForUser("")
	if err != nil {
		return nil
	}
	out := make([]dlna.CatalogEntry, 0, len(catalog.Artists)+len(catalog.Albums))
	for _, artist := range catalog.Artists {
		out = append(out, dlna.CatalogEntry{
			ID:    artist.ID,
			Title: artist.Name,
			Kind:  "artist",
		})
	}
	for _, album := range catalog.Albums {
		out = append(out, dlna.CatalogEntry{
			ID:    album.ID,
			Title: fmt.Sprintf("%s - %s", album.Artist, album.Name),
			Kind:  "album",
		})
	}
	return out
}

func (a subsonicProviderAdapter) CreatePlaylist(ctx context.Context, melovianUserID, name string, songIDs []string) (subsonicserver.PlaylistSummary, error) {
	userID := a.LibraryUserID(melovianUserID)
	pl, err := a.server.listen.CreatePlaylist(userID, name)
	if err != nil {
		return subsonicserver.PlaylistSummary{}, err
	}
	if len(songIDs) > 0 {
		if err := a.setPlaylistSongs(ctx, melovianUserID, pl.ID, songIDs); err != nil {
			return subsonicserver.PlaylistSummary{}, err
		}
		pl, err = a.server.listen.GetPlaylist(userID, pl.ID)
		if err != nil {
			return subsonicserver.PlaylistSummary{}, err
		}
	}
	return playlistSummary(pl), nil
}

func (a subsonicProviderAdapter) UpdatePlaylist(ctx context.Context, melovianUserID, playlistID, name string, songIDs []string) (subsonicserver.PlaylistSummary, error) {
	userID := a.LibraryUserID(melovianUserID)
	if strings.TrimSpace(name) != "" {
		if err := a.server.listen.RenamePlaylist(userID, playlistID, name); err != nil {
			return subsonicserver.PlaylistSummary{}, err
		}
	}
	if songIDs != nil {
		if err := a.setPlaylistSongs(ctx, melovianUserID, playlistID, songIDs); err != nil {
			return subsonicserver.PlaylistSummary{}, err
		}
	}
	pl, err := a.server.listen.GetPlaylist(userID, playlistID)
	if err != nil {
		return subsonicserver.PlaylistSummary{}, err
	}
	return playlistSummary(pl), nil
}

func (a subsonicProviderAdapter) DeletePlaylist(ctx context.Context, melovianUserID, playlistID string) error {
	return a.server.listen.DeletePlaylist(a.LibraryUserID(melovianUserID), playlistID)
}

func (a subsonicProviderAdapter) setPlaylistSongs(ctx context.Context, melovianUserID, playlistID string, songIDs []string) error {
	userID := a.LibraryUserID(melovianUserID)
	catalog, err := a.Catalog(ctx, melovianUserID)
	if err != nil {
		return err
	}
	tracks := make([]store.PlaylistTrack, 0, len(songIDs))
	for i, id := range songIDs {
		song, ok := catalog.Song(id)
		if !ok {
			continue
		}
		tracks = append(tracks, store.PlaylistTrack{
			PlaylistID: playlistID,
			TrackID:    song.ID,
			Position:   i,
			TrackTitle: song.Title,
			ArtistName: song.Artist,
			AlbumID:    song.AlbumID,
			AlbumTitle: song.Album,
			DurationMs: song.Duration * 1000,
			CoverArtID: song.CoverArt,
		})
	}
	return a.server.listen.SetPlaylistTracks(userID, playlistID, tracks)
}

func playlistSummary(pl store.MusicPlaylist) subsonicserver.PlaylistSummary {
	return subsonicserver.PlaylistSummary{
		ID:        pl.ID,
		Name:      pl.Name,
		SongCount: pl.TrackCount,
		Duration:  pl.DurationMs / 1000,
		Public:    false,
		Created:   pl.CreatedAt,
		Changed:   pl.UpdatedAt,
		Kind:      pl.Kind,
	}
}

func (a subsonicProviderAdapter) ListShares(ctx context.Context, melovianUserID string) ([]subsonicserver.ShareSummary, error) {
	userID := a.LibraryUserID(melovianUserID)
	items, err := a.server.shares.ListForUser(userID)
	if err != nil {
		return nil, err
	}
	base := a.server.publicBaseURL()
	out := make([]subsonicserver.ShareSummary, len(items))
	for i, item := range items {
		out[i] = shareSummary(item, base)
	}
	return out, nil
}

func (a subsonicProviderAdapter) CreateShare(ctx context.Context, melovianUserID, resourceID, description string, expires *time.Time) (subsonicserver.ShareSummary, error) {
	resourceType := "song"
	if strings.HasPrefix(resourceID, "alb_") {
		resourceType = "album"
	}
	share, err := a.server.shares.Create(store.CreateShareInput{
		UserID:       a.LibraryUserID(melovianUserID),
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Description:  description,
		ExpiresAt:    expires,
	})
	if err != nil {
		return subsonicserver.ShareSummary{}, err
	}
	return shareSummary(share, a.server.publicBaseURL()), nil
}

func (a subsonicProviderAdapter) DeleteShare(ctx context.Context, melovianUserID, shareID string) error {
	return a.server.shares.Delete(a.LibraryUserID(melovianUserID), shareID)
}

func (a subsonicProviderAdapter) JukeboxStatus(ctx context.Context) jukebox.Status {
	return a.server.jukebox.Status()
}

func (a subsonicProviderAdapter) JukeboxControl(ctx context.Context, action string, track *jukebox.Track, positionSec int, gain float64, queue []jukebox.Track) jukebox.Status {
	return a.server.jukebox.Control(action, track, positionSec, gain, queue)
}

func shareSummary(share store.Share, baseURL string) subsonicserver.ShareSummary {
	return subsonicserver.ShareSummary{
		ID:          share.ID,
		Token:       share.Token,
		URL:         strings.TrimRight(baseURL, "/") + "/share/" + share.Token,
		Description: share.Description,
		ResourceID:  share.ResourceID,
		Expires:     share.ExpiresAt,
		Created:     share.CreatedAt,
		VisitCount:  share.VisitCount,
	}
}
