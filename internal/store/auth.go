// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	sessionTTL       = 30 * 24 * time.Hour
	sessionCookieKey = "melovian_session"
)

type AuthUser struct {
	ID        string
	Username  string
	CreatedAt time.Time
}

type AuthSession struct {
	ID        string
	ExpiresAt time.Time
	Current   bool
}

type AuthStore struct {
	db     *DB
	secret []byte
}

func NewAuthStore(db *DB, secret string) *AuthStore {
	return &AuthStore{
		db:     db,
		secret: []byte(strings.TrimSpace(secret)),
	}
}

func (s *AuthStore) Enabled() bool {
	return len(s.secret) > 0
}

func (s *AuthStore) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS melovian_users (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS melovian_sessions (
	token_hash TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES melovian_users(id) ON DELETE CASCADE,
	expires_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_melovian_sessions_user ON melovian_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_melovian_sessions_expires ON melovian_sessions(expires_at);
`
	if err := s.db.execScript(schema); err != nil {
		return err
	}
	if err := s.migrateOIDCColumns(); err != nil {
		return err
	}
	return s.migrateSubsonicColumns()
}

func (s *AuthStore) migrateSubsonicColumns() error {
	_, err := s.db.exec(`ALTER TABLE melovian_users ADD COLUMN subsonic_api_secret TEXT NOT NULL DEFAULT ''`)
	if err != nil && !s.db.dialect.IsDuplicateColumn(err) {
		return err
	}
	return nil
}

func (s *AuthStore) migrateOIDCColumns() error {
	columns := []string{
		`ALTER TABLE melovian_users ADD COLUMN oidc_issuer TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE melovian_users ADD COLUMN oidc_subject TEXT NOT NULL DEFAULT ''`,
	}
	for _, stmt := range columns {
		if _, err := s.db.exec(stmt); err != nil {
			if !s.db.dialect.IsDuplicateColumn(err) {
				return err
			}
		}
	}
	_, err := s.db.exec(`
CREATE UNIQUE INDEX IF NOT EXISTS idx_melovian_users_oidc
ON melovian_users(oidc_issuer, oidc_subject)
WHERE oidc_issuer != '' AND oidc_subject != ''`)
	return err
}

func (s *AuthStore) CountUsers() (int, error) {
	var count int
	err := s.db.queryRow(`SELECT COUNT(*) FROM melovian_users`).Scan(&count)
	return count, err
}

func (s *AuthStore) CreateUser(username, password string) (AuthUser, error) {
	name := strings.TrimSpace(username)
	if name == "" {
		return AuthUser{}, fmt.Errorf("username is required")
	}
	if len(password) < 8 {
		return AuthUser{}, fmt.Errorf("password must be at least 8 characters")
	}

	hash, err := HashPassword(password)
	if err != nil {
		return AuthUser{}, err
	}

	id := newInstanceID()
	now := nowUnix()
	_, err = s.db.exec(
		`INSERT INTO melovian_users (id, username, password_hash, subsonic_api_secret, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, name, hash, password, now,
	)
	if err != nil {
		return AuthUser{}, err
	}
	return AuthUser{
		ID:        id,
		Username:  name,
		CreatedAt: time.Unix(now, 0),
	}, nil
}

func (s *AuthStore) Authenticate(username, password string) (AuthUser, error) {
	name := strings.TrimSpace(username)
	row := s.db.queryRow(
		`SELECT id, username, password_hash, created_at FROM melovian_users WHERE username = ?`,
		name,
	)
	var user AuthUser
	var created int64
	var hash string
	if err := row.Scan(&user.ID, &user.Username, &hash, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AuthUser{}, fmt.Errorf("invalid credentials")
		}
		return AuthUser{}, err
	}
	if hash == "" {
		return AuthUser{}, fmt.Errorf("invalid credentials")
	}
	ok, err := VerifyPassword(hash, password)
	if err != nil || !ok {
		return AuthUser{}, fmt.Errorf("invalid credentials")
	}
	if NeedsRehash(hash) {
		if newHash, hashErr := HashPassword(password); hashErr == nil {
			_, _ = s.db.exec(
				`UPDATE melovian_users SET password_hash = ? WHERE id = ?`,
				newHash, user.ID,
			)
		}
	}
	user.CreatedAt = time.Unix(created, 0)
	return user, nil
}

