// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const localTrackSelectColumns = `id, library_id, rel_path, abs_path, file_sig, content_hash, size, mtime,
		        title, artist, album, album_artist, track_num, disc_num, duration_ms, genre, year, format,
		        media_kind, status, duplicate_of, updated_at`

const localTrackAudioFilter = `(media_kind = '' OR media_kind = 'audio')`
const localTrackVideoFilter = `media_kind = 'video'`

type LocalTrack struct {
	ID          string
	LibraryID   string
	RelPath     string
	AbsPath     string
	FileSig     string
	ContentHash string
	Size        int64
	Mtime       int64
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
	TrackNum    int
	DiscNum     int
	DurationMs  int
	Genre       string
	Year        int
	Format      string
	MediaKind   string
	Status      string
	DuplicateOf string
	UpdatedAt   time.Time
}

type UpsertLocalTrackInput struct {
	LibraryID   string
	RelPath     string
	AbsPath     string
	FileSig     string
	ContentHash string
	Size        int64
	Mtime       int64
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
	TrackNum    int
	DiscNum     int
	DurationMs  int
	Genre       string
	Year        int
	Format      string
	MediaKind   string
}

type MetadataSearchQuery struct {
	LibraryID string
	Query     string
	Issue     string
	Limit     int
	Offset    int
}

type LocalTrackStore struct {
	db *DB
}

func NewLocalTrackStore(db *DB) *LocalTrackStore {
	return &LocalTrackStore{db: db}
}

func newLocalTrackID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("trk-%d", time.Now().UnixNano())
	}
	return "trk_" + hex.EncodeToString(buf)
}

func (s *LocalTrackStore) GetByRelPath(libraryID, relPath string) (LocalTrack, error) {
	row := s.db.queryRow(
		`SELECT `+localTrackSelectColumns+`
		 FROM local_tracks WHERE library_id = ? AND rel_path = ?`,
		libraryID, relPath,
	)
	return scanLocalTrack(row)
}

func (s *LocalTrackStore) Upsert(input UpsertLocalTrackInput) (LocalTrack, error) {
	existing, err := s.GetByRelPath(input.LibraryID, input.RelPath)
	now := nowUnix()
	mediaKind := NormalizeMediaKind(input.MediaKind)
	if err != nil {
		if err != sql.ErrNoRows {
			return LocalTrack{}, err
		}
		id := newLocalTrackID()
		_, err = s.db.exec(
			`INSERT INTO local_tracks (
				id, library_id, rel_path, abs_path, file_sig, content_hash, size, mtime,
				title, artist, album, album_artist, track_num, disc_num, duration_ms, genre, year, format,
				media_kind, status, duplicate_of, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, input.LibraryID, input.RelPath, input.AbsPath, input.FileSig, input.ContentHash,
			input.Size, input.Mtime, input.Title, input.Artist, input.Album, input.AlbumArtist,
			input.TrackNum, input.DiscNum, input.DurationMs, input.Genre, input.Year, input.Format,
			mediaKind, TrackStatusPresent, "", now,
		)
		if err != nil {
			return LocalTrack{}, err
		}
		return s.GetByRelPath(input.LibraryID, input.RelPath)
	}

	_, err = s.db.exec(
		`UPDATE local_tracks SET abs_path = ?, file_sig = ?, content_hash = ?, size = ?, mtime = ?,
		 title = ?, artist = ?, album = ?, album_artist = ?, track_num = ?, disc_num = ?,
		 duration_ms = ?, genre = ?, year = ?, format = ?, media_kind = ?, status = ?, duplicate_of = '', updated_at = ?
		 WHERE id = ?`,
		input.AbsPath, input.FileSig, input.ContentHash, input.Size, input.Mtime,
		input.Title, input.Artist, input.Album, input.AlbumArtist, input.TrackNum, input.DiscNum,
		input.DurationMs, input.Genre, input.Year, input.Format, mediaKind, TrackStatusPresent, now, existing.ID,
	)
	if err != nil {
		return LocalTrack{}, err
	}
	return s.GetByRelPath(input.LibraryID, input.RelPath)
}

