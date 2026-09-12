// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"melovian/internal/api/apishared"
	"melovian/internal/localmusic"
	"melovian/internal/store"
	"melovian/internal/subsonicserver"
)

type subsonicAuthAdapter struct {
	auth    *store.AuthStore
	cfg     bool
	limiter *apishared.RateLimiter
}

func (a subsonicAuthAdapter) AuthRequired() bool {
	return a.auth != nil && a.auth.Enabled() && a.cfg
}

func (a subsonicAuthAdapter) Authenticate(username, password string) (string, error) {
	if a.auth == nil {
		return "", errors.New("auth unavailable")
	}
	if a.limited(username) {
		return "", errors.New("too many failed attempts, try again later")
	}
	user, err := a.auth.AuthenticateSubsonic(username, password)
	if err != nil {
		a.recordFailure(username)
		return "", err
	}
	a.resetFailures(username)
	return user.ID, nil
}

func (a subsonicAuthAdapter) AuthenticateToken(username, token, salt string) (string, error) {
	if a.limited(username) {
		return "", errors.New("too many failed attempts, try again later")
	}
	secret, err := a.auth.SubsonicAPISecret(username)
	if err != nil {
		a.recordFailure(username)
		return "", err
	}
	if !subsonicserver.VerifyToken(secret, token, salt) {
		a.recordFailure(username)
		return "", errors.New("invalid token")
	}
	user, err := a.auth.GetUserByUsername(username)
	if err != nil {
		return "", err
	}
	a.resetFailures(username)
	return user.ID, nil
}

func (a subsonicAuthAdapter) limited(username string) bool {
	return a.limiter != nil && a.limiter.Blocked("rest|"+username)
}

func (a subsonicAuthAdapter) recordFailure(username string) {
	if a.limiter != nil {
		a.limiter.Record("rest|" + username)
	}
}

func (a subsonicAuthAdapter) resetFailures(username string) {
	if a.limiter != nil {
		a.limiter.Reset("rest|" + username)
	}
}

type subsonicProviderAdapter struct {
	server *Server
}

func (a subsonicProviderAdapter) Catalog(ctx context.Context, userID string) (localmusic.Catalog, error) {
	return a.server.localCatalogForUser(userID)
}

func (a subsonicProviderAdapter) Track(ctx context.Context, userID, trackID string) (store.LocalTrack, store.LocalLibrary, error) {
	track, err := a.server.localTracks.GetByID(trackID)
	if err != nil {
		return store.LocalTrack{}, store.LocalLibrary{}, err
	}
	if userID != "" {
		lib, err := a.server.localLibraries.GetForUser(userID, track.LibraryID)
		if err != nil {
			return store.LocalTrack{}, store.LocalLibrary{}, err
		}
		return track, lib, nil
	}
	lib, err := a.server.localLibraries.Get(track.LibraryID)
	if err != nil {
		return store.LocalTrack{}, store.LocalLibrary{}, err
	}
	return track, lib, nil
}

func (a subsonicProviderAdapter) Cover(ctx context.Context, userID, id string) ([]byte, string, error) {
	catalog, err := a.server.localCatalogForUser(userID)
	if err != nil {
		return nil, "", err
	}
	trackID, ok := catalog.CoverTrackID(id)
	if !ok {
		return nil, "", sql.ErrNoRows
	}
	track, lib, err := a.Track(ctx, userID, trackID)
	if err != nil {
		return nil, "", err
	}
	return a.server.coverBytesForTrack(lib, track)
}

func (a subsonicProviderAdapter) StreamPath(ctx context.Context, userID string, track store.LocalTrack, lib store.LocalLibrary) (string, error) {
	if track.Status != store.TrackStatusPresent || track.DuplicateOf != "" {
		return "", sql.ErrNoRows
	}
	return localmusic.ResolveTrackPath(lib.Path, track.AbsPath)
}

func (a subsonicProviderAdapter) LibraryUserID(melovianUserID string) string {
	return subsonicserver.SubsonicLibraryUserID(melovianUserID)
}

