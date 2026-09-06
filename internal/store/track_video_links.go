// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	VideoSourceLocal     = "local"
	VideoSourceInvidious = "invidious"
	VideoSourceYouTube   = "youtube"
)

type TrackVideoLink struct {
	UserID    string
	TrackID   string
	Source    string
	VideoID   string
	Title     string
	UpdatedAt time.Time
}

type UpsertTrackVideoLinkInput struct {
	UserID  string
	TrackID string
	Source  string
	VideoID string
	Title   string
}

type TrackVideoLinkStore struct {
	db *DB
}

func NewTrackVideoLinkStore(db *DB) *TrackVideoLinkStore {
	return &TrackVideoLinkStore{db: db}
}

func NormalizeVideoSource(source string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case VideoSourceLocal:
		return VideoSourceLocal, nil
	case VideoSourceInvidious:
		return VideoSourceInvidious, nil
	case VideoSourceYouTube:
		return VideoSourceYouTube, nil
	default:
		return "", fmt.Errorf("unsupported video source")
	}
}

func (s *TrackVideoLinkStore) Get(userID, trackID string) (TrackVideoLink, error) {
	row := s.db.queryRow(
		`SELECT user_id, track_id, source, video_id, title, updated_at
		 FROM track_video_links WHERE user_id = ? AND track_id = ?`,
		userID, trackID,
	)
	var link TrackVideoLink
	var updated int64
	err := row.Scan(&link.UserID, &link.TrackID, &link.Source, &link.VideoID, &link.Title, &updated)
	if err != nil {
		return TrackVideoLink{}, err
	}
	link.UpdatedAt = time.Unix(updated, 0)
	return link, nil
}

func (s *TrackVideoLinkStore) Upsert(input UpsertTrackVideoLinkInput) (TrackVideoLink, error) {
	source, err := NormalizeVideoSource(input.Source)
	if err != nil {
		return TrackVideoLink{}, err
	}
	trackID := strings.TrimSpace(input.TrackID)
	videoID := strings.TrimSpace(input.VideoID)
	if trackID == "" || videoID == "" {
		return TrackVideoLink{}, fmt.Errorf("track id and video id are required")
	}
	now := nowUnix()
	_, err = s.db.exec(
		`INSERT INTO track_video_links (user_id, track_id, source, video_id, title, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id, track_id) DO UPDATE SET
		   source = excluded.source,
		   video_id = excluded.video_id,
		   title = excluded.title,
		   updated_at = excluded.updated_at`,
		input.UserID, trackID, source, videoID, strings.TrimSpace(input.Title), now,
	)
	if err != nil {
		return TrackVideoLink{}, err
	}
	return s.Get(input.UserID, trackID)
}

func (s *TrackVideoLinkStore) Delete(userID, trackID string) error {
	res, err := s.db.exec(
		`DELETE FROM track_video_links WHERE user_id = ? AND track_id = ?`,
		userID, trackID,
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
