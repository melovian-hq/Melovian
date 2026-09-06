// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type LocalLibrary struct {
	ID             string
	UserID         string
	Name           string
	Path           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastScannedAt  *time.Time
	TrackCount     int
	MissingCount   int
	DuplicateCount int
	ScanStatus     string
	ScanError      string
}

const (
	prefActiveLocalLibraryID = "active_local_library_id"
	TrackStatusPresent       = "present"
	TrackStatusMissing       = "missing"
)

type CreateLocalLibraryInput struct {
	Name string
	Path string
}

type UpdateLocalLibraryInput struct {
	Name string
	Path string
}

type LocalLibraryStore struct {
	db *DB
}

func NewLocalLibraryStore(db *DB) *LocalLibraryStore {
	return &LocalLibraryStore{db: db}
}

func newLocalLibraryID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("lib-%d", time.Now().UnixNano())
	}
	return "lib_" + hex.EncodeToString(buf)
}

func (s *LocalLibraryStore) ListForUser(userID string) ([]LocalLibrary, error) {
	if userID == "" {
		return s.List()
	}
	rows, err := s.db.query(
		`SELECT id, user_id, name, path, created_at, updated_at, last_scanned_at,
		        track_count, missing_count, duplicate_count, scan_status, scan_error
		 FROM local_libraries WHERE user_id = ? ORDER BY name ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanLocalLibraries(rows)
}

func (s *LocalLibraryStore) List() ([]LocalLibrary, error) {
	rows, err := s.db.query(
		`SELECT id, user_id, name, path, created_at, updated_at, last_scanned_at,
		        track_count, missing_count, duplicate_count, scan_status, scan_error
		 FROM local_libraries ORDER BY name ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanLocalLibraries(rows)
}

func (s *LocalLibraryStore) GetForUser(userID, id string) (LocalLibrary, error) {
	lib, err := s.Get(id)
	if err != nil {
		return LocalLibrary{}, err
	}
	if userID != "" && lib.UserID != userID {
		return LocalLibrary{}, sql.ErrNoRows
	}
	return lib, nil
}

func (s *LocalLibraryStore) Get(id string) (LocalLibrary, error) {
	row := s.db.queryRow(
		`SELECT id, user_id, name, path, created_at, updated_at, last_scanned_at,
		        track_count, missing_count, duplicate_count, scan_status, scan_error
		 FROM local_libraries WHERE id = ?`,
		id,
	)
	return scanLocalLibrary(row)
}

func (s *LocalLibraryStore) FindByPathForUser(userID, path string) (LocalLibrary, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return LocalLibrary{}, sql.ErrNoRows
	}

	if userID == "" {
		row := s.db.queryRow(
			`SELECT id, user_id, name, path, created_at, updated_at, last_scanned_at,
			        track_count, missing_count, duplicate_count, scan_status, scan_error
			 FROM local_libraries WHERE path = ? LIMIT 1`,
			path,
		)
		return scanLocalLibrary(row)
	}

	row := s.db.queryRow(
		`SELECT id, user_id, name, path, created_at, updated_at, last_scanned_at,
		        track_count, missing_count, duplicate_count, scan_status, scan_error
		 FROM local_libraries WHERE user_id = ? AND path = ? LIMIT 1`,
		userID, path,
	)
	return scanLocalLibrary(row)
}

func (s *LocalLibraryStore) CreateForUser(userID string, input CreateLocalLibraryInput) (LocalLibrary, error) {
	lib, err := s.create(input)
	if err != nil {
		return LocalLibrary{}, err
	}
	if userID != "" {
		_, err = s.db.exec(`UPDATE local_libraries SET user_id = ? WHERE id = ?`, userID, lib.ID)
		if err != nil {
			return LocalLibrary{}, err
		}
		lib.UserID = userID
	}
	return lib, nil
}

func (s *LocalLibraryStore) Create(input CreateLocalLibraryInput) (LocalLibrary, error) {
	return s.create(input)
}

