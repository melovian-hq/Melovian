// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestSubsonicAPISecretIsNotPlaintextPassword(t *testing.T) {
	db := OpenTestDB(t)
	auth := NewAuthStore(db, "test-secret")

	user, err := auth.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	secret, err := auth.SubsonicAPISecret("alice")
	if err != nil {
		t.Fatalf("SubsonicAPISecret: %v", err)
	}
	if secret == "password123" {
		t.Fatal("subsonic api secret must not store the account password")
	}
	if _, err := hex.DecodeString(secret); err != nil || len(secret) != 64 {
		t.Fatalf("expected 64 char hex secret, got %q", secret)
	}

	var stored string
	if err := db.queryRow(
		`SELECT subsonic_api_secret FROM melovian_users WHERE id = ?`, user.ID,
	).Scan(&stored); err != nil {
		t.Fatalf("read secret: %v", err)
	}
	if stored == "password123" {
		t.Fatal("account password stored in plaintext in subsonic_api_secret")
	}
}

func TestAuthenticateSubsonicAcceptsPasswordAndKey(t *testing.T) {
	db := OpenTestDB(t)
	auth := NewAuthStore(db, "test-secret")

	user, err := auth.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	secret, err := auth.SubsonicAPISecret("alice")
	if err != nil {
		t.Fatalf("SubsonicAPISecret: %v", err)
	}

	got, err := auth.AuthenticateSubsonic("alice", "password123")
	if err != nil || got.ID != user.ID {
		t.Fatalf("password auth: %v", err)
	}
	got, err = auth.AuthenticateSubsonic("alice", secret)
	if err != nil || got.ID != user.ID {
		t.Fatalf("api key auth: %v", err)
	}
	if _, err := auth.AuthenticateSubsonic("alice", "wrong"); err == nil {
		t.Fatal("expected failure for wrong credential")
	}
}

func TestChangePasswordRotatesSubsonicSecret(t *testing.T) {
	db := OpenTestDB(t)
	auth := NewAuthStore(db, "test-secret")

	user, err := auth.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	before, err := auth.SubsonicAPISecret("alice")
	if err != nil {
		t.Fatalf("SubsonicAPISecret: %v", err)
	}
	if err := auth.ChangePassword(user.ID, "password123", "newpassword456"); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
	after, err := auth.SubsonicAPISecret("alice")
	if err != nil {
		t.Fatalf("SubsonicAPISecret: %v", err)
	}
	if after == before || after == "newpassword456" {
		t.Fatal("password change must rotate the api secret to a random value")
	}
	if _, err := auth.AuthenticateSubsonic("alice", before); err == nil {
		t.Fatal("old api key must stop working after rotation")
	}
}

func TestLegacyPasswordSecretMigratesOnLogin(t *testing.T) {
	db := OpenTestDB(t)
	auth := NewAuthStore(db, "test-secret")

	user, err := auth.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	// Simulate a row written before the secret was decoupled from the
	// account password.
	if _, err := db.exec(
		`UPDATE melovian_users SET subsonic_api_secret = ? WHERE id = ?`,
		"password123", user.ID,
	); err != nil {
		t.Fatalf("seed legacy secret: %v", err)
	}

	if _, err := auth.Authenticate("alice", "password123"); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	secret, err := auth.SubsonicAPISecret("alice")
	if err != nil {
		t.Fatalf("SubsonicAPISecret: %v", err)
	}
	if secret == "password123" {
		t.Fatal("legacy plaintext password must be replaced on successful login")
	}
	if _, err := hex.DecodeString(secret); err != nil || len(secret) != 64 {
		t.Fatalf("expected migrated secret to be random hex, got %q", secret)
	}
}

func TestRegenerateSubsonicAPISecret(t *testing.T) {
	db := OpenTestDB(t)
	auth := NewAuthStore(db, "test-secret")

	user, err := auth.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	first, err := auth.SubsonicAPISecretForUser(user.ID)
	if err != nil {
		t.Fatalf("SubsonicAPISecretForUser: %v", err)
	}
	second, err := auth.RegenerateSubsonicAPISecret(user.ID)
	if err != nil {
		t.Fatalf("RegenerateSubsonicAPISecret: %v", err)
	}
	if first == second {
		t.Fatal("regenerate must produce a new key")
	}
	if !strings.EqualFold(first, first) || len(second) != 64 {
		t.Fatal("unexpected key format")
	}
}
