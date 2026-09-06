// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const listenEventsBackfillKey = "listen_events_backfilled"

type ListenEvent struct {
	ID         int64
	UserID     string
	TrackID    string
	PlayedAt   time.Time
	TrackTitle string
	ArtistName string
	AlbumID    string
	AlbumTitle string
	DurationMs int
	CoverArtID string
}

type ListenEventJSON struct {
	ID         int64  `json:"id"`
	TrackID    string `json:"trackId"`
	TrackTitle string `json:"trackTitle"`
	ArtistName string `json:"artistName"`
	AlbumID    string `json:"albumId"`
	AlbumTitle string `json:"albumTitle"`
	DurationMs int    `json:"durationMs"`
	CoverArtID string `json:"coverArtId"`
	PlayedAt   string `json:"playedAt"`
}

func (e ListenEvent) JSON() ListenEventJSON {
	return ListenEventJSON{
		ID:         e.ID,
		TrackID:    e.TrackID,
		TrackTitle: e.TrackTitle,
		ArtistName: e.ArtistName,
		AlbumID:    e.AlbumID,
		AlbumTitle: e.AlbumTitle,
		DurationMs: e.DurationMs,
		CoverArtID: e.CoverArtID,
		PlayedAt:   e.PlayedAt.UTC().Format(time.RFC3339),
	}
}

type ListenEventsQuery struct {
	Limit  int
	Offset int
	Since  int64
	Until  int64
	Search string
}

func (s *ListenStore) InsertListenEvent(userID string, input ListenUpsertInput, playedAt int64) error {
	if input.TrackID == "" {
		return fmt.Errorf("track id is required")
	}
	if playedAt <= 0 {
		playedAt = nowUnix()
	}
	_, err := s.db.exec(
		`INSERT INTO listen_events (
			user_id, track_id, played_at, track_title, artist_name, album_id, album_title, duration_ms, cover_art_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, input.TrackID, playedAt,
		input.TrackTitle, input.ArtistName, input.AlbumID, input.AlbumTitle, input.DurationMs, input.CoverArtID,
	)
	return err
}

func (s *ListenStore) ListListenEvents(userID string, q ListenEventsQuery) ([]ListenEvent, error) {
	if q.Limit <= 0 {
		q.Limit = 50
	}
	if q.Limit > 500 {
		q.Limit = 500
	}
	if q.Offset < 0 {
		q.Offset = 0
	}

	var b strings.Builder
	args := make([]any, 0, 10)
	b.WriteString(`SELECT id, user_id, track_id, played_at, track_title, artist_name, album_id, album_title, duration_ms, cover_art_id
		FROM listen_events WHERE user_id = ?`)
	args = append(args, userID)

	if q.Since > 0 {
		b.WriteString(` AND played_at >= ?`)
		args = append(args, q.Since)
	}
	if q.Until > 0 {
		b.WriteString(` AND played_at < ?`)
		args = append(args, q.Until)
	}
	if search := strings.TrimSpace(q.Search); search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		b.WriteString(` AND (LOWER(track_title) LIKE ? OR LOWER(artist_name) LIKE ? OR LOWER(album_title) LIKE ?)`)
		args = append(args, pattern, pattern, pattern)
	}

	b.WriteString(` ORDER BY played_at DESC, id DESC LIMIT ? OFFSET ?`)
	args = append(args, q.Limit, q.Offset)

	rows, err := s.db.query(b.String(), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return s.scanListenEventRows(rows)
}

func (s *ListenStore) ListenEventYears(userID string) ([]int, error) {
	yearExpr := s.db.dialect.YearFromUnix("played_at")
	rows, err := s.db.query(
		`SELECT DISTINCT `+yearExpr+`
		 FROM listen_events WHERE user_id = ?
		 ORDER BY 1 DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	years := make([]int, 0, 4)
	for rows.Next() {
		var year int
		if err := rows.Scan(&year); err != nil {
			return nil, err
		}
		if year > 0 {
			years = append(years, year)
		}
	}
	return years, rows.Err()
}

func (s *ListenStore) ClearHistory(userID string) error {
	if _, err := s.db.exec(`DELETE FROM listen_events WHERE user_id = ?`, userID); err != nil {
		return err
	}
	_, err := s.db.exec(`DELETE FROM listen_progress WHERE user_id = ?`, userID)
	return err
}

func (s *ListenStore) BackfillListenEvents() error {
	var done int
	err := s.db.queryRow(
		`SELECT COUNT(*) FROM app_settings WHERE key = ?`,
		listenEventsBackfillKey,
	).Scan(&done)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if done > 0 {
		return nil
	}

	_, err = s.db.exec(
		`INSERT INTO listen_events (
			user_id, track_id, played_at, track_title, artist_name, album_id, album_title, duration_ms, cover_art_id
		)
		SELECT user_id, track_id, last_played_at, track_title, artist_name, album_id, album_title, duration_ms, cover_art_id
		FROM listen_progress
		WHERE play_count > 0 AND last_played_at > 0`,
	)
	if err != nil {
		return err
	}

	_, err = s.db.exec(
		`INSERT INTO app_settings (key, value) VALUES (?, '1')
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		listenEventsBackfillKey,
	)
	return err
}

func (s *ListenStore) scanListenEventRows(rows *sql.Rows) ([]ListenEvent, error) {
	items := make([]ListenEvent, 0, 32)
	for rows.Next() {
		var item ListenEvent
		var playedUnix int64
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.TrackID, &playedUnix,
			&item.TrackTitle, &item.ArtistName, &item.AlbumID, &item.AlbumTitle, &item.DurationMs, &item.CoverArtID,
		); err != nil {
			return nil, err
		}
		item.PlayedAt = time.Unix(playedUnix, 0)
		items = append(items, item)
	}
	return items, rows.Err()
}
