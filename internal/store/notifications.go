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
)

const (
	NotificationShareReceived = "share_received"
	NotificationPartyInvite   = "party_invite"
	NotificationPartyJoined   = "party_joined"
	NotificationPartyLeft     = "party_left"
	NotificationPartyEnded    = "party_ended"
)

type Notification struct {
	ID        string
	UserID    string
	Kind      string
	Title     string
	Body      string
	Href      string
	Payload   string
	ReadAt    *time.Time
	CreatedAt time.Time
}

type CreateNotificationInput struct {
	UserID  string
	Kind    string
	Title   string
	Body    string
	Href    string
	Payload string
}

type NotificationStore struct {
	db *DB
}

func NewNotificationStore(db *DB) *NotificationStore {
	return &NotificationStore{db: db}
}

func (s *NotificationStore) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS notifications (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	kind TEXT NOT NULL,
	title TEXT NOT NULL,
	body TEXT NOT NULL DEFAULT '',
	href TEXT NOT NULL DEFAULT '',
	payload_json TEXT NOT NULL DEFAULT '',
	read_at INTEGER,
	created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_notifications_user_created
	ON notifications(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread
	ON notifications(user_id, read_at);
`
	return s.db.execScript(schema)
}

func (s *NotificationStore) Create(input CreateNotificationInput) (Notification, error) {
	userID := strings.TrimSpace(input.UserID)
	kind := strings.TrimSpace(input.Kind)
	title := strings.TrimSpace(input.Title)
	if userID == "" || kind == "" || title == "" {
		return Notification{}, errors.New("user, kind, and title are required")
	}
	id := newNotificationID()
	now := time.Now().Unix()
	_, err := s.db.exec(
		`INSERT INTO notifications (
			id, user_id, kind, title, body, href, payload_json, read_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, NULL, ?)`,
		id, userID, kind, title,
		strings.TrimSpace(input.Body),
		strings.TrimSpace(input.Href),
		strings.TrimSpace(input.Payload),
		now,
	)
	if err != nil {
		return Notification{}, err
	}
	return s.Get(userID, id)
}

func (s *NotificationStore) Get(userID, id string) (Notification, error) {
	row := s.db.queryRow(
		`SELECT id, user_id, kind, title, body, href, payload_json, read_at, created_at
		 FROM notifications WHERE id = ? AND user_id = ?`,
		id, userID,
	)
	return scanNotification(row)
}

func (s *NotificationStore) List(userID string, limit, offset int) ([]Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.query(
		`SELECT id, user_id, kind, title, body, href, payload_json, read_at, created_at
		 FROM notifications WHERE user_id = ?
		 ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNotifications(rows)
}

func (s *NotificationStore) UnreadCount(userID string) (int, error) {
	var count int
	err := s.db.queryRow(
		`SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read_at IS NULL`,
		userID,
	).Scan(&count)
	return count, err
}

func (s *NotificationStore) MarkRead(userID, id string) (Notification, error) {
	now := time.Now().Unix()
	res, err := s.db.exec(
		`UPDATE notifications SET read_at = COALESCE(read_at, ?)
		 WHERE id = ? AND user_id = ?`,
		now, id, userID,
	)
	if err != nil {
		return Notification{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Notification{}, err
	}
	if n == 0 {
		return Notification{}, sql.ErrNoRows
	}
	return s.Get(userID, id)
}

func (s *NotificationStore) MarkAllRead(userID string) (int64, error) {
	now := time.Now().Unix()
	res, err := s.db.exec(
		`UPDATE notifications SET read_at = ? WHERE user_id = ? AND read_at IS NULL`,
		now, userID,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *NotificationStore) Delete(userID, id string) error {
	res, err := s.db.exec(
		`DELETE FROM notifications WHERE id = ? AND user_id = ?`,
		id, userID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func newNotificationID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("ntf_%d", time.Now().UnixNano())
	}
	return "ntf_" + hex.EncodeToString(buf)
}

func scanNotification(row *sql.Row) (Notification, error) {
	var n Notification
	var created int64
	var readAt sql.NullInt64
	err := row.Scan(
		&n.ID, &n.UserID, &n.Kind, &n.Title, &n.Body, &n.Href, &n.Payload, &readAt, &created,
	)
	if err != nil {
		return Notification{}, err
	}
	if readAt.Valid {
		t := time.Unix(readAt.Int64, 0)
		n.ReadAt = &t
	}
	n.CreatedAt = time.Unix(created, 0)
	return n, nil
}

func scanNotifications(rows *sql.Rows) ([]Notification, error) {
	var items []Notification
	for rows.Next() {
		var n Notification
		var created int64
		var readAt sql.NullInt64
		if err := rows.Scan(
			&n.ID, &n.UserID, &n.Kind, &n.Title, &n.Body, &n.Href, &n.Payload, &readAt, &created,
		); err != nil {
			return nil, err
		}
		if readAt.Valid {
			t := time.Unix(readAt.Int64, 0)
			n.ReadAt = &t
		}
		n.CreatedAt = time.Unix(created, 0)
		items = append(items, n)
	}
	return items, rows.Err()
}
