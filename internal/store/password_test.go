// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("expected argon2id hash, got %q", hash)
	}
	ok, err := VerifyPassword(hash, "correct-horse")
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatal("expected password to verify")
	}
	ok, err = VerifyPassword(hash, "wrong-password")
	if err != nil {
		t.Fatalf("VerifyPassword wrong: %v", err)
	}
	if ok {
		t.Fatal("expected wrong password to fail")
	}
	if NeedsRehash(hash) {
		t.Fatal("expected current argon2id hash to not need rehash")
	}
}

func TestVerifyPasswordBcryptLegacy(t *testing.T) {
	bcryptHash, err := bcrypt.GenerateFromPassword([]byte("legacy-secret"), 10)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	hash := string(bcryptHash)
	ok, err := VerifyPassword(hash, "legacy-secret")
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatal("expected bcrypt hash to verify")
	}
	ok, err = VerifyPassword(hash, "nope")
	if err != nil {
		t.Fatalf("VerifyPassword wrong: %v", err)
	}
	if ok {
		t.Fatal("expected wrong password to fail against bcrypt")
	}
	if !NeedsRehash(hash) {
		t.Fatal("expected bcrypt hash to need rehash")
	}
}

func TestVerifyPasswordRejectsUnknown(t *testing.T) {
	ok, err := VerifyPassword("", "x")
	if err == nil || ok {
		t.Fatal("expected empty hash rejection")
	}
	ok, err = VerifyPassword("$scrypt$not-supported", "x")
	if err == nil || ok {
		t.Fatal("expected unknown format rejection")
	}
}

func TestNeedsRehashStaleArgon2Params(t *testing.T) {
	stale := encodeArgon2id([]byte("0123456789abcdef"), []byte("abcdefghijklmnopqrstuvwxyzabcdef"), 1, 1024, 1)
	if !NeedsRehash(stale) {
		t.Fatal("expected stale argon2id params to need rehash")
	}
}
