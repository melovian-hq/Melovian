// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ListenProgress struct {
	UserID       string
	TrackID      string
	PositionMs   int64
	Played       bool
	PlayCount    int
	ListenedMs   int64
	LastPlayedAt time.Time
	TrackTitle   string
	ArtistName   string
	AlbumID      string
	AlbumTitle   string
	DurationMs   int
	CoverArtID   string
}

type ListenProgressJSON struct {
	TrackID      string `json:"trackId"`
	TrackTitle   string `json:"trackTitle"`
	ArtistName   string `json:"artistName"`
	AlbumID      string `json:"albumId"`
	AlbumTitle   string `json:"albumTitle"`
	PositionMs   int64  `json:"positionMs"`
	DurationMs   int    `json:"durationMs"`
	Played       bool   `json:"played"`
	PlayCount    int    `json:"playCount"`
	ListenedMs   int64  `json:"listenedMs"`
	LastPlayedAt string `json:"lastPlayedAt"`
	CoverArtID   string `json:"coverArtId"`
}

func (p ListenProgress) JSON() ListenProgressJSON {
	return ListenProgressJSON{
		TrackID:      p.TrackID,
		TrackTitle:   p.TrackTitle,
		ArtistName:   p.ArtistName,
		AlbumID:      p.AlbumID,
		AlbumTitle:   p.AlbumTitle,
		PositionMs:   p.PositionMs,
		DurationMs:   p.DurationMs,
		Played:       p.Played,
		PlayCount:    p.PlayCount,
		ListenedMs:   p.ListenedMs,
		LastPlayedAt: p.LastPlayedAt.UTC().Format(time.RFC3339),
		CoverArtID:   p.CoverArtID,
	}
}

type MusicPlaylist struct {
	ID          string
	UserID      string
	Name        string
	Kind        string
	RulesJSON   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	TrackCount  int
	DurationMs  int
	CoverArtIDs []string
	Tracks      []PlaylistTrack
}

type PlaylistTrack struct {
	PlaylistID string
	TrackID    string
	Position   int
	TrackTitle string
	ArtistName string
	AlbumID    string
	AlbumTitle string
	DurationMs int
	CoverArtID string
}

type ListenStore struct {
	db *DB
}

func NewListenStore(db *DB) *ListenStore {
	return &ListenStore{db: db}
}

type ListenUpsertInput struct {
	TrackID       string
	PositionMs    int64
	Played        bool
	IncrementPlay bool
	DeltaMs       int64
	TrackTitle    string
	ArtistName    string
	AlbumID       string
	AlbumTitle    string
	DurationMs    int
	CoverArtID    string
	// LastPlayedAt overrides the listen timestamp when non-zero (demo seeding).
	LastPlayedAt int64
}

func clampListenDelta(deltaMs int64, durationMs int) int64 {
	if deltaMs <= 0 {
		return 0
	}
	const maxChunkMs = 15000
	if deltaMs > maxChunkMs {
		deltaMs = maxChunkMs
	}
	if durationMs > 0 && deltaMs > int64(durationMs) {
		deltaMs = int64(durationMs)
	}
	return deltaMs
}

