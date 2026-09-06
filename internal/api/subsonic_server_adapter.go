// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"melovian/internal/localmusic"
	"melovian/internal/store"
	"melovian/internal/subsonicserver"
)

type subsonicAuthAdapter struct {
	auth *store.AuthStore
	cfg  bool
}

func (a subsonicAuthAdapter) AuthRequired() bool {
	return a.auth != nil && a.auth.Enabled() && a.cfg
}

func (a subsonicAuthAdapter) Authenticate(username, password string) (string, error) {
	if a.auth == nil {
		return "", errors.New("auth unavailable")
	}
	user, err := a.auth.Authenticate(username, password)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

func (a subsonicAuthAdapter) AuthenticateToken(username, token, salt string) (string, error) {
	secret, err := a.auth.SubsonicAPISecret(username)
	if err != nil {
		return "", err
	}
	if !subsonicserver.VerifyToken(secret, token, salt) {
		return "", errors.New("invalid token")
	}
	user, err := a.auth.GetUserByUsername(username)
	if err != nil {
		return "", err
	}
	return user.ID, nil
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

func (s *Server) libraryIDsForUser(userID string) ([]string, error) {
	multi, err := s.preferences.GetMultiLocalLibrary(userID)
	if err != nil {
		return nil, err
	}
	if multi {
		libs, err := s.localLibraries.ListForUser(userID)
		if err != nil {
			return nil, err
		}
		ids := make([]string, len(libs))
		for i, lib := range libs {
			ids[i] = lib.ID
		}
		return ids, nil
	}
	lib, err := s.localLibraries.GetActiveForUser(userID)
	if err != nil {
		return nil, err
	}
	return []string{lib.ID}, nil
}

func (s *Server) newSubsonicServer() *subsonicserver.Server {
	return subsonicserver.New(
		subsonicProviderAdapter{server: s},
		subsonicAuthAdapter{auth: s.auth, cfg: s.cfg.SubsonicServerEffective()},
		s.cfg.SubsonicServerEffective() && s.localLibraryEnabled(),
	)
}

func (s *Server) localCatalogForUser(userID string) (localmusic.Catalog, error) {
	multi, err := s.preferences.GetMultiLocalLibrary(userID)
	if err != nil {
		return localmusic.Catalog{}, err
	}
	if multi {
		return s.mergedLocalCatalog(userID)
	}
	lib, err := s.localLibraries.GetActiveForUser(userID)
	if err != nil {
		return localmusic.Catalog{}, err
	}
	version := localmusic.CatalogVersion(lib)
	if catalog, ok := s.catalogCache.Get(lib.ID, version); ok {
		return catalog, nil
	}
	catalog, err := localmusic.LoadCatalog(s.localTracks, lib.ID)
	if err != nil {
		return localmusic.Catalog{}, err
	}
	s.catalogCache.Set(lib.ID, version, catalog)
	return catalog, nil
}

func (s *Server) mergedLocalCatalog(userID string) (localmusic.Catalog, error) {
	cacheKey := "merged:" + userID
	libs, err := s.localLibraries.ListForUser(userID)
	if err != nil {
		return localmusic.Catalog{}, err
	}
	if len(libs) == 0 {
		return localmusic.Catalog{}, sql.ErrNoRows
	}
	var version uint64
	for _, lib := range libs {
		version ^= localmusic.CatalogVersion(lib)
	}
	if catalog, ok := s.catalogCache.Get(cacheKey, version); ok {
		return catalog, nil
	}
	var tracks []store.CatalogTrack
	for _, lib := range libs {
		items, listErr := s.localTracks.ListForCatalog(lib.ID)
		if listErr != nil {
			return localmusic.Catalog{}, listErr
		}
		tracks = append(tracks, items...)
	}
	catalog := localmusic.BuildCatalog(tracks)
	s.catalogCache.Set(cacheKey, version, catalog)
	return catalog, nil
}

func (s *Server) coverBytesForTrack(lib store.LocalLibrary, track store.LocalTrack) ([]byte, string, error) {
	data, mime, ok := s.coverBytesFromTrackFile(lib, track.ID)
	if !ok {
		return nil, "", sql.ErrNoRows
	}
	return data, mime, nil
}