func (s *LocalTrackStore) MarkMissingExcept(libraryID string, seen map[string]struct{}) (int, error) {
	rows, err := s.db.query(
		`SELECT rel_path FROM local_tracks WHERE library_id = ? AND status = ?`,
		libraryID, TrackStatusPresent,
	)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	var toMark []string
	for rows.Next() {
		var relPath string
		if err := rows.Scan(&relPath); err != nil {
			return 0, err
		}
		if _, ok := seen[relPath]; !ok {
			toMark = append(toMark, relPath)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(toMark) == 0 {
		return 0, nil
	}

	now := nowUnix()
	for _, relPath := range toMark {
		_, err := s.db.exec(
			`UPDATE local_tracks SET status = ?, duplicate_of = '', updated_at = ?
			 WHERE library_id = ? AND rel_path = ?`,
			TrackStatusMissing, now, libraryID, relPath,
		)
		if err != nil {
			return 0, err
		}
	}
	return len(toMark), nil
}

func (s *LocalTrackStore) MarkMissingByRelPath(libraryID, relPath string) (bool, error) {
	res, err := s.db.exec(
		`UPDATE local_tracks SET status = ?, duplicate_of = '', updated_at = ?
		 WHERE library_id = ? AND rel_path = ? AND status = ?`,
		TrackStatusMissing, nowUnix(), libraryID, relPath, TrackStatusPresent,
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *LocalTrackStore) MarkMissingByPrefix(libraryID, relPrefix string) (int, error) {
	prefix := strings.TrimSuffix(relPrefix, "/") + "/%"
	res, err := s.db.exec(
		`UPDATE local_tracks SET status = ?, duplicate_of = '', updated_at = ?
		 WHERE library_id = ? AND rel_path LIKE ? AND status = ?`,
		TrackStatusMissing, nowUnix(), libraryID, prefix, TrackStatusPresent,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (s *LocalTrackStore) FindMissingByContentHash(libraryID, contentHash string) (LocalTrack, error) {
	row := s.db.queryRow(
		`SELECT `+localTrackSelectColumns+`
		 FROM local_tracks
		 WHERE library_id = ? AND content_hash = ? AND status = ?
		 ORDER BY updated_at DESC LIMIT 1`,
		libraryID, contentHash, TrackStatusMissing,
	)
	return scanLocalTrack(row)
}

// ReassignTrackPath moves an existing row to a new location on disk. Used when
// the watcher detects that a file was renamed or moved within the library.
func (s *LocalTrackStore) ReassignTrackPath(trackID, relPath, absPath, fileSig string, size, mtime int64) error {
	_, err := s.db.exec(
		`UPDATE local_tracks SET rel_path = ?, abs_path = ?, file_sig = ?, size = ?, mtime = ?,
		 status = ?, updated_at = ?
		 WHERE id = ?`,
		relPath, absPath, fileSig, size, mtime, TrackStatusPresent, nowUnix(), trackID,
	)
	return err
}

func (s *LocalTrackStore) ListByGenre(libraryID, genre string, limit, offset int) ([]LocalTrack, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.query(
		`SELECT `+localTrackSelectColumns+`
		 FROM local_tracks
		 WHERE library_id = ? AND status = ? AND duplicate_of = ''
		   AND `+localTrackAudioFilter+`
		   AND LOWER(genre) = LOWER(?)
		 ORDER BY artist ASC, album ASC, track_num ASC, title ASC
		 LIMIT ? OFFSET ?`,
		libraryID, TrackStatusPresent, genre, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanLocalTracks(rows)
}

// ListSimilarTracks returns tracks sharing the same genre first, falling back
// to the same artist when the track has no genre tag.
func (s *LocalTrackStore) ListSimilarTracks(libraryID, genre, artist, excludeID string, limit int) ([]LocalTrack, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var rows *sql.Rows
	var err error
	if genre != "" {
		rows, err = s.db.query(
			`SELECT `+localTrackSelectColumns+`
			 FROM local_tracks
			 WHERE library_id = ? AND status = ? AND duplicate_of = '' AND id != ?
			   AND `+localTrackAudioFilter+`
			   AND LOWER(genre) = LOWER(?)
			 ORDER BY RANDOM() LIMIT ?`,
			libraryID, TrackStatusPresent, excludeID, genre, limit,
		)
	} else {
		rows, err = s.db.query(
			`SELECT `+localTrackSelectColumns+`
			 FROM local_tracks
			 WHERE library_id = ? AND status = ? AND duplicate_of = '' AND id != ?
			   AND `+localTrackAudioFilter+`
			   AND LOWER(artist) = LOWER(?) AND artist != ''
			 ORDER BY RANDOM() LIMIT ?`,
			libraryID, TrackStatusPresent, excludeID, artist, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanLocalTracks(rows)
}

func (s *LocalTrackStore) ReconcileDuplicates(libraryID string) (int, error) {
	rows, err := s.db.query(
		`SELECT id, content_hash FROM local_tracks
		 WHERE library_id = ? AND status = ? AND content_hash != ''
		 ORDER BY updated_at ASC`,
		libraryID, TrackStatusPresent,
	)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	type entry struct {
		id   string
		hash string
	}
	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.id, &e.hash); err != nil {
			return 0, err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	seen := make(map[string]string)
	dupes := 0
	now := nowUnix()
	for _, e := range entries {
		canonical, ok := seen[e.hash]
		if !ok {
			seen[e.hash] = e.id
			_, err := s.db.exec(
				`UPDATE local_tracks SET duplicate_of = '', updated_at = ? WHERE id = ?`,
				now, e.id,
			)
			if err != nil {
				return 0, err
			}
			continue
		}
		dupes++
		_, err := s.db.exec(
			`UPDATE local_tracks SET duplicate_of = ?, updated_at = ? WHERE id = ?`,
			canonical, now, e.id,
		)
		if err != nil {
			return 0, err
		}
	}
	return dupes, nil
}

func (s *LocalTrackStore) CountByStatus(libraryID string) (present, missing, duplicates int, err error) {
	err = s.db.queryRow(
		`SELECT
		 SUM(CASE WHEN status = ? AND duplicate_of = '' AND `+localTrackAudioFilter+` THEN 1 ELSE 0 END),
		 SUM(CASE WHEN status = ? THEN 1 ELSE 0 END),
		 SUM(CASE WHEN status = ? AND duplicate_of != '' AND `+localTrackAudioFilter+` THEN 1 ELSE 0 END)
		 FROM local_tracks WHERE library_id = ?`,
		TrackStatusPresent, TrackStatusMissing, TrackStatusPresent, libraryID,
	).Scan(&present, &missing, &duplicates)
	return
}

func (s *LocalTrackStore) CountDistinctArtistsAlbums(libraryID string) (artists, albums int, err error) {
	err = s.db.queryRow(
		`SELECT COUNT(DISTINCT artist), COUNT(DISTINCT album)
		 FROM local_tracks WHERE library_id = ? AND status = ? AND duplicate_of = ''
		   AND `+localTrackAudioFilter,
		libraryID, TrackStatusPresent,
	).Scan(&artists, &albums)
	return
}

func (s *LocalTrackStore) ListPresent(libraryID string) ([]LocalTrack, error) {
	rows, err := s.db.query(
		`SELECT `+localTrackSelectColumns+`
		 FROM local_tracks WHERE library_id = ? AND status = ? AND duplicate_of = ''
		   AND `+localTrackAudioFilter+`
		 ORDER BY artist ASC, album ASC, track_num ASC, title ASC`,
		libraryID, TrackStatusPresent,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanLocalTracks(rows)
}

func (s *LocalTrackStore) ListPresentInLibraries(libraryIDs []string) ([]LocalTrack, error) {
	if len(libraryIDs) == 0 {
		return nil, nil
	}
	if len(libraryIDs) == 1 {
		return s.ListPresent(libraryIDs[0])
	}
	placeholders := strings.Repeat("?,", len(libraryIDs))
	placeholders = placeholders[:len(placeholders)-1]
	query := `SELECT ` + localTrackSelectColumns + `
		 FROM local_tracks WHERE library_id IN (` + placeholders + `)
		   AND status = ? AND duplicate_of = ''
		   AND ` + localTrackAudioFilter + `
		 ORDER BY artist ASC, album ASC, track_num ASC, title ASC`
	args := make([]any, 0, len(libraryIDs)+1)
	for _, id := range libraryIDs {
		args = append(args, id)
	}
	args = append(args, TrackStatusPresent)
	rows, err := s.db.query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanLocalTracks(rows)
}

func (s *LocalTrackStore) SearchMetadata(q MetadataSearchQuery) ([]LocalTrack, int, error) {
	if q.Limit <= 0 {
		q.Limit = 50
	}
	if q.Limit > 200 {
		q.Limit = 200
	}
	if q.Offset < 0 {
		q.Offset = 0
	}

	where := []string{"library_id = ?", "status = ?", "duplicate_of = ''", localTrackAudioFilter}
	args := []any{q.LibraryID, TrackStatusPresent}

	needle := strings.TrimSpace(strings.ToLower(q.Query))
	if needle != "" {
		like := "%" + needle + "%"
		where = append(where, `(LOWER(title) LIKE ? OR LOWER(artist) LIKE ? OR LOWER(album) LIKE ?
		 OR LOWER(rel_path) LIKE ? OR LOWER(album_artist) LIKE ?)`)
		args = append(args, like, like, like, like, like)
	}

	switch q.Issue {
	case "unknown-artist":
		where = append(where, `(artist = '' OR LOWER(artist) IN ('unknown', 'unknown artist', 'unknownartist'))`)
	case "unknown-album":
		where = append(where, `(album = '' OR LOWER(album) IN ('unknown', 'unknown album', 'unknownalbum'))`)
	case "missing-title":
		where = append(where, `title = ''`)
	case "any":
		where = append(where, `(artist = '' OR LOWER(artist) IN ('unknown', 'unknown artist', 'unknownartist')
		 OR album = '' OR LOWER(album) IN ('unknown', 'unknown album', 'unknownalbum')
		 OR title = '')`)
	}

	whereSQL := strings.Join(where, " AND ")
	countQuery := `SELECT COUNT(*) FROM local_tracks WHERE ` + whereSQL
	var total int
	if err := s.db.queryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := append(append([]any{}, args...), q.Limit, q.Offset)
	rows, err := s.db.query(
		`SELECT `+localTrackSelectColumns+`
		 FROM local_tracks WHERE `+whereSQL+`
		 ORDER BY artist ASC, album ASC, track_num ASC, title ASC
		 LIMIT ? OFFSET ?`,
		listArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	tracks, err := scanLocalTracks(rows)
	return tracks, total, err
}

func (s *LocalTrackStore) ListByAlbumArtist(libraryID, album, artist string) ([]LocalTrack, error) {
	rows, err := s.db.query(
		`SELECT `+localTrackSelectColumns+`
		 FROM local_tracks
		 WHERE library_id = ? AND status = ? AND duplicate_of = ''
		   AND `+localTrackAudioFilter+`
		   AND LOWER(album) = LOWER(?) AND LOWER(artist) = LOWER(?)
		 ORDER BY track_num ASC, title ASC`,
		libraryID, TrackStatusPresent, album, artist,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanLocalTracks(rows)
}

func (s *LocalTrackStore) CountMetadataIssues(libraryID string) (unknownArtist, unknownAlbum, missingTitle, any int, err error) {
	err = s.db.queryRow(
		`SELECT
		 SUM(CASE WHEN artist = '' OR LOWER(artist) IN ('unknown', 'unknown artist', 'unknownartist') THEN 1 ELSE 0 END),
		 SUM(CASE WHEN album = '' OR LOWER(album) IN ('unknown', 'unknown album', 'unknownalbum') THEN 1 ELSE 0 END),
		 SUM(CASE WHEN title = '' THEN 1 ELSE 0 END),
		 SUM(CASE WHEN artist = '' OR LOWER(artist) IN ('unknown', 'unknown artist', 'unknownartist')
		   OR album = '' OR LOWER(album) IN ('unknown', 'unknown album', 'unknownalbum')
		   OR title = '' THEN 1 ELSE 0 END)
		 FROM local_tracks WHERE library_id = ? AND status = ? AND duplicate_of = ''
		   AND `+localTrackAudioFilter,
		libraryID, TrackStatusPresent,
	).Scan(&unknownArtist, &unknownAlbum, &missingTitle, &any)
	return
}

func (s *LocalTrackStore) Get(libraryID, trackID string) (LocalTrack, error) {
	row := s.db.queryRow(
		`SELECT `+localTrackSelectColumns+`
		 FROM local_tracks WHERE library_id = ? AND id = ?`,
		libraryID, trackID,
	)
	return scanLocalTrack(row)
}

func (s *LocalTrackStore) GetByID(trackID string) (LocalTrack, error) {
	row := s.db.queryRow(
		`SELECT `+localTrackSelectColumns+`
		 FROM local_tracks WHERE id = ?`,
		trackID,
	)
	return scanLocalTrack(row)
}

func fallbackTitle(path string) string {
	base := path
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	if i := strings.LastIndex(base, "."); i > 0 {
		base = base[:i]
	}
	return strings.TrimSpace(base)
}

func scanLocalTrack(row *sql.Row) (LocalTrack, error) {
	var track LocalTrack
	var updated int64
	err := row.Scan(
		&track.ID, &track.LibraryID, &track.RelPath, &track.AbsPath, &track.FileSig, &track.ContentHash,
		&track.Size, &track.Mtime, &track.Title, &track.Artist, &track.Album, &track.AlbumArtist,
		&track.TrackNum, &track.DiscNum, &track.DurationMs, &track.Genre, &track.Year, &track.Format,
		&track.MediaKind, &track.Status, &track.DuplicateOf, &updated,
	)
	if err != nil {
		return LocalTrack{}, err
	}
	track.MediaKind = NormalizeMediaKind(track.MediaKind)
	track.UpdatedAt = time.Unix(updated, 0)
	if track.Title == "" {
		track.Title = fallbackTitle(track.RelPath)
	}
	return track, nil
}

func scanLocalTracks(rows *sql.Rows) ([]LocalTrack, error) {
	var items []LocalTrack
	for rows.Next() {
		var track LocalTrack
		var updated int64
		if err := rows.Scan(
			&track.ID, &track.LibraryID, &track.RelPath, &track.AbsPath, &track.FileSig, &track.ContentHash,
			&track.Size, &track.Mtime, &track.Title, &track.Artist, &track.Album, &track.AlbumArtist,
			&track.TrackNum, &track.DiscNum, &track.DurationMs, &track.Genre, &track.Year, &track.Format,
			&track.MediaKind, &track.Status, &track.DuplicateOf, &updated,
		); err != nil {
			return nil, err
		}
		track.MediaKind = NormalizeMediaKind(track.MediaKind)
		track.UpdatedAt = time.Unix(updated, 0)
		if track.Title == "" {
			track.Title = fallbackTitle(track.RelPath)
		}
		items = append(items, track)
	}
	return items, rows.Err()
}

type GenreCount struct {
	Name  string
	Count int
}

func (s *LocalTrackStore) ListGenreCounts(libraryIDs []string) ([]GenreCount, error) {
	if len(libraryIDs) == 0 {
		return nil, nil
	}
	placeholders := strings.Repeat("?,", len(libraryIDs))
	placeholders = placeholders[:len(placeholders)-1]
	query := `SELECT genre, COUNT(*) FROM local_tracks
		WHERE library_id IN (` + placeholders + `)
		  AND status = ? AND duplicate_of = '' AND genre != ''
		  AND ` + localTrackAudioFilter + `
		GROUP BY genre ORDER BY genre ASC`
	args := make([]any, 0, len(libraryIDs)+1)
	for _, id := range libraryIDs {
		args = append(args, id)
	}
	args = append(args, TrackStatusPresent)
	rows, err := s.db.query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var items []GenreCount
	for rows.Next() {
		var item GenreCount
		if err := rows.Scan(&item.Name, &item.Count); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *LocalTrackStore) ListVideos(libraryID string) ([]LocalTrack, error) {
	rows, err := s.db.query(
		`SELECT `+localTrackSelectColumns+`
		 FROM local_tracks WHERE library_id = ? AND status = ? AND duplicate_of = ''
		   AND `+localTrackVideoFilter+`
		 ORDER BY artist ASC, title ASC`,
		libraryID, TrackStatusPresent,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanLocalTracks(rows)
}

func (s *LocalTrackStore) ListVideosInLibraries(libraryIDs []string) ([]LocalTrack, error) {
	if len(libraryIDs) == 0 {
		return nil, nil
	}
	if len(libraryIDs) == 1 {
		return s.ListVideos(libraryIDs[0])
	}
	placeholders := strings.Repeat("?,", len(libraryIDs))
	placeholders = placeholders[:len(placeholders)-1]
	query := `SELECT ` + localTrackSelectColumns + `
		 FROM local_tracks WHERE library_id IN (` + placeholders + `)
		   AND status = ? AND duplicate_of = ''
		   AND ` + localTrackVideoFilter + `
		 ORDER BY artist ASC, title ASC`
	args := make([]any, 0, len(libraryIDs)+1)
	for _, id := range libraryIDs {
		args = append(args, id)
	}
	args = append(args, TrackStatusPresent)
	rows, err := s.db.query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanLocalTracks(rows)
}
