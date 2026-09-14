// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"encoding/hex"
	"testing"
)

func TestHasPassword(t *testing.T) {
	db := OpenTestDB(t)
	auth := NewAuthStore(db, "test-secret")

	local, err := auth.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	hasPassword, err := auth.HasPassword(local.ID)
	if err != nil {
		t.Fatalf("HasPassword: %v", err)
	}
	if !hasPassword {
		t.Fatal("expected local user to report a password")
	}

	oidcUser, err := auth.FindOrCreateOIDCUser("https://issuer.example", "subject-1", "bob")
	if err != nil {
		t.Fatalf("FindOrCreateOIDCUser: %v", err)
	}
	hasPassword, err = auth.HasPassword(oidcUser.ID)
	if err != nil {
		t.Fatalf("HasPassword: %v", err)
	}
	if hasPassword {
		t.Fatal("expected oidc user to report no password")
	}

	if _, err := auth.HasPassword("missing-user"); err == nil {
		t.Fatal("expected error for unknown user")
	}
}

func TestOIDCUserLazySubsonicSecret(t *testing.T) {
	db := OpenTestDB(t)
	auth := NewAuthStore(db, "test-secret")

	user, err := auth.FindOrCreateOIDCUser("oauth2:idp.example", "subject-1", "alice")
	if err != nil {
		t.Fatalf("FindOrCreateOIDCUser: %v", err)
	}
	if _, err := auth.SubsonicAPISecretForUser(user.ID); err == nil {
		t.Fatal("expected no subsonic secret for a fresh oidc user")
	}

	secret, err := auth.RegenerateSubsonicAPISecret(user.ID)
	if err != nil {
		t.Fatalf("RegenerateSubsonicAPISecret: %v", err)
	}
	if _, err := hex.DecodeString(secret); err != nil || len(secret) != 64 {
		t.Fatalf("expected 64 char hex secret, got %q", secret)
	}

	stored, err := auth.SubsonicAPISecretForUser(user.ID)
	if err != nil {
		t.Fatalf("SubsonicAPISecretForUser: %v", err)
	}
	if stored != secret {
		t.Fatal("expected stored secret to match the generated one")
	}

	got, err := auth.AuthenticateSubsonic("alice", secret)
	if err != nil || got.ID != user.ID {
		t.Fatalf("api key auth for oidc user: %v", err)
	}
}

func TestCreateOIDCUserReturnsExistingOnSubjectConflict(t *testing.T) {
	db := OpenTestDB(t)
	auth := NewAuthStore(db, "test-secret")

	first, err := auth.createOIDCUser("https://issuer.example", "subject-1", "alice")
	if err != nil {
		t.Fatalf("createOIDCUser: %v", err)
	}

	// The losing side of a concurrent first login hits the
	// (oidc_issuer, oidc_subject) unique index. It must return the row the
	// winner created rather than retrying usernames into failure.
	again, err := auth.createOIDCUser("https://issuer.example", "subject-1", "alice")
	if err != nil {
		t.Fatalf("second createOIDCUser: %v", err)
	}
	if again.ID != first.ID {
		t.Fatalf("expected existing user %q, got %q", first.ID, again.ID)
	}
}

func TestCreateOIDCUserRetriesOnlyOnUsernameConflict(t *testing.T) {
	db := OpenTestDB(t)
	auth := NewAuthStore(db, "test-secret")

	if _, err := auth.CreateUser("alice", "password123"); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	user, err := auth.createOIDCUser("https://issuer.example", "subject-9", "alice")
	if err != nil {
		t.Fatalf("createOIDCUser: %v", err)
	}
	if user.Username != "alice-2" {
		t.Fatalf("expected a suffixed username on conflict, got %q", user.Username)
	}

	again, err := auth.FindOrCreateOIDCUser("https://issuer.example", "subject-9", "alice")
	if err != nil {
		t.Fatalf("FindOrCreateOIDCUser: %v", err)
	}
	if again.ID != user.ID {
		t.Fatalf("expected stable identity %q, got %q", user.ID, again.ID)
	}
}
