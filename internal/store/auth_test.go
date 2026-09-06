// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"melovian/internal/appconfig"
)

func TestAuthCreateAndAuthenticate(t *testing.T) {
	dir := t.TempDir()
	cfg := appconfig.Config{
		DataDir:      dir,
		DatabasePath: filepath.Join(dir, "test.db"),
	}
	db, err := OpenDB(cfg.DatabasePath, cfg)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	auth := NewAuthStore(db, "test-secret")
	user, err := auth.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	var storedHash string
	if err := db.queryRow(`SELECT password_hash FROM melovian_users WHERE id = ?`, user.ID).Scan(&storedHash); err != nil {
		t.Fatalf("read hash: %v", err)
	}
	if !strings.HasPrefix(storedHash, "$argon2id$") {
		t.Fatalf("expected argon2id hash on create, got %q", storedHash)
	}

	got, err := auth.Authenticate("alice", "password123")
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("expected user id %q, got %q", user.ID, got.ID)
	}

	token, _, err := auth.CreateSession(user.ID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	userID, err := auth.UserIDFromToken(token)
	if err != nil {
		t.Fatalf("UserIDFromToken: %v", err)
	}
	if userID != user.ID {
		t.Fatalf("expected session user %q, got %q", user.ID, userID)
	}
}

func TestAuthFindOrCreateOIDCUser(t *testing.T) {
	dir := t.TempDir()
	cfg := appconfig.Config{
		DataDir:      dir,
		DatabasePath: filepath.Join(dir, "test.db"),
	}
	db, err := OpenDB(cfg.DatabasePath, cfg)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	auth := NewAuthStore(db, "test-secret")
	user, err := auth.FindOrCreateOIDCUser("https://issuer.example", "subject-1", "alice")
	if err != nil {
		t.Fatalf("FindOrCreateOIDCUser: %v", err)
	}
	if user.Username != "alice" {
		t.Fatalf("expected username alice, got %q", user.Username)
	}

	same, err := auth.FindOrCreateOIDCUser("https://issuer.example", "subject-1", "ignored")
	if err != nil {
		t.Fatalf("FindOrCreateOIDCUser second call: %v", err)
	}
	if same.ID != user.ID {
		t.Fatalf("expected same user id %q, got %q", user.ID, same.ID)
	}

	other, err := auth.FindOrCreateOIDCUser("https://issuer.example", "subject-2", "alice")
	if err != nil {
		t.Fatalf("FindOrCreateOIDCUser duplicate username: %v", err)
	}
	if other.ID == user.ID {
		t.Fatal("expected different user for different subject")
	}
	if other.Username == user.Username {
		t.Fatal("expected unique username when subject differs")
	}
}

func TestAuthWrongPassword(t *testing.T) {
	db := OpenTestDB(t)
	auth := NewAuthStore(db, "test-secret")
	if _, err := auth.CreateUser("alice", "password123"); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if _, err := auth.Authenticate("alice", "wrong-password"); err == nil {
		t.Fatal("expected auth failure for wrong password")
	}
}

func TestAuthValidationErrors(t *testing.T) {
	db := OpenTestDB(t)
	auth := NewAuthStore(db, "test-secret")

	if _, err := auth.CreateUser("", "password123"); err == nil {
		t.Fatal("expected empty username error")
	}
	if _, err := auth.CreateUser("alice", "short"); err == nil {
		t.Fatal("expected short password error")
	}
}

func TestAuthExpiredSessionRejected(t *testing.T) {
	db := OpenTestDB(t)
	auth := NewAuthStore(db, "test-secret")
	user, err := auth.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	token, _, err := auth.CreateSession(user.ID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	expired := time.Now().Add(-time.Hour).Unix()
	_, err = db.exec(
		`UPDATE melovian_sessions SET expires_at = ? WHERE token_hash = ?`,
		expired, hashSessionToken(auth.secret, token),
	)
	if err != nil {
		t.Fatalf("expire session: %v", err)
	}

	if _, err := auth.UserIDFromToken(token); err == nil {
		t.Fatal("expected expired session to be rejected")
	}
	if _, err := auth.UserIDFromToken(token); err == nil {
		t.Fatal("expected expired session row to be deleted")
	}
}

func TestAuthDeleteSession(t *testing.T) {
	db := OpenTestDB(t)
	auth := NewAuthStore(db, "test-secret")
	user, err := auth.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	token, _, err := auth.CreateSession(user.ID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := auth.DeleteSession(token); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if _, err := auth.UserIDFromToken(token); err == nil {
		t.Fatal("expected deleted session to be invalid")
	}
}

func TestAuthDisabledWithoutSecret(t *testing.T) {
	db := OpenTestDB(t)
	auth := NewAuthStore(db, "")
	if auth.Enabled() {
		t.Fatal("expected auth disabled without secret")
	}
}

func TestAuthUpgradesBcryptOnAuthenticate(t *testing.T) {
	db := OpenTestDB(t)
	auth := NewAuthStore(db, "test-secret")
	user, err := auth.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	bcryptHash, err := bcrypt.GenerateFromPassword([]byte("password123"), 10)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	if _, err := db.exec(
		`UPDATE melovian_users SET password_hash = ? WHERE id = ?`,
		string(bcryptHash), user.ID,
	); err != nil {
		t.Fatalf("seed bcrypt hash: %v", err)
	}

	if _, err := auth.Authenticate("alice", "password123"); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}

	var storedHash string
	if err := db.queryRow(`SELECT password_hash FROM melovian_users WHERE id = ?`, user.ID).Scan(&storedHash); err != nil {
		t.Fatalf("read hash: %v", err)
	}
	if !strings.HasPrefix(storedHash, "$argon2id$") {
		t.Fatalf("expected upgraded argon2id hash, got %q", storedHash)
	}
	ok, err := VerifyPassword(storedHash, "password123")
	if err != nil || !ok {
		t.Fatalf("upgraded hash should verify: ok=%v err=%v", ok, err)
	}
}

func TestUpdateUsername(t *testing.T) {
	db := OpenTestDB(t)
	auth := NewAuthStore(db, "test-secret")
	user, err := auth.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	updated, err := auth.UpdateUsername(user.ID, "alice_new")
	if err != nil {
		t.Fatalf("UpdateUsername: %v", err)
	}
	if updated.Username != "alice_new" {
		t.Fatalf("username = %q", updated.Username)
	}
	if _, err := auth.CreateUser("bob", "password123"); err != nil {
		t.Fatalf("CreateUser bob: %v", err)
	}
	if _, err := auth.UpdateUsername(user.ID, "bob"); err == nil {
		t.Fatal("expected taken username error")
	}
}