func (s *LocalLibraryStore) create(input CreateLocalLibraryInput) (LocalLibrary, error) {
	name := strings.TrimSpace(input.Name)
	path := strings.TrimSpace(input.Path)
	if name == "" {
		return LocalLibrary{}, fmt.Errorf("name is required")
	}
	if path == "" {
		return LocalLibrary{}, fmt.Errorf("path is required")
	}

	id := newLocalLibraryID()
	now := nowUnix()
	_, err := s.db.exec(
		`INSERT INTO local_libraries (id, name, path, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		id, name, path, now, now,
	)
	if err != nil {
		return LocalLibrary{}, err
	}
	return s.Get(id)
}

func (s *LocalLibraryStore) Update(id string, input UpdateLocalLibraryInput) (LocalLibrary, error) {
	existing, err := s.Get(id)
	if err != nil {
		return LocalLibrary{}, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = existing.Name
	}
	path := strings.TrimSpace(input.Path)
	if path == "" {
		path = existing.Path
	}

	now := nowUnix()
	_, err = s.db.exec(
		`UPDATE local_libraries SET name = ?, path = ?, updated_at = ? WHERE id = ?`,
		name, path, now, id,
	)
	if err != nil {
		return LocalLibrary{}, err
	}
	return s.Get(id)
}

func (s *LocalLibraryStore) Delete(id string) error {
	res, err := s.db.exec(`DELETE FROM local_libraries WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *LocalLibraryStore) SetScanState(id, status, scanError string) error {
	now := nowUnix()
	_, err := s.db.exec(
		`UPDATE local_libraries SET scan_status = ?, scan_error = ?, updated_at = ? WHERE id = ?`,
		status, scanError, now, id,
	)
	return err
}

func (s *LocalLibraryStore) UpdateScanStats(id string, trackCount, missingCount, duplicateCount int) error {
	now := nowUnix()
	_, err := s.db.exec(
		`UPDATE local_libraries SET track_count = ?, missing_count = ?, duplicate_count = ?,
		 last_scanned_at = ?, scan_status = 'idle', scan_error = '', updated_at = ?
		 WHERE id = ?`,
		trackCount, missingCount, duplicateCount, now, now, id,
	)
	return err
}

func (s *LocalLibraryStore) GetActiveIDForUser(userID string) (string, error) {
	if userID == "" {
		return s.GetActiveID()
	}
	prefs := NewPreferencesStore(s.db)
	value, err := prefs.Get(userID, prefActiveLocalLibraryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

func (s *LocalLibraryStore) GetActiveForUser(userID string) (LocalLibrary, error) {
	id, err := s.GetActiveIDForUser(userID)
	if err != nil {
		return LocalLibrary{}, err
	}
	if id == "" {
		return LocalLibrary{}, sql.ErrNoRows
	}
	return s.GetForUser(userID, id)
}

func (s *LocalLibraryStore) SetActiveForUser(userID, id string) error {
	if userID == "" {
		return s.SetActive(id)
	}
	if _, err := s.GetForUser(userID, id); err != nil {
		return err
	}
	prefs := NewPreferencesStore(s.db)
	return prefs.Set(userID, prefActiveLocalLibraryID, id)
}

func (s *LocalLibraryStore) ClearActiveForUser(userID string) error {
	if userID == "" {
		return s.db.setSetting(prefActiveLocalLibraryID, "")
	}
	prefs := NewPreferencesStore(s.db)
	err := prefs.Delete(userID, prefActiveLocalLibraryID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return err
}

func (s *LocalLibraryStore) GetActiveID() (string, error) {
	value, err := s.db.getSetting(prefActiveLocalLibraryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

func (s *LocalLibraryStore) SetActive(id string) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	return s.db.setSetting(prefActiveLocalLibraryID, id)
}

func (s *LocalLibraryStore) PublicView(lib LocalLibrary) map[string]any {
	payload := map[string]any{
		"id":             lib.ID,
		"type":           "local",
		"name":           lib.Name,
		"path":           lib.Path,
		"trackCount":     lib.TrackCount,
		"missingCount":   lib.MissingCount,
		"duplicateCount": lib.DuplicateCount,
		"scanStatus":     lib.ScanStatus,
		"createdAt":      lib.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":      lib.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if lib.LastScannedAt != nil {
		payload["lastScannedAt"] = lib.LastScannedAt.UTC().Format(time.RFC3339)
	}
	if lib.ScanError != "" {
		payload["scanError"] = lib.ScanError
	}
	return payload
}

func scanLocalLibrary(row *sql.Row) (LocalLibrary, error) {
	var lib LocalLibrary
	var created, updated int64
	var lastScanned sql.NullInt64
	err := row.Scan(
		&lib.ID, &lib.UserID, &lib.Name, &lib.Path, &created, &updated, &lastScanned,
		&lib.TrackCount, &lib.MissingCount, &lib.DuplicateCount, &lib.ScanStatus, &lib.ScanError,
	)
	if err != nil {
		return LocalLibrary{}, err
	}
	lib.CreatedAt = time.Unix(created, 0)
	lib.UpdatedAt = time.Unix(updated, 0)
	if lastScanned.Valid {
		t := time.Unix(lastScanned.Int64, 0)
		lib.LastScannedAt = &t
	}
	return lib, nil
}

func scanLocalLibraries(rows *sql.Rows) ([]LocalLibrary, error) {
	var items []LocalLibrary
	for rows.Next() {
		var lib LocalLibrary
		var created, updated int64
		var lastScanned sql.NullInt64
		if err := rows.Scan(
			&lib.ID, &lib.UserID, &lib.Name, &lib.Path, &created, &updated, &lastScanned,
			&lib.TrackCount, &lib.MissingCount, &lib.DuplicateCount, &lib.ScanStatus, &lib.ScanError,
		); err != nil {
			return nil, err
		}
		lib.CreatedAt = time.Unix(created, 0)
		lib.UpdatedAt = time.Unix(updated, 0)
		if lastScanned.Valid {
			t := time.Unix(lastScanned.Int64, 0)
			lib.LastScannedAt = &t
		}
		items = append(items, lib)
	}
	return items, rows.Err()
}