func (s *ListenStore) Upsert(userID string, input ListenUpsertInput) error {
	if input.TrackID == "" {
		return fmt.Errorf("track id is required")
	}

	now := nowUnix()
	playedAt := now
	if input.LastPlayedAt > 0 {
		playedAt = input.LastPlayedAt
	}
	played := 0
	if input.Played {
		played = 1
	}
	playIncrement := 0
	if input.IncrementPlay {
		playIncrement = 1
	}
	deltaMs := clampListenDelta(input.DeltaMs, input.DurationMs)

	_, err := s.db.exec(
		`INSERT INTO listen_progress (
			user_id, track_id, position_ms, played, play_count, listened_ms, last_played_at,
			track_title, artist_name, album_id, album_title, duration_ms, cover_art_id, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, track_id) DO UPDATE SET
			position_ms = excluded.position_ms,
			played = excluded.played,
			play_count = listen_progress.play_count + ?,
			listened_ms = listen_progress.listened_ms + ?,
			last_played_at = excluded.last_played_at,
			track_title = CASE WHEN excluded.track_title != '' THEN excluded.track_title ELSE listen_progress.track_title END,
			artist_name = CASE WHEN excluded.artist_name != '' THEN excluded.artist_name ELSE listen_progress.artist_name END,
			album_id = CASE WHEN excluded.album_id != '' THEN excluded.album_id ELSE listen_progress.album_id END,
			album_title = CASE WHEN excluded.album_title != '' THEN excluded.album_title ELSE listen_progress.album_title END,
			duration_ms = CASE WHEN excluded.duration_ms != 0 THEN excluded.duration_ms ELSE listen_progress.duration_ms END,
			cover_art_id = CASE WHEN excluded.cover_art_id != '' THEN excluded.cover_art_id ELSE listen_progress.cover_art_id END,
			updated_at = excluded.updated_at`,
		userID, input.TrackID, input.PositionMs, played, playIncrement, deltaMs, playedAt,
		input.TrackTitle, input.ArtistName, input.AlbumID, input.AlbumTitle, input.DurationMs, input.CoverArtID, now,
		playIncrement, deltaMs,
	)
	if err != nil {
		return err
	}
	if input.IncrementPlay {
		return s.InsertListenEvent(userID, input, playedAt)
	}
	return nil
}

func (s *ListenStore) Get(userID, trackID string) (ListenProgress, error) {
	return s.scanOne(
		`SELECT user_id, track_id, position_ms, played, play_count, listened_ms, last_played_at,
		        track_title, artist_name, album_id, album_title, duration_ms, cover_art_id
		 FROM listen_progress WHERE user_id = ? AND track_id = ?`,
		userID, trackID,
	)
}

