// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

const (
	ShareAccessPublic     = "public"
	ShareAccessPassword   = "password"
	ShareAccessRestricted = "restricted"
)

type Share struct {
	ID           string
	UserID       string
	Token        string
	ResourceType string
	ResourceID   string
	Description  string
	AccessMode   string
	PasswordHash string
	InstanceID   string
	OwnerScope   string
	ExpiresAt    *time.Time
	VisitCount   int
	CreatedAt    time.Time
	RecipientIDs []string
}

type ShareStore struct {
	db *DB
}

func NewShareStore(db *DB) *ShareStore {
	return &ShareStore{db: db}
}

func (s *ShareStore) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS music_shares (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	token TEXT NOT NULL UNIQUE,
	resource_type TEXT NOT NULL,
	resource_id TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	expires_at INTEGER,
	visit_count INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_music_shares_user ON music_shares(user_id, created_at DESC);
CREATE TABLE IF NOT EXISTS music_share_recipients (
	share_id TEXT NOT NULL REFERENCES music_shares(id) ON DELETE CASCADE,
	user_id TEXT NOT NULL,
	PRIMARY KEY (share_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_music_share_recipients_user ON music_share_recipients(user_id);
`
	if err := s.db.execScript(schema); err != nil {
		return err
	}
	for _, stmt := range []string{
		`ALTER TABLE music_shares ADD COLUMN access_mode TEXT NOT NULL DEFAULT 'public'`,
		`ALTER TABLE music_shares ADD COLUMN password_hash TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE music_shares ADD COLUMN instance_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE music_shares ADD COLUMN owner_scope TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err := s.db.exec(stmt); err != nil && !s.db.dialect.IsDuplicateColumn(err) {
			return err
		}
	}
	return nil
}

type CreateShareInput struct {
	UserID       string
	ResourceType string
	ResourceID   string
	Description  string
	ExpiresAt    *time.Time
	AccessMode   string
	Password     string
	InstanceID   string
	OwnerScope   string
	RecipientIDs []string
}

func (s *ShareStore) Create(input CreateShareInput) (Share, error) {
	resourceType := strings.TrimSpace(input.ResourceType)
	resourceID := strings.TrimSpace(input.ResourceID)
	if resourceType == "" || resourceID == "" {
		return Share{}, errors.New("resource type and id are required")
	}
	accessMode := normalizeShareAccessMode(input.AccessMode)
	passwordHash := ""
	if accessMode == ShareAccessPassword {
		pw := strings.TrimSpace(input.Password)
		if pw == "" {
			return Share{}, errors.New("password is required for password shares")
		}
		hash, err := HashPassword(pw)
		if err != nil {
			return Share{}, err
		}
		passwordHash = hash
	}
	if accessMode == ShareAccessRestricted && len(input.RecipientIDs) == 0 {
		return Share{}, errors.New("at least one recipient is required for restricted shares")
	}

	id := newShareID()
	token := newShareToken()
	now := nowUnix()
	var expires *int64
	if input.ExpiresAt != nil {
		v := input.ExpiresAt.Unix()
		expires = &v
	}

	tx, err := s.db.begin()
	if err != nil {
		return Share{}, err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec(
		`INSERT INTO music_shares (
			id, user_id, token, resource_type, resource_id, description, expires_at, visit_count, created_at,
			access_mode, password_hash, instance_id, owner_scope
		) VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?)`,
		id, input.UserID, token, resourceType, resourceID, strings.TrimSpace(input.Description), expires, now,
		accessMode, passwordHash, strings.TrimSpace(input.InstanceID), strings.TrimSpace(input.OwnerScope),
	)
	if err != nil {
		return Share{}, err
	}

	seen := make(map[string]struct{}, len(input.RecipientIDs))
	for _, recipientID := range input.RecipientIDs {
		recipientID = strings.TrimSpace(recipientID)
		if recipientID == "" {
			continue
		}
		if _, ok := seen[recipientID]; ok {
			continue
		}
		seen[recipientID] = struct{}{}
		if _, err := tx.Exec(
			`INSERT INTO music_share_recipients (share_id, user_id) VALUES (?, ?)`,
			id, recipientID,
		); err != nil {
			return Share{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Share{}, err
	}
	return s.GetByID(input.UserID, id)
}

func (s *ShareStore) ListForUser(userID string) ([]Share, error) {
	rows, err := s.db.query(
		`SELECT id, user_id, token, resource_type, resource_id, description, expires_at, visit_count, created_at,
		        access_mode, password_hash, instance_id, owner_scope
		 FROM music_shares WHERE user_id = ? ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items, err := scanShares(rows)
	if err != nil {
		return nil, err
	}
	for i := range items {
		recipients, err := s.listRecipientIDs(items[i].ID)
		if err != nil {
			return nil, err
		}
		items[i].RecipientIDs = recipients
	}
	return items, nil
}

func (s *ShareStore) ListInbox(recipientUserID string) ([]Share, error) {
	rows, err := s.db.query(
		`SELECT s.id, s.user_id, s.token, s.resource_type, s.resource_id, s.description, s.expires_at, s.visit_count, s.created_at,
		        s.access_mode, s.password_hash, s.instance_id, s.owner_scope
		 FROM music_shares s
		 INNER JOIN music_share_recipients r ON r.share_id = s.id
		 WHERE r.user_id = ? AND s.access_mode = ?
		 ORDER BY s.created_at DESC`,
		recipientUserID, ShareAccessRestricted,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items, err := scanShares(rows)
	if err != nil {
		return nil, err
	}
	for i := range items {
		recipients, err := s.listRecipientIDs(items[i].ID)
		if err != nil {
			return nil, err
		}
		items[i].RecipientIDs = recipients
	}
	return items, nil
}

func (s *ShareStore) GetByToken(token string) (Share, error) {
	row := s.db.queryRow(
		`SELECT id, user_id, token, resource_type, resource_id, description, expires_at, visit_count, created_at,
		        access_mode, password_hash, instance_id, owner_scope
		 FROM music_shares WHERE token = ?`,
		token,
	)
	share, err := scanShare(row)
	if err != nil {
		return Share{}, err
	}
	recipients, err := s.listRecipientIDs(share.ID)
	if err != nil {
		return Share{}, err
	}
	share.RecipientIDs = recipients
	return share, nil
}

func (s *ShareStore) GetByID(userID, id string) (Share, error) {
	row := s.db.queryRow(
		`SELECT id, user_id, token, resource_type, resource_id, description, expires_at, visit_count, created_at,
		        access_mode, password_hash, instance_id, owner_scope
		 FROM music_shares WHERE id = ? AND user_id = ?`,
		id, userID,
	)
	share, err := scanShare(row)
	if err != nil {
		return Share{}, err
	}
	recipients, err := s.listRecipientIDs(share.ID)
	if err != nil {
		return Share{}, err
	}
	share.RecipientIDs = recipients
	return share, nil
}

func (s *ShareStore) Delete(userID, id string) error {
	res, err := s.db.exec(`DELETE FROM music_shares WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *ShareStore) TouchVisit(token string) error {
	_, err := s.db.exec(`UPDATE music_shares SET visit_count = visit_count + 1 WHERE token = ?`, token)
	return err
}

func (s *ShareStore) CheckPassword(share Share, password string) bool {
	if share.AccessMode != ShareAccessPassword || share.PasswordHash == "" {
		return false
	}
	ok, err := VerifyPassword(share.PasswordHash, password)
	if err != nil || !ok {
		return false
	}
	if NeedsRehash(share.PasswordHash) {
		if newHash, hashErr := HashPassword(password); hashErr == nil {
			_, _ = s.db.exec(
				`UPDATE music_shares SET password_hash = ? WHERE id = ?`,
				newHash, share.ID,
			)
		}
	}
	return true
}

func (s *ShareStore) IsRecipient(share Share, userID string) bool {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false
	}
	return slices.Contains(share.RecipientIDs, userID)
}

func (s *Share) Expired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*s.ExpiresAt)
}

func (s *ShareStore) listRecipientIDs(shareID string) ([]string, error) {
	rows, err := s.db.query(
		`SELECT user_id FROM music_share_recipients WHERE share_id = ? ORDER BY user_id ASC`,
		shareID,
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

func normalizeShareAccessMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case ShareAccessPassword:
		return ShareAccessPassword
	case ShareAccessRestricted:
		return ShareAccessRestricted
	default:
		return ShareAccessPublic
	}
}

func newShareID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("share-%d", time.Now().UnixNano())
	}
	return "shr_" + hex.EncodeToString(buf)
}

