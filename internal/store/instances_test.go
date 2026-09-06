// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"melovian/internal/appconfig"
)

func TestInstanceCRUD(t *testing.T) {
	db := OpenTestDB(t)
	store := NewInstanceStore(db)

	inst, err := store.Create(CreateInstanceInput{
		Name:      "Home",
		ServerURL: "http://music.example.com",
		Username:  "user",
		Password:  "secret",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if inst.ID == "" {
		t.Fatal("expected instance id")
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(items))
	}

	updated, err := store.Update(inst.ID, UpdateInstanceInput{Name: "Office"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Office" {
		t.Fatalf("expected name Office, got %q", updated.Name)
	}

	if err := store.SetActive(inst.ID); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	activeID, err := store.GetActiveID()
	if err != nil {
		t.Fatalf("GetActiveID: %v", err)
	}
	if activeID != inst.ID {
		t.Fatalf("expected active id %q, got %q", inst.ID, activeID)
	}

	active, err := store.GetActive()
	if err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	if active.ID != inst.ID {
		t.Fatalf("GetActive returned wrong instance")
	}

	if err := store.Delete(inst.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = store.Get(inst.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestLegacyMigrationFromSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.db")

	db, err := OpenDB(path, appconfig.Config{})
	if err != nil {
		t.Fatalf("first OpenDB: %v", err)
	}
	if err := db.setSetting(appconfig.SettingNavidromeServer, "http://legacy.example.com"); err != nil {
		t.Fatalf("set server: %v", err)
	}
	if err := db.setSetting(appconfig.SettingNavidromeUser, "legacyuser"); err != nil {
		t.Fatalf("set user: %v", err)
	}
	if err := db.setSetting(appconfig.SettingNavidromePassword, "legacypass"); err != nil {
		t.Fatalf("set password: %v", err)
	}
	_ = db.Close()

	db, err = OpenDB(path, appconfig.Config{})
	if err != nil {
		t.Fatalf("second OpenDB: %v", err)
	}
	defer func() { _ = db.Close() }()

	store := NewInstanceStore(db)
	items, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 migrated instance, got %d", len(items))
	}
	if items[0].ServerURL != "http://legacy.example.com" {
		t.Fatalf("unexpected server url: %q", items[0].ServerURL)
	}

	activeID, err := store.GetActiveID()
	if err != nil {
		t.Fatalf("GetActiveID: %v", err)
	}
	if activeID != items[0].ID {
		t.Fatalf("expected migrated instance to be active")
	}
}
