// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"path/filepath"
	"strings"
	"testing"

	"melovian/internal/appconfig"
)

func newCipherTestStore(t *testing.T) *InstanceStore {
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

	cipher, err := LoadSecretCipher(dir)
	if err != nil {
		t.Fatalf("LoadSecretCipher: %v", err)
	}
	if cipher == nil {
		t.Fatal("expected cipher for populated data dir")
	}
	instances := NewInstanceStore(db)
	instances.SetCipher(cipher)
	return instances
}

func TestInstancePasswordEncryptedAtRest(t *testing.T) {
	instances := newCipherTestStore(t)

	inst, err := instances.Create(CreateInstanceInput{
		Name:      "test",
		ServerURL: "https://music.example",
		Username:  "alice",
		Password:  "hunter2",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if inst.Password != "hunter2" {
		t.Fatalf("expected decrypted password on read, got %q", inst.Password)
	}

	var stored string
	if err := instances.db.queryRow(
		`SELECT password FROM subsonic_instances WHERE id = ?`, inst.ID,
	).Scan(&stored); err != nil {
		t.Fatalf("read stored password: %v", err)
	}
	if !strings.HasPrefix(stored, secretCipherPrefix) {
		t.Fatalf("password stored unencrypted: %q", stored)
	}
	if strings.Contains(stored, "hunter2") {
		t.Fatal("plaintext password present in stored value")
	}
}

func TestInstancePasswordLegacyMigration(t *testing.T) {
	instances := newCipherTestStore(t)

	if _, err := instances.db.exec(
		`INSERT INTO subsonic_instances (id, name, server_url, username, password, server_name, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 1, 1)`,
		"legacy-1", "legacy", "https://music.example", "alice", "hunter2", "",
	); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	// Legacy plaintext still reads correctly.
	inst, err := instances.Get("legacy-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if inst.Password != "hunter2" {
		t.Fatalf("legacy password should read as plaintext, got %q", inst.Password)
	}

	if err := instances.MigratePasswords(); err != nil {
		t.Fatalf("MigratePasswords: %v", err)
	}
	var stored string
	if err := instances.db.queryRow(
		`SELECT password FROM subsonic_instances WHERE id = ?`, "legacy-1",
	).Scan(&stored); err != nil {
		t.Fatalf("read stored: %v", err)
	}
	if !strings.HasPrefix(stored, secretCipherPrefix) {
		t.Fatalf("legacy password not migrated: %q", stored)
	}

	inst, err = instances.Get("legacy-1")
	if err != nil {
		t.Fatalf("Get after migration: %v", err)
	}
	if inst.Password != "hunter2" {
		t.Fatalf("migrated password should decrypt to plaintext, got %q", inst.Password)
	}
}

func TestInstanceUpdateKeepsEncryption(t *testing.T) {
	instances := newCipherTestStore(t)

	inst, err := instances.Create(CreateInstanceInput{
		Name:      "test",
		ServerURL: "https://music.example",
		Username:  "alice",
		Password:  "hunter2",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := instances.Update(inst.ID, UpdateInstanceInput{Name: "renamed"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	var stored string
	if err := instances.db.queryRow(
		`SELECT password FROM subsonic_instances WHERE id = ?`, inst.ID,
	).Scan(&stored); err != nil {
		t.Fatalf("read stored: %v", err)
	}
	if !strings.HasPrefix(stored, secretCipherPrefix) {
		t.Fatalf("update must keep the password encrypted, got %q", stored)
	}
	got, err := instances.Get(inst.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Password != "hunter2" {
		t.Fatalf("password changed on update: %q", got.Password)
	}
}

func TestInstanceStoreWithoutCipherReadsPlaintext(t *testing.T) {
	db := OpenTestDB(t)
	instances := NewInstanceStore(db)
	inst, err := instances.Create(CreateInstanceInput{
		Name:      "test",
		ServerURL: "https://music.example",
		Username:  "alice",
		Password:  "hunter2",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if inst.Password != "hunter2" {
		t.Fatalf("expected plaintext pass-through without cipher, got %q", inst.Password)
	}
}
