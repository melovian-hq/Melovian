// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestSharePasswordUsesArgon2id(t *testing.T) {
	db := OpenTestDB(t)
	shares := NewShareStore(db)

	share, err := shares.Create(CreateShareInput{
		UserID:       "user-1",
		ResourceType: "album",
		ResourceID:   "album-1",
		AccessMode:   ShareAccessPassword,
		Password:     "share-secret",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !strings.HasPrefix(share.PasswordHash, "$argon2id$") {
		t.Fatalf("expected argon2id share hash, got %q", share.PasswordHash)
	}
	if !shares.CheckPassword(share, "share-secret") {
		t.Fatal("expected share password to verify")
	}
	if shares.CheckPassword(share, "wrong") {
		t.Fatal("expected wrong share password to fail")
	}
}

func TestSharePasswordUpgradesBcrypt(t *testing.T) {
	db := OpenTestDB(t)
	shares := NewShareStore(db)

	share, err := shares.Create(CreateShareInput{
		UserID:       "user-1",
		ResourceType: "album",
		ResourceID:   "album-1",
		AccessMode:   ShareAccessPassword,
		Password:     "share-secret",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	bcryptHash, err := bcrypt.GenerateFromPassword([]byte("share-secret"), 10)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	if _, err := db.exec(
		`UPDATE music_shares SET password_hash = ? WHERE id = ?`,
		string(bcryptHash), share.ID,
	); err != nil {
		t.Fatalf("seed bcrypt hash: %v", err)
	}
	share.PasswordHash = string(bcryptHash)

	if !shares.CheckPassword(share, "share-secret") {
		t.Fatal("expected bcrypt share password to verify")
	}

	var storedHash string
	if err := db.queryRow(`SELECT password_hash FROM music_shares WHERE id = ?`, share.ID).Scan(&storedHash); err != nil {
		t.Fatalf("read hash: %v", err)
	}
	if !strings.HasPrefix(storedHash, "$argon2id$") {
		t.Fatalf("expected upgraded argon2id share hash, got %q", storedHash)
	}
}
