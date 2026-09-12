// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"melovian/internal/appconfig"
)

func newPrefsCipherTestStore(t *testing.T) *PreferencesStore {
	t.Helper()
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

	prefs := NewPreferencesStore(db)
	if err := prefs.EnsureSecretCipher(dir); err != nil {
		t.Fatalf("EnsureSecretCipher: %v", err)
	}
	if prefs.cipher == nil {
		t.Fatal("expected cipher for populated data dir")
	}
	return prefs
}

func storedPrefValue(t *testing.T, prefs *PreferencesStore, userID, key string) string {
	t.Helper()
	var stored string
	if err := prefs.db.queryRow(
		`SELECT value FROM user_preferences WHERE user_id = ? AND key = ?`, userID, key,
	).Scan(&stored); err != nil {
		t.Fatalf("read stored value: %v", err)
	}
	return stored
}

func TestPreferenceSecretsEncryptedAtRest(t *testing.T) {
	prefs := newPrefsCipherTestStore(t)

	raw := `{"apiKey":"key-123","apiSecret":"shhh-secret","sessionKey":"sess-abc","endpoint":"https://scrobble.example/"}`
	if err := prefs.Set("u1", PrefKeyLastFMSettings, raw); err != nil {
		t.Fatalf("Set: %v", err)
	}

	stored := storedPrefValue(t, prefs, "u1", PrefKeyLastFMSettings)
	for _, secret := range []string{"key-123", "shhh-secret", "sess-abc"} {
		if strings.Contains(stored, secret) {
			t.Fatalf("plaintext secret %q present in stored value %q", secret, stored)
		}
	}
	if !strings.Contains(stored, secretCipherPrefix) {
		t.Fatalf("credential fields stored unencrypted: %q", stored)
	}
	if !strings.Contains(stored, "https://scrobble.example/") {
		t.Fatalf("non-secret endpoint should stay readable, got %q", stored)
	}

	got, err := prefs.Get("u1", PrefKeyLastFMSettings)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(got), &payload); err != nil {
		t.Fatalf("stored blob is not valid JSON: %v", err)
	}
	if payload["apiKey"] != "key-123" || payload["apiSecret"] != "shhh-secret" || payload["sessionKey"] != "sess-abc" {
		t.Fatalf("credentials did not decrypt on read: %q", got)
	}
	if payload["endpoint"] != "https://scrobble.example/" {
		t.Fatalf("endpoint changed on round trip: %q", payload["endpoint"])
	}
}

func TestPreferenceSecretsLegacyMigration(t *testing.T) {
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
	prefs := NewPreferencesStore(db)

	// Seed a row the way pre-encryption builds wrote it.
	if _, err := db.exec(
		`INSERT INTO user_preferences (user_id, key, value, updated_at) VALUES (?, ?, ?, 1)`,
		"u1", PrefKeyRockskySettings, `{"token":"tok-plaintext"}`,
	); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	if err := prefs.EnsureSecretCipher(dir); err != nil {
		t.Fatalf("EnsureSecretCipher: %v", err)
	}

	stored := storedPrefValue(t, prefs, "u1", PrefKeyRockskySettings)
	if strings.Contains(stored, "tok-plaintext") {
		t.Fatalf("legacy token not migrated: %q", stored)
	}
	if !strings.Contains(stored, secretCipherPrefix) {
		t.Fatalf("legacy token not migrated: %q", stored)
	}

	got, err := prefs.Get("u1", PrefKeyRockskySettings)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(got), &payload); err != nil {
		t.Fatalf("stored blob is not valid JSON: %v", err)
	}
	if payload["token"] != "tok-plaintext" {
		t.Fatalf("migrated token should decrypt to plaintext, got %q", payload["token"])
	}
}

func TestPreferenceSecretsLegacyReadWithoutMigration(t *testing.T) {
	prefs := newPrefsCipherTestStore(t)

	// A plaintext row written after the cipher loaded still reads back.
	if _, err := prefs.db.exec(
		`INSERT INTO user_preferences (user_id, key, value, updated_at) VALUES (?, ?, ?, 1)`,
		"u1", PrefKeyListenBrainzSettings, `{"token":"tok-legacy","endpoint":"https://lb.example"}`,
	); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}
	got, err := prefs.Get("u1", PrefKeyListenBrainzSettings)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(got), &payload); err != nil {
		t.Fatalf("legacy blob is not valid JSON: %v", err)
	}
	if payload["token"] != "tok-legacy" {
		t.Fatalf("legacy token should read as plaintext, got %q", payload["token"])
	}

	// The next write upgrades it.
	if err := prefs.Set("u1", PrefKeyListenBrainzSettings, got); err != nil {
		t.Fatalf("Set: %v", err)
	}
	stored := storedPrefValue(t, prefs, "u1", PrefKeyListenBrainzSettings)
	if strings.Contains(stored, "tok-legacy") {
		t.Fatalf("rewritten blob still holds plaintext: %q", stored)
	}
}

func TestPreferenceNonSecretKeysPassThrough(t *testing.T) {
	prefs := newPrefsCipherTestStore(t)

	raw := `{"bands":[0,1,2],"preamp":-3}`
	if err := prefs.Set("u1", PrefKeyMusicEq, raw); err != nil {
		t.Fatalf("Set: %v", err)
	}
	stored := storedPrefValue(t, prefs, "u1", PrefKeyMusicEq)
	if stored != raw {
		t.Fatalf("non-secret preference rewritten: %q", stored)
	}
	got, err := prefs.Get("u1", PrefKeyMusicEq)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != raw {
		t.Fatalf("non-secret preference changed on read: %q", got)
	}
}

func TestPreferencesWithoutCipherStorePlaintext(t *testing.T) {
	db := OpenTestDB(t)
	prefs := NewPreferencesStore(db)
	raw := `{"token":"tok-plain"}`
	if err := prefs.Set("u1", PrefKeyRockskySettings, raw); err != nil {
		t.Fatalf("Set: %v", err)
	}
	stored := storedPrefValue(t, prefs, "u1", PrefKeyRockskySettings)
	if stored != raw {
		t.Fatalf("expected plaintext pass-through without cipher, got %q", stored)
	}
}