func newShareToken() string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("tok-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func scanShare(row *sql.Row) (Share, error) {
	var share Share
	var created int64
	var expires sql.NullInt64
	err := row.Scan(
		&share.ID, &share.UserID, &share.Token, &share.ResourceType, &share.ResourceID,
		&share.Description, &expires, &share.VisitCount, &created,
		&share.AccessMode, &share.PasswordHash, &share.InstanceID, &share.OwnerScope,
	)
	if err != nil {
		return Share{}, err
	}
	if expires.Valid {
		t := time.Unix(expires.Int64, 0)
		share.ExpiresAt = &t
	}
	share.CreatedAt = time.Unix(created, 0)
	if share.AccessMode == "" {
		share.AccessMode = ShareAccessPublic
	}
	return share, nil
}

func scanShares(rows *sql.Rows) ([]Share, error) {
	var items []Share
	for rows.Next() {
		var share Share
		var created int64
		var expires sql.NullInt64
		if err := rows.Scan(
			&share.ID, &share.UserID, &share.Token, &share.ResourceType, &share.ResourceID,
			&share.Description, &expires, &share.VisitCount, &created,
			&share.AccessMode, &share.PasswordHash, &share.InstanceID, &share.OwnerScope,
		); err != nil {
			return nil, err
		}
		if expires.Valid {
			t := time.Unix(expires.Int64, 0)
			share.ExpiresAt = &t
		}
		share.CreatedAt = time.Unix(created, 0)
		if share.AccessMode == "" {
			share.AccessMode = ShareAccessPublic
		}
		items = append(items, share)
	}
	return items, rows.Err()
}