func (s *AuthStore) GetUser(id string) (AuthUser, error) {
	row := s.db.queryRow(
		`SELECT id, username, created_at FROM melovian_users WHERE id = ?`,
		id,
	)
	var user AuthUser
	var created int64
	if err := row.Scan(&user.ID, &user.Username, &created); err != nil {
		return AuthUser{}, err
	}
	user.CreatedAt = time.Unix(created, 0)
	return user, nil
}

func (s *AuthStore) FindOrCreateOIDCUser(issuer, subject, username string) (AuthUser, error) {
	iss := strings.TrimSpace(issuer)
	sub := strings.TrimSpace(subject)
	if iss == "" || sub == "" {
		return AuthUser{}, fmt.Errorf("oidc issuer and subject are required")
	}

	row := s.db.queryRow(
		`SELECT id, username, created_at FROM melovian_users WHERE oidc_issuer = ? AND oidc_subject = ?`,
		iss, sub,
	)
	var user AuthUser
	var created int64
	err := row.Scan(&user.ID, &user.Username, &created)
	if err == nil {
		user.CreatedAt = time.Unix(created, 0)
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return AuthUser{}, err
	}

	name := normalizeOIDCUsername(username, sub)
	for attempt := range 5 {
		candidate := name
		if attempt > 0 {
			candidate = fmt.Sprintf("%s-%d", name, attempt+1)
		}
		id := newInstanceID()
		now := nowUnix()
		_, err := s.db.exec(
			`INSERT INTO melovian_users (id, username, password_hash, created_at, oidc_issuer, oidc_subject) VALUES (?, ?, '', ?, ?, ?)`,
			id, candidate, now, iss, sub,
		)
		if err == nil {
			return AuthUser{
				ID:        id,
				Username:  candidate,
				CreatedAt: time.Unix(now, 0),
			}, nil
		}
		if !strings.Contains(strings.ToLower(err.Error()), "unique") {
			return AuthUser{}, err
		}
	}
	return AuthUser{}, fmt.Errorf("failed to create oidc user")
}

func normalizeOIDCUsername(username, subject string) string {
	name := strings.TrimSpace(username)
	if name == "" {
		name = strings.TrimSpace(subject)
	}
	if name == "" {
		name = "user"
	}
	if len(name) > 64 {
		name = name[:64]
	}
	return name
}

func (s *AuthStore) CreateSession(userID string) (string, time.Time, error) {
	token, err := randomToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().Add(sessionTTL)
	_, err = s.db.exec(
		`INSERT INTO melovian_sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)`,
		hashSessionToken(s.secret, token), userID, expires.Unix(),
	)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

func (s *AuthStore) DeleteSession(token string) error {
	_, err := s.db.exec(
		`DELETE FROM melovian_sessions WHERE token_hash = ?`,
		hashSessionToken(s.secret, token),
	)
	return err
}

func (s *AuthStore) UserIDFromToken(token string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("missing session")
	}
	row := s.db.queryRow(
		`SELECT user_id, expires_at FROM melovian_sessions WHERE token_hash = ?`,
		hashSessionToken(s.secret, token),
	)
	var userID string
	var expires int64
	if err := row.Scan(&userID, &expires); err != nil {
		return "", err
	}
	if time.Now().Unix() > expires {
		_, _ = s.db.exec(
			`DELETE FROM melovian_sessions WHERE token_hash = ?`,
			hashSessionToken(s.secret, token),
		)
		return "", fmt.Errorf("session expired")
	}
	return userID, nil
}

func (s *AuthStore) PurgeExpiredSessions() {
	_, _ = s.db.exec(`DELETE FROM melovian_sessions WHERE expires_at < ?`, nowUnix())
}

func (s *AuthStore) ListSessions(userID, currentToken string) ([]AuthSession, error) {
	currentHash := ""
	if currentToken != "" {
		currentHash = hashSessionToken(s.secret, currentToken)
	}
	rows, err := s.db.query(
		`SELECT token_hash, expires_at FROM melovian_sessions WHERE user_id = ? AND expires_at >= ? ORDER BY expires_at DESC`,
		userID, nowUnix(),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var sessions []AuthSession
	for rows.Next() {
		var tokenHash string
		var expires int64
		if err := rows.Scan(&tokenHash, &expires); err != nil {
			return nil, err
		}
		sessions = append(sessions, AuthSession{
			ID:        tokenHash[:16],
			ExpiresAt: time.Unix(expires, 0),
			Current:   tokenHash == currentHash,
		})
	}
	return sessions, rows.Err()
}

func (s *AuthStore) RevokeSession(userID, sessionID, currentToken string) error {
	currentHash := ""
	if currentToken != "" {
		currentHash = hashSessionToken(s.secret, currentToken)
	}
	rows, err := s.db.query(
		`SELECT token_hash FROM melovian_sessions WHERE user_id = ? AND expires_at >= ?`,
		userID, nowUnix(),
	)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var tokenHash string
		if err := rows.Scan(&tokenHash); err != nil {
			return err
		}
		if tokenHash[:16] != sessionID {
			continue
		}
		if tokenHash == currentHash {
			return fmt.Errorf("cannot revoke current session")
		}
		_, err := s.db.exec(`DELETE FROM melovian_sessions WHERE token_hash = ?`, tokenHash)
		return err
	}
	return sql.ErrNoRows
}