func (s *ListenStore) Delete(userID, trackID string) error {
	res, err := s.db.exec(`DELETE FROM listen_progress WHERE user_id = ? AND track_id = ?`, userID, trackID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *ListenStore) MarkPlayed(userID, trackID string) error {
	now := nowUnix()
	res, err := s.db.exec(
		`UPDATE listen_progress SET played = 1, position_ms = 0, play_count = play_count + 1,
		 last_played_at = ?, updated_at = ? WHERE user_id = ? AND track_id = ?`,
		now, now, userID, trackID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		_, err = s.db.exec(
			`INSERT INTO listen_progress (user_id, track_id, position_ms, played, play_count, listened_ms, last_played_at, updated_at)
			 VALUES (?, ?, 0, 1, 1, 0, ?, ?)`,
			userID, trackID, now, now,
		)
		return err
	}
	return nil
}

func (s *ListenStore) ResumeTracks(userID string, limit int) ([]ListenProgress, error) {
	if limit <= 0 {
		limit = 18
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.query(
		`SELECT user_id, track_id, position_ms, played, play_count, listened_ms, last_played_at,
		        track_title, artist_name, album_id, album_title, duration_ms, cover_art_id
		 FROM listen_progress
		 WHERE user_id = ? AND played = 0 AND position_ms > 5000
		 ORDER BY last_played_at DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return s.scanRows(rows)
}

func (s *ListenStore) History(userID string, limit int) ([]ListenProgress, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.query(
		`SELECT user_id, track_id, position_ms, played, play_count, listened_ms, last_played_at,
		        track_title, artist_name, album_id, album_title, duration_ms, cover_art_id
		 FROM listen_progress WHERE user_id = ?
		 ORDER BY last_played_at DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return s.scanRows(rows)
}

func (s *ListenStore) Batch(userID string, trackIDs []string) (map[string]ListenProgress, error) {
	result := make(map[string]ListenProgress, len(trackIDs))
	if len(trackIDs) == 0 {
		return result, nil
	}
	if len(trackIDs) > 200 {
		trackIDs = trackIDs[:200]
	}

	placeholders := make([]string, len(trackIDs))
	args := make([]any, 0, len(trackIDs)+1)
	args = append(args, userID)
	for i, id := range trackIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	var query strings.Builder
	query.WriteString(`SELECT user_id, track_id, position_ms, played, play_count, listened_ms, last_played_at,
	        track_title, artist_name, album_id, album_title, duration_ms, cover_art_id
	 FROM listen_progress WHERE user_id = ? AND track_id IN (`)
	query.WriteString(strings.Join(placeholders, ","))
	query.WriteByte(')')

	rows, err := s.db.query(query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items, err := s.scanRows(rows)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		result[item.TrackID] = item
	}
	return result, nil
}

type ListenStats struct {
	TotalPlays       int
	UniqueTracks     int
	TotalListeningMs int64
	TopArtists       []StatEntry
	TopTracks        []StatEntry
	TopAlbums        []StatEntry
}

type StatEntry struct {
	Key   string
	Label string
	Count int
}

func (s *ListenStore) Stats(userID string, limit int) (ListenStats, error) {
	if limit <= 0 {
		limit = 8
	}

	var stats ListenStats
	row := s.db.queryRow(
		`SELECT COALESCE(SUM(play_count), 0), COUNT(DISTINCT track_id),
		        COALESCE(SUM(CASE WHEN listened_ms > 0 THEN listened_ms ELSE duration_ms * play_count END), 0)
		 FROM listen_progress WHERE user_id = ?`,
		userID,
	)
	if err := row.Scan(&stats.TotalPlays, &stats.UniqueTracks, &stats.TotalListeningMs); err != nil {
		return stats, err
	}

	artistRows, err := s.db.query(
		`SELECT artist_name,
		        CAST(SUM(CASE WHEN listened_ms > 0 THEN listened_ms ELSE duration_ms * play_count END) / 1000 AS INTEGER) AS weight
		 FROM listen_progress WHERE user_id = ? AND artist_name != ''
		 GROUP BY artist_name ORDER BY weight DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return stats, err
	}
	defer func() { _ = artistRows.Close() }()
	for artistRows.Next() {
		var entry StatEntry
		if err := artistRows.Scan(&entry.Label, &entry.Count); err != nil {
			return stats, err
		}
		entry.Key = entry.Label
		stats.TopArtists = append(stats.TopArtists, entry)
	}

	trackRows, err := s.db.query(
		`SELECT track_id, track_title,
		        CAST(SUM(CASE WHEN listened_ms > 0 THEN listened_ms ELSE duration_ms * play_count END) / 1000 AS INTEGER) AS weight
		 FROM listen_progress WHERE user_id = ? AND track_title != ''
		 GROUP BY track_id ORDER BY weight DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return stats, err
	}
	defer func() { _ = trackRows.Close() }()
	for trackRows.Next() {
		var entry StatEntry
		if err := trackRows.Scan(&entry.Key, &entry.Label, &entry.Count); err != nil {
			return stats, err
		}
		stats.TopTracks = append(stats.TopTracks, entry)
	}

	albumRows, err := s.db.query(
		`SELECT album_id, album_title,
		        CAST(SUM(CASE WHEN listened_ms > 0 THEN listened_ms ELSE duration_ms * play_count END) / 1000 AS INTEGER) AS weight
		 FROM listen_progress WHERE user_id = ? AND album_title != ''
		 GROUP BY album_id ORDER BY weight DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return stats, err
	}
	defer func() { _ = albumRows.Close() }()
	for albumRows.Next() {
		var entry StatEntry
		if err := albumRows.Scan(&entry.Key, &entry.Label, &entry.Count); err != nil {
			return stats, err
		}
		stats.TopAlbums = append(stats.TopAlbums, entry)
	}

	return stats, nil
}

func (s *ListenStore) TopArtistNames(userID string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 5
	}
	rows, err := s.db.query(
		`SELECT artist_name FROM listen_progress WHERE user_id = ? AND artist_name != ''
		 GROUP BY artist_name
		 ORDER BY SUM(CASE WHEN listened_ms > 0 THEN listened_ms ELSE duration_ms * play_count END) DESC
		 LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func (s *ListenStore) scanOne(query string, args ...any) (ListenProgress, error) {
	row := s.db.queryRow(query, args...)
	var item ListenProgress
	var played int
	var lastPlayedUnix int64
	err := row.Scan(
		&item.UserID, &item.TrackID, &item.PositionMs, &played, &item.PlayCount, &item.ListenedMs, &lastPlayedUnix,
		&item.TrackTitle, &item.ArtistName, &item.AlbumID, &item.AlbumTitle, &item.DurationMs, &item.CoverArtID,
	)
	if err != nil {
		return ListenProgress{}, err
	}
	item.Played = played == 1
	item.LastPlayedAt = time.Unix(lastPlayedUnix, 0)
	return item, nil
}

func (s *ListenStore) scanRows(rows *sql.Rows) ([]ListenProgress, error) {
	var items []ListenProgress
	for rows.Next() {
		var item ListenProgress
		var played int
		var lastPlayedUnix int64
		if err := rows.Scan(
			&item.UserID, &item.TrackID, &item.PositionMs, &played, &item.PlayCount, &item.ListenedMs, &lastPlayedUnix,
			&item.TrackTitle, &item.ArtistName, &item.AlbumID, &item.AlbumTitle, &item.DurationMs, &item.CoverArtID,
		); err != nil {
			return nil, err
		}
		item.Played = played == 1
		item.LastPlayedAt = time.Unix(lastPlayedUnix, 0)
		items = append(items, item)
	}
	return items, rows.Err()
}

func newPlaylistID() string {
	return mustRandomHex(16)
}

func (s *ListenStore) ListPlaylists(userID string) ([]MusicPlaylist, error) {
	rows, err := s.db.query(
		`SELECT p.id, p.user_id, p.name, p.kind, p.rules_json, p.created_at, p.updated_at,
		        (SELECT COUNT(*) FROM music_playlist_tracks t WHERE t.playlist_id = p.id) AS track_count,
		        (SELECT COALESCE(SUM(duration_ms), 0) FROM music_playlist_tracks t WHERE t.playlist_id = p.id) AS duration_ms
		 FROM music_playlists p
		 WHERE p.user_id = ? ORDER BY p.updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var playlists []MusicPlaylist
	for rows.Next() {
		var pl MusicPlaylist
		var created, updated int64
		if err := rows.Scan(
			&pl.ID, &pl.UserID, &pl.Name, &pl.Kind, &pl.RulesJSON, &created, &updated, &pl.TrackCount, &pl.DurationMs,
		); err != nil {
			return nil, err
		}
		pl.CreatedAt = time.Unix(created, 0)
		pl.UpdatedAt = time.Unix(updated, 0)
		if pl.Kind == "" {
			pl.Kind = "static"
		}
		playlists = append(playlists, pl)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range playlists {
		covers, err := s.listPlaylistCoverArtIDs(playlists[i].ID, 3)
		if err != nil {
			return nil, err
		}
		playlists[i].CoverArtIDs = covers
	}
	return playlists, nil
}

func (s *ListenStore) listPlaylistCoverArtIDs(playlistID string, limit int) ([]string, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := s.db.query(
		`SELECT cover_art_id
		 FROM music_playlist_tracks
		 WHERE playlist_id = ? AND cover_art_id != ''
		 GROUP BY cover_art_id
		 ORDER BY MIN(position) ASC
		 LIMIT ?`,
		playlistID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *ListenStore) GetPlaylist(userID, playlistID string) (MusicPlaylist, error) {
	var pl MusicPlaylist
	var created, updated int64
	err := s.db.queryRow(
		`SELECT id, user_id, name, kind, rules_json, created_at, updated_at FROM music_playlists
		 WHERE id = ? AND user_id = ?`,
		playlistID, userID,
	).Scan(&pl.ID, &pl.UserID, &pl.Name, &pl.Kind, &pl.RulesJSON, &created, &updated)
	if err != nil {
		return MusicPlaylist{}, err
	}
	pl.CreatedAt = time.Unix(created, 0)
	pl.UpdatedAt = time.Unix(updated, 0)
	if pl.Kind == "" {
		pl.Kind = "static"
	}

	tracks, err := s.listPlaylistTracks(playlistID)
	if err != nil {
		return MusicPlaylist{}, err
	}
	pl.Tracks = tracks
	pl.TrackCount = len(tracks)
	pl.DurationMs = playlistDurationMs(tracks)
	return pl, nil
}

func playlistDurationMs(tracks []PlaylistTrack) int {
	total := 0
	for _, track := range tracks {
		total += track.DurationMs
	}
	return total
}

func (s *ListenStore) listPlaylistTracks(playlistID string) ([]PlaylistTrack, error) {
	rows, err := s.db.query(
		`SELECT playlist_id, track_id, position, track_title, artist_name, album_id, album_title, duration_ms, cover_art_id
		 FROM music_playlist_tracks WHERE playlist_id = ? ORDER BY position ASC`,
		playlistID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var tracks []PlaylistTrack
	for rows.Next() {
		var t PlaylistTrack
		if err := rows.Scan(&t.PlaylistID, &t.TrackID, &t.Position, &t.TrackTitle, &t.ArtistName, &t.AlbumID, &t.AlbumTitle, &t.DurationMs, &t.CoverArtID); err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

type CreatePlaylistOpts struct {
	Name      string
	Kind      string
	RulesJSON string
}

func (s *ListenStore) CreatePlaylist(userID, name string) (MusicPlaylist, error) {
	return s.CreatePlaylistOpts(userID, CreatePlaylistOpts{Name: name})
}

func (s *ListenStore) CreatePlaylistOpts(userID string, opts CreatePlaylistOpts) (MusicPlaylist, error) {
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		return MusicPlaylist{}, fmt.Errorf("playlist name is required")
	}
	kind := strings.TrimSpace(opts.Kind)
	if kind == "" {
		kind = "static"
	}
	if kind != "static" && kind != "smart" {
		return MusicPlaylist{}, fmt.Errorf("invalid playlist kind")
	}
	rulesJSON := opts.RulesJSON
	if kind == "smart" && strings.TrimSpace(rulesJSON) == "" {
		return MusicPlaylist{}, fmt.Errorf("smart playlist rules are required")
	}
	if kind == "static" {
		rulesJSON = ""
	}

	id := newPlaylistID()
	now := nowUnix()
	_, err := s.db.exec(
		`INSERT INTO music_playlists (id, user_id, name, kind, rules_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, userID, name, kind, rulesJSON, now, now,
	)
	if err != nil {
		return MusicPlaylist{}, err
	}
	return MusicPlaylist{
		ID:        id,
		UserID:    userID,
		Name:      name,
		Kind:      kind,
		RulesJSON: rulesJSON,
		CreatedAt: time.Unix(now, 0),
		UpdatedAt: time.Unix(now, 0),
		Tracks:    []PlaylistTrack{},
	}, nil
}

func (s *ListenStore) RenamePlaylist(userID, playlistID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("playlist name is required")
	}
	now := nowUnix()
	res, err := s.db.exec(
		`UPDATE music_playlists SET name = ?, updated_at = ? WHERE id = ? AND user_id = ?`,
		name, now, playlistID, userID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *ListenStore) DeletePlaylist(userID, playlistID string) error {
	res, err := s.db.exec(`DELETE FROM music_playlists WHERE id = ? AND user_id = ?`, playlistID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *ListenStore) SetPlaylistTracks(userID, playlistID string, tracks []PlaylistTrack) error {
	pl, err := s.GetPlaylist(userID, playlistID)
	if err != nil {
		return err
	}
	_ = pl

	tx, err := s.db.begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM music_playlist_tracks WHERE playlist_id = ?`, playlistID); err != nil {
		return err
	}

	for i, track := range tracks {
		_, err := tx.Exec(
			`INSERT INTO music_playlist_tracks (
				playlist_id, track_id, position, track_title, artist_name, album_id, album_title, duration_ms, cover_art_id
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			playlistID, track.TrackID, i, track.TrackTitle, track.ArtistName, track.AlbumID, track.AlbumTitle, track.DurationMs, track.CoverArtID,
		)
		if err != nil {
			return err
		}
	}

	now := nowUnix()
	if _, err := tx.Exec(`UPDATE music_playlists SET updated_at = ? WHERE id = ?`, now, playlistID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *ListenStore) AddPlaylistTrack(userID, playlistID string, track PlaylistTrack) error {
	pl, err := s.GetPlaylist(userID, playlistID)
	if err != nil {
		return err
	}

	position := len(pl.Tracks)
	_, err = s.db.exec(
		`INSERT INTO music_playlist_tracks (
			playlist_id, track_id, position, track_title, artist_name, album_id, album_title, duration_ms, cover_art_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(playlist_id, track_id) DO NOTHING`,
		playlistID, track.TrackID, position, track.TrackTitle, track.ArtistName, track.AlbumID, track.AlbumTitle, track.DurationMs, track.CoverArtID,
	)
	if err != nil {
		return err
	}

	now := nowUnix()
	_, err = s.db.exec(`UPDATE music_playlists SET updated_at = ? WHERE id = ?`, now, playlistID)
	return err
}

func (s *ListenStore) RemovePlaylistTrack(userID, playlistID, trackID string) error {
	var owner string
	err := s.db.queryRow(
		`SELECT user_id FROM music_playlists WHERE id = ?`,
		playlistID,
	).Scan(&owner)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		return err
	}
	if owner != userID {
		return sql.ErrNoRows
	}

	var position int
	err = s.db.queryRow(
		`SELECT position FROM music_playlist_tracks WHERE playlist_id = ? AND track_id = ?`,
		playlistID, trackID,
	).Scan(&position)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		return err
	}

	tx, err := s.db.begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(
		`DELETE FROM music_playlist_tracks WHERE playlist_id = ? AND track_id = ?`,
		playlistID, trackID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`UPDATE music_playlist_tracks SET position = position - 1 WHERE playlist_id = ? AND position > ?`,
		playlistID, position,
	); err != nil {
		return err
	}

	now := nowUnix()
	if _, err := tx.Exec(
		`UPDATE music_playlists SET updated_at = ? WHERE id = ?`,
		now, playlistID,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (pl MusicPlaylist) ToJSON() map[string]any {
	tracks := make([]map[string]any, 0, len(pl.Tracks))
	for _, t := range pl.Tracks {
		tracks = append(tracks, t.ToJSON())
	}
	durationMs := pl.DurationMs
	if durationMs == 0 && len(pl.Tracks) > 0 {
		durationMs = playlistDurationMs(pl.Tracks)
	}
	coverArtIds := pl.CoverArtIDs
	if len(coverArtIds) == 0 && len(pl.Tracks) > 0 {
		seen := make(map[string]struct{}, 3)
		for _, track := range pl.Tracks {
			id := strings.TrimSpace(track.CoverArtID)
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			coverArtIds = append(coverArtIds, id)
			if len(coverArtIds) >= 3 {
				break
			}
		}
	}

	return map[string]any{
		"id":          pl.ID,
		"name":        pl.Name,
		"kind":        playlistKind(pl),
		"rulesJson":   pl.RulesJSON,
		"createdAt":   pl.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":   pl.UpdatedAt.UTC().Format(time.RFC3339),
		"trackCount":  playlistTrackCount(pl),
		"durationMs":  durationMs,
		"coverArtIds": coverArtIds,
		"tracks":      tracks,
	}
}

func playlistKind(pl MusicPlaylist) string {
	if pl.Kind == "" {
		return "static"
	}
	return pl.Kind
}

func playlistTrackCount(pl MusicPlaylist) int {
	if len(pl.Tracks) > 0 {
		return len(pl.Tracks)
	}
	return pl.TrackCount
}

func (t PlaylistTrack) ToJSON() map[string]any {
	return map[string]any{
		"trackId":    t.TrackID,
		"trackTitle": t.TrackTitle,
		"artistName": t.ArtistName,
		"albumId":    t.AlbumID,
		"albumTitle": t.AlbumTitle,
		"durationMs": t.DurationMs,
		"coverArtId": t.CoverArtID,
		"position":   t.Position,
	}
}

type FavoriteTrack struct {
	UserID      string
	TrackID     string
	TrackTitle  string
	ArtistName  string
	AlbumID     string
	AlbumTitle  string
	DurationMs  int
	CoverArtID  string
	FavoritedAt time.Time
}

func (s *ListenStore) ListFavorites(userID string, limit int) ([]FavoriteTrack, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.query(
		`SELECT user_id, track_id, track_title, artist_name, album_id, album_title, duration_ms, cover_art_id, favorited_at
		 FROM music_favorites WHERE user_id = ? ORDER BY favorited_at DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return s.scanFavoriteRows(rows)
}

func (s *ListenStore) AddFavorite(userID string, track FavoriteTrack) error {
	if track.TrackID == "" {
		return fmt.Errorf("track id is required")
	}
	now := nowUnix()
	_, err := s.db.exec(
		`INSERT INTO music_favorites (
			user_id, track_id, track_title, artist_name, album_id, album_title, duration_ms, cover_art_id, favorited_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, track_id) DO UPDATE SET
			track_title = excluded.track_title,
			artist_name = excluded.artist_name,
			album_id = excluded.album_id,
			album_title = excluded.album_title,
			duration_ms = excluded.duration_ms,
			cover_art_id = excluded.cover_art_id,
			favorited_at = excluded.favorited_at`,
		userID, track.TrackID, track.TrackTitle, track.ArtistName, track.AlbumID, track.AlbumTitle, track.DurationMs, track.CoverArtID, now,
	)
	return err
}

func (s *ListenStore) RemoveFavorite(userID, trackID string) error {
	res, err := s.db.exec(`DELETE FROM music_favorites WHERE user_id = ? AND track_id = ?`, userID, trackID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *ListenStore) scanFavoriteRows(rows *sql.Rows) ([]FavoriteTrack, error) {
	var items []FavoriteTrack
	for rows.Next() {
		var item FavoriteTrack
		var favoritedUnix int64
		if err := rows.Scan(
			&item.UserID, &item.TrackID, &item.TrackTitle, &item.ArtistName, &item.AlbumID, &item.AlbumTitle, &item.DurationMs, &item.CoverArtID, &favoritedUnix,
		); err != nil {
			return nil, err
		}
		item.FavoritedAt = time.Unix(favoritedUnix, 0)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (f FavoriteTrack) ToJSON() map[string]any {
	return map[string]any{
		"trackId":     f.TrackID,
		"trackTitle":  f.TrackTitle,
		"artistName":  f.ArtistName,
		"albumId":     f.AlbumID,
		"albumTitle":  f.AlbumTitle,
		"durationMs":  f.DurationMs,
		"coverArtId":  f.CoverArtID,
		"favoritedAt": f.FavoritedAt.UTC().Format(time.RFC3339),
	}
}
