// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"database/sql"
	"strings"
)

// DownloadedTrack describes a track cached on disk for offline playback.
type DownloadedTrack struct {
	InstanceID  string `json:"instanceId"`
	TrackID     string `json:"trackId"`
	Path        string `json:"path"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
	TrackTitle  string `json:"trackTitle"`
	ArtistName  string `json:"artistName"`
	CreatedAt   int64  `json:"createdAt"`
}

// DownloadStore persists metadata about offline track downloads.
type DownloadStore struct {
	db *DB
}

func NewDownloadStore(db *DB) *DownloadStore {
	return &DownloadStore{db: db}
}

// Upsert records (or replaces) a downloaded track entry.
func (s *DownloadStore) Upsert(t DownloadedTrack) error {
	if t.CreatedAt == 0 {
		t.CreatedAt = nowUnix()
	}
	_, err := s.db.exec(
		`INSERT INTO downloaded_tracks
			(instance_id, track_id, path, content_type, size, track_title, artist_name, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(instance_id, track_id) DO UPDATE SET
			path = excluded.path,
			content_type = excluded.content_type,
			size = excluded.size,
			track_title = excluded.track_title,
			artist_name = excluded.artist_name,
			created_at = excluded.created_at`,
		t.InstanceID, t.TrackID, t.Path, t.ContentType, t.Size,
		t.TrackTitle, t.ArtistName, t.CreatedAt,
	)
	return err
}

// Get returns the stored entry for a track, or sql.ErrNoRows when absent.
func (s *DownloadStore) Get(instanceID, trackID string) (DownloadedTrack, error) {
	var t DownloadedTrack
	err := s.db.queryRow(
		`SELECT instance_id, track_id, path, content_type, size, track_title, artist_name, created_at
		 FROM downloaded_tracks WHERE instance_id = ? AND track_id = ?`,
		instanceID, trackID,
	).Scan(&t.InstanceID, &t.TrackID, &t.Path, &t.ContentType, &t.Size,
		&t.TrackTitle, &t.ArtistName, &t.CreatedAt)
	return t, err
}

// List returns all downloaded tracks for an instance, newest first.
func (s *DownloadStore) List(instanceID string) ([]DownloadedTrack, error) {
	rows, err := s.db.query(
		`SELECT instance_id, track_id, path, content_type, size, track_title, artist_name, created_at
		 FROM downloaded_tracks WHERE instance_id = ? ORDER BY created_at DESC`,
		instanceID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []DownloadedTrack
	for rows.Next() {
		var t DownloadedTrack
		if err := rows.Scan(&t.InstanceID, &t.TrackID, &t.Path, &t.ContentType,
			&t.Size, &t.TrackTitle, &t.ArtistName, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Stats returns total cached bytes and track count for an instance.
func (s *DownloadStore) Stats(instanceID string) (totalSize int64, count int, err error) {
	err = s.db.queryRow(
		`SELECT COALESCE(SUM(size), 0), COUNT(*) FROM downloaded_tracks WHERE instance_id = ?`,
		instanceID,
	).Scan(&totalSize, &count)
	return totalSize, count, err
}

// ListOldest returns cached tracks ordered oldest-first for eviction.
func (s *DownloadStore) ListOldest(instanceID string) ([]DownloadedTrack, error) {
	rows, err := s.db.query(
		`SELECT instance_id, track_id, path, content_type, size, track_title, artist_name, created_at
		 FROM downloaded_tracks WHERE instance_id = ? ORDER BY created_at ASC`,
		instanceID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []DownloadedTrack
	for rows.Next() {
		var t DownloadedTrack
		if err := rows.Scan(&t.InstanceID, &t.TrackID, &t.Path, &t.ContentType,
			&t.Size, &t.TrackTitle, &t.ArtistName, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ClearInstance removes all download entries for an instance and returns file paths.
func (s *DownloadStore) ClearInstance(instanceID string) ([]string, error) {
	items, err := s.List(instanceID)
	if err != nil {
		return nil, err
	}
	if _, err := s.db.exec(`DELETE FROM downloaded_tracks WHERE instance_id = ?`, instanceID); err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(items))
	for _, item := range items {
		if item.Path != "" {
			paths = append(paths, item.Path)
		}
	}
	return paths, nil
}

// EvictUntil deletes oldest cached tracks until total size is at or below targetBytes.
func (s *DownloadStore) EvictUntil(instanceID string, targetBytes int64) ([]string, error) {
	totalSize, _, err := s.Stats(instanceID)
	if err != nil {
		return nil, err
	}
	if totalSize <= targetBytes {
		return nil, nil
	}

	rows, err := s.db.query(
		`SELECT track_id, path, size FROM downloaded_tracks
		 WHERE instance_id = ? ORDER BY created_at ASC`,
		instanceID,
	)
	if err != nil {
		return nil, err
	}

	var trackIDs []string
	var paths []string
	for rows.Next() {
		if totalSize <= targetBytes {
			break
		}
		var trackID, path string
		var size int64
		if err := rows.Scan(&trackID, &path, &size); err != nil {
			_ = rows.Close()
			return nil, err
		}
		trackIDs = append(trackIDs, trackID)
		if path != "" {
			paths = append(paths, path)
		}
		if size <= 0 {
			size = 1
		}
		totalSize -= size
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if len(trackIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(trackIDs))
	args := make([]any, 0, len(trackIDs)+1)
	args = append(args, instanceID)
	for i, id := range trackIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := `DELETE FROM downloaded_tracks WHERE instance_id = ? AND track_id IN (` + //#nosec G202 -- placeholders are fixed "?" literals, values bound via args
		strings.Join(placeholders, ",") + `)`
	if _, err := s.db.exec(query, args...); err != nil {
		return nil, err
	}

	return paths, nil
}

// Delete removes a download entry and returns its prior stored path, if any.
func (s *DownloadStore) Delete(instanceID, trackID string) (string, error) {
	var path string
	err := s.db.queryRow(
		`SELECT path FROM downloaded_tracks WHERE instance_id = ? AND track_id = ?`,
		instanceID, trackID,
	).Scan(&path)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if _, err := s.db.exec(
		`DELETE FROM downloaded_tracks WHERE instance_id = ? AND track_id = ?`,
		instanceID, trackID,
	); err != nil {
		return "", err
	}
	return path, nil
}