func (s *AuthStore) ChangePassword(userID, currentPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	row := s.db.queryRow(
		`SELECT password_hash FROM melovian_users WHERE id = ?`,
		userID,
	)
	var hash string
	if err := row.Scan(&hash); err != nil {
		return err
	}
	if hash == "" {
		return fmt.Errorf("password change not available for this account")
	}
	ok, err := VerifyPassword(hash, currentPassword)
	if err != nil || !ok {
		return fmt.Errorf("invalid credentials")
	}
	newHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	_, err = s.db.exec(
		`UPDATE melovian_users SET password_hash = ?, subsonic_api_secret = ? WHERE id = ?`,
		newHash, newPassword, userID,
	)
	return err
}

func (s *AuthStore) UpdateUsername(userID, newUsername string) (AuthUser, error) {
	name := strings.TrimSpace(newUsername)
	if name == "" {
		return AuthUser{}, fmt.Errorf("username is required")
	}
	if len(name) < 2 {
		return AuthUser{}, fmt.Errorf("username must be at least 2 characters")
	}
	if len(name) > 64 {
		return AuthUser{}, fmt.Errorf("username must be at most 64 characters")
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			continue
		}
		return AuthUser{}, fmt.Errorf("username may only contain letters, numbers, '_', '-', or '.'")
	}

	current, err := s.GetUser(userID)
	if err != nil {
		return AuthUser{}, err
	}
	if strings.EqualFold(current.Username, name) {
		if current.Username == name {
			return current, nil
		}
	}

	existing, err := s.GetUserByUsername(name)
	if err == nil && existing.ID != userID {
		return AuthUser{}, fmt.Errorf("username already taken")
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return AuthUser{}, err
	}

	_, err = s.db.exec(`UPDATE melovian_users SET username = ? WHERE id = ?`, name, userID)
	if err != nil {
		return AuthUser{}, err
	}
	return s.GetUser(userID)
}

func (s *AuthStore) GetUserByUsername(username string) (AuthUser, error) {
	name := strings.TrimSpace(username)
	row := s.db.queryRow(
		`SELECT id, username, created_at FROM melovian_users WHERE username = ?`,
		name,
	)
	var user AuthUser
	var created int64
	if err := row.Scan(&user.ID, &user.Username, &created); err != nil {
		return AuthUser{}, err
	}
	user.CreatedAt = time.Unix(created, 0)
	return user, nil
}

func (s *AuthStore) SubsonicAPISecret(username string) (string, error) {
	name := strings.TrimSpace(username)
	row := s.db.queryRow(
		`SELECT subsonic_api_secret FROM melovian_users WHERE username = ?`,
		name,
	)
	var secret string
	if err := row.Scan(&secret); err != nil {
		return "", err
	}
	if strings.TrimSpace(secret) == "" {
		return "", fmt.Errorf("subsonic api secret not configured")
	}
	return secret, nil
}

func (s *AuthStore) DeleteUser(userID, password string) error {
	row := s.db.queryRow(
		`SELECT password_hash FROM melovian_users WHERE id = ?`,
		userID,
	)
	var hash string
	if err := row.Scan(&hash); err != nil {
		return err
	}
	if hash != "" {
		ok, err := VerifyPassword(hash, password)
		if err != nil || !ok {
			return fmt.Errorf("invalid credentials")
		}
	}
	_, err := s.db.exec(`DELETE FROM melovian_users WHERE id = ?`, userID)
	return err
}

func SessionCookieName() string {
	return sessionCookieKey
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashSessionToken(secret []byte, token string) string {
	sum := sha256.Sum256(append(secret, []byte(token)...))
	return hex.EncodeToString(sum[:])
}
