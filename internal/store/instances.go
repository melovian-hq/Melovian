// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"melovian/internal/appconfig"
)

type SubsonicInstance struct {
	ID         string
	UserID     string
	Name       string
	ServerURL  string
	Username   string
	Password   string
	ServerName string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LastUsedAt *time.Time
}

const prefActiveInstanceID = "active_instance_id"

type CreateInstanceInput struct {
	Name       string
	ServerURL  string
	Username   string
	Password   string
	ServerName string
}

type UpdateInstanceInput struct {
	Name       string
	ServerURL  string
	Username   string
	Password   string
	ServerName string
}

type InstanceStore struct {
	db *DB
}

func NewInstanceStore(db *DB) *InstanceStore {
	return &InstanceStore{db: db}
}

func newInstanceID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("inst-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func (s *InstanceStore) ListForUser(userID string) ([]SubsonicInstance, error) {
	if userID == "" {
		return s.List()
	}
	rows, err := s.db.query(
		`SELECT id, user_id, name, server_url, username, password, server_name, created_at, updated_at, last_used_at
		 FROM subsonic_instances WHERE user_id = ? ORDER BY name ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanInstances(rows)
}

func (s *InstanceStore) List() ([]SubsonicInstance, error) {
	rows, err := s.db.query(
		`SELECT id, user_id, name, server_url, username, password, server_name, created_at, updated_at, last_used_at
		 FROM subsonic_instances ORDER BY name ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanInstances(rows)
}

func (s *InstanceStore) GetForUser(userID, id string) (SubsonicInstance, error) {
	inst, err := s.Get(id)
	if err != nil {
		return SubsonicInstance{}, err
	}
	if userID != "" && inst.UserID != userID {
		return SubsonicInstance{}, sql.ErrNoRows
	}
	return inst, nil
}

func (s *InstanceStore) Get(id string) (SubsonicInstance, error) {
	row := s.db.queryRow(
		`SELECT id, user_id, name, server_url, username, password, server_name, created_at, updated_at, last_used_at
		 FROM subsonic_instances WHERE id = ?`,
		id,
	)
	return scanInstance(row)
}

func (s *InstanceStore) CreateForUser(userID string, input CreateInstanceInput) (SubsonicInstance, error) {
	inst, err := s.create(input)
	if err != nil {
		return SubsonicInstance{}, err
	}
	if userID != "" {
		_, err = s.db.exec(`UPDATE subsonic_instances SET user_id = ? WHERE id = ?`, userID, inst.ID)
		if err != nil {
			return SubsonicInstance{}, err
		}
		inst.UserID = userID
	}
	return inst, nil
}

func (s *InstanceStore) Create(input CreateInstanceInput) (SubsonicInstance, error) {
	return s.create(input)
}

func (s *InstanceStore) create(input CreateInstanceInput) (SubsonicInstance, error) {
	name := strings.TrimSpace(input.Name)
	serverURL := strings.TrimRight(strings.TrimSpace(input.ServerURL), "/")
	username := strings.TrimSpace(input.Username)
	password := input.Password

	if name == "" {
		return SubsonicInstance{}, fmt.Errorf("name is required")
	}
	if serverURL == "" {
		return SubsonicInstance{}, fmt.Errorf("server url is required")
	}
	if username == "" {
		return SubsonicInstance{}, fmt.Errorf("username is required")
	}
	if password == "" {
		return SubsonicInstance{}, fmt.Errorf("password is required")
	}

	id := newInstanceID()
	now := nowUnix()
	_, err := s.db.exec(
		`INSERT INTO subsonic_instances (id, name, server_url, username, password, server_name, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, name, serverURL, username, password, strings.TrimSpace(input.ServerName), now, now,
	)
	if err != nil {
		return SubsonicInstance{}, err
	}
	return s.Get(id)
}

func (s *InstanceStore) Update(id string, input UpdateInstanceInput) (SubsonicInstance, error) {
	existing, err := s.Get(id)
	if err != nil {
		return SubsonicInstance{}, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = existing.Name
	}
	serverURL := strings.TrimRight(strings.TrimSpace(input.ServerURL), "/")
	if serverURL == "" {
		serverURL = existing.ServerURL
	}
	username := strings.TrimSpace(input.Username)
	if username == "" {
		username = existing.Username
	}
	password := input.Password
	if password == "" {
		password = existing.Password
	}
	serverName := strings.TrimSpace(input.ServerName)
	if serverName == "" && input.ServerName == "" {
		serverName = existing.ServerName
	}

	now := nowUnix()
	_, err = s.db.exec(
		`UPDATE subsonic_instances SET name = ?, server_url = ?, username = ?, password = ?,
		 server_name = ?, updated_at = ? WHERE id = ?`,
		name, serverURL, username, password, serverName, now, id,
	)
	if err != nil {
		return SubsonicInstance{}, err
	}
	return s.Get(id)
}

func (s *InstanceStore) Delete(id string) error {
	res, err := s.db.exec(`DELETE FROM subsonic_instances WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}

	activeID, err := s.GetActiveID()
	if err != nil || activeID != id {
		return nil
	}

	_ = s.db.setSetting(appconfig.SettingActiveInstanceID, "")
	return nil
}

func (s *InstanceStore) GetActiveIDForUser(userID string) (string, error) {
	if userID == "" {
		return s.GetActiveID()
	}
	prefs := NewPreferencesStore(s.db)
	value, err := prefs.Get(userID, prefActiveInstanceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

func (s *InstanceStore) GetActiveForUser(userID string) (SubsonicInstance, error) {
	id, err := s.GetActiveIDForUser(userID)
	if err != nil {
		return SubsonicInstance{}, err
	}
	if id == "" {
		return SubsonicInstance{}, sql.ErrNoRows
	}
	return s.GetForUser(userID, id)
}

func (s *InstanceStore) ClearActiveForUser(userID string) error {
	if userID == "" {
		return s.db.setSetting(appconfig.SettingActiveInstanceID, "")
	}
	prefs := NewPreferencesStore(s.db)
	err := prefs.Delete(userID, prefActiveInstanceID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return err
}

func (s *InstanceStore) SetActiveForUser(userID, id string) error {
	if userID == "" {
		return s.SetActive(id)
	}
	if _, err := s.GetForUser(userID, id); err != nil {
		return err
	}
	prefs := NewPreferencesStore(s.db)
	return prefs.Set(userID, prefActiveInstanceID, id)
}

func (s *InstanceStore) GetActiveID() (string, error) {
	value, err := s.db.getSetting(appconfig.SettingActiveInstanceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

func (s *InstanceStore) GetActive() (SubsonicInstance, error) {
	id, err := s.GetActiveID()
	if err != nil {
		return SubsonicInstance{}, err
	}
	if id == "" {
		return SubsonicInstance{}, sql.ErrNoRows
	}
	return s.Get(id)
}

func (s *InstanceStore) SetActive(id string) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	return s.db.setSetting(appconfig.SettingActiveInstanceID, id)
}

func (s *InstanceStore) TouchLastUsed(id string) error {
	now := nowUnix()
	res, err := s.db.exec(`UPDATE subsonic_instances SET last_used_at = ? WHERE id = ?`, now, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *InstanceStore) PublicView(inst SubsonicInstance) map[string]any {
	payload := map[string]any{
		"id":         inst.ID,
		"name":       inst.Name,
		"serverUrl":  inst.ServerURL,
		"username":   inst.Username,
		"serverName": inst.ServerName,
		"createdAt":  inst.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":  inst.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if inst.LastUsedAt != nil {
		payload["lastUsedAt"] = inst.LastUsedAt.UTC().Format(time.RFC3339)
	}
	return payload
}

func scanInstance(row *sql.Row) (SubsonicInstance, error) {
	var inst SubsonicInstance
	var created, updated int64
	var lastUsed sql.NullInt64
	err := row.Scan(
		&inst.ID, &inst.UserID, &inst.Name, &inst.ServerURL, &inst.Username, &inst.Password,
		&inst.ServerName, &created, &updated, &lastUsed,
	)
	if err != nil {
		return SubsonicInstance{}, err
	}
	inst.CreatedAt = time.Unix(created, 0)
	inst.UpdatedAt = time.Unix(updated, 0)
	if lastUsed.Valid {
		t := time.Unix(lastUsed.Int64, 0)
		inst.LastUsedAt = &t
	}
	return inst, nil
}

func scanInstances(rows *sql.Rows) ([]SubsonicInstance, error) {
	var items []SubsonicInstance
	for rows.Next() {
		var inst SubsonicInstance
		var created, updated int64
		var lastUsed sql.NullInt64
		if err := rows.Scan(
			&inst.ID, &inst.UserID, &inst.Name, &inst.ServerURL, &inst.Username, &inst.Password,
			&inst.ServerName, &created, &updated, &lastUsed,
		); err != nil {
			return nil, err
		}
		inst.CreatedAt = time.Unix(created, 0)
		inst.UpdatedAt = time.Unix(updated, 0)
		if lastUsed.Valid {
			t := time.Unix(lastUsed.Int64, 0)
			inst.LastUsedAt = &t
		}
		items = append(items, inst)
	}
	return items, rows.Err()
}