func (a subsonicProviderAdapter) ListPlaylists(ctx context.Context, melovianUserID string) ([]subsonicserver.PlaylistSummary, error) {
	userID := a.LibraryUserID(melovianUserID)
	items, err := a.server.listen.ListPlaylists(userID)
	if err != nil {
		return nil, err
	}
	out := make([]subsonicserver.PlaylistSummary, len(items))
	for i, item := range items {
		out[i] = subsonicserver.PlaylistSummary{
			ID:        item.ID,
			Name:      item.Name,
			SongCount: item.TrackCount,
			Duration:  item.DurationMs / 1000,
			Public:    false,
			Created:   item.CreatedAt,
			Changed:   item.UpdatedAt,
			Kind:      item.Kind,
		}
	}
	return out, nil
}

func (a subsonicProviderAdapter) GetPlaylist(ctx context.Context, melovianUserID, playlistID string) (subsonicserver.PlaylistSummary, []localmusic.Song, error) {
	userID := a.LibraryUserID(melovianUserID)
	pl, err := a.server.listen.GetPlaylist(userID, playlistID)
	if err != nil {
		return subsonicserver.PlaylistSummary{}, nil, err
	}
	catalog, err := a.Catalog(ctx, melovianUserID)
	if err != nil {
		return subsonicserver.PlaylistSummary{}, nil, err
	}
	songs := make([]localmusic.Song, 0, len(pl.Tracks))
	for _, track := range pl.Tracks {
		if song, ok := catalog.Song(track.TrackID); ok {
			songs = append(songs, song)
		}
	}
	summary := subsonicserver.PlaylistSummary{
		ID:        pl.ID,
		Name:      pl.Name,
		SongCount: len(songs),
		Duration:  pl.DurationMs / 1000,
		Public:    false,
		Created:   pl.CreatedAt,
		Changed:   pl.UpdatedAt,
		Kind:      pl.Kind,
	}
	return summary, songs, nil
}

func (a subsonicProviderAdapter) ListStarredIDs(ctx context.Context, melovianUserID string) ([]string, error) {
	userID := a.LibraryUserID(melovianUserID)
	items, err := a.server.listen.ListFavorites(userID, 5000)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.TrackID
	}
	return ids, nil
}

func (a subsonicProviderAdapter) Star(ctx context.Context, melovianUserID, trackID string, track store.LocalTrack) error {
	userID := a.LibraryUserID(melovianUserID)
	return a.server.listen.AddFavorite(userID, store.FavoriteTrack{
		TrackID:    trackID,
		TrackTitle: track.Title,
		ArtistName: track.Artist,
		AlbumID:    "",
		AlbumTitle: track.Album,
		DurationMs: track.DurationMs,
		CoverArtID: track.ID,
	})
}

func (a subsonicProviderAdapter) Unstar(ctx context.Context, melovianUserID, trackID string) error {
	userID := a.LibraryUserID(melovianUserID)
	return a.server.listen.RemoveFavorite(userID, trackID)
}

func (a subsonicProviderAdapter) Scrobble(ctx context.Context, melovianUserID string, track store.LocalTrack, playedAt time.Time) error {
	userID := a.LibraryUserID(melovianUserID)
	return a.server.listen.Upsert(userID, store.ListenUpsertInput{
		TrackID:       track.ID,
		Played:        true,
		IncrementPlay: true,
		TrackTitle:    track.Title,
		ArtistName:    track.Artist,
		AlbumTitle:    track.Album,
		DurationMs:    track.DurationMs,
		CoverArtID:    track.ID,
	})
}

func (a subsonicProviderAdapter) ListGenres(ctx context.Context, melovianUserID string) ([]subsonicserver.Genre, error) {
	libraryIDs, err := a.server.libraryIDsForUser(melovianUserID)
	if err != nil {
		return nil, err
	}
	counts, err := a.server.localTracks.ListGenreCounts(libraryIDs)
	if err != nil {
		return nil, err
	}
	out := make([]subsonicserver.Genre, len(counts))
	for i, item := range counts {
		out[i] = subsonicserver.Genre{Name: item.Name, SongCount: item.Count}
	}
	return out, nil
}

func (s *Server) newSubsonicServer() *subsonicserver.Server {
	srv := subsonicserver.New(
		subsonicProviderAdapter{server: s},
		subsonicAuthAdapter{auth: s.auth, cfg: s.cfg.SubsonicServerEffective(), limiter: s.authLimiter},
		s.cfg.SubsonicServerEffective() && s.localLibraryEnabled(),
	)
	srv.SetReadOnly(s.cfg.DemoModeEffective())
	return srv
}
