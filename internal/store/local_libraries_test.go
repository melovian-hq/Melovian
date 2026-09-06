// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"database/sql"
	"errors"
	"testing"
)

func TestLocalLibraryCRUD(t *testing.T) {
	db := OpenTestDB(t)
	libraries := NewLocalLibraryStore(db)

	dir := t.TempDir()
	lib, err := libraries.Create(CreateLocalLibraryInput{
		Name: "Downloads",
		Path: dir,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if lib.ID == "" {
		t.Fatal("expected library id")
	}

	items, err := libraries.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 library, got %d", len(items))
	}

	updated, err := libraries.Update(lib.ID, UpdateLocalLibraryInput{Name: "Music"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Music" {
		t.Fatalf("expected name Music, got %q", updated.Name)
	}

	if err := libraries.SetActive(lib.ID); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	activeID, err := libraries.GetActiveID()
	if err != nil {
		t.Fatalf("GetActiveID: %v", err)
	}
	if activeID != lib.ID {
		t.Fatalf("expected active id %q, got %q", lib.ID, activeID)
	}

	if err := libraries.Delete(lib.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = libraries.Get(lib.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestLocalLibraryFindByPathForUser(t *testing.T) {
	db := OpenTestDB(t)
	libraries := NewLocalLibraryStore(db)

	dir := t.TempDir()
	lib, err := libraries.Create(CreateLocalLibraryInput{
		Name: "Main",
		Path: dir,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	found, err := libraries.FindByPathForUser("", dir)
	if err != nil {
		t.Fatalf("FindByPathForUser: %v", err)
	}
	if found.ID != lib.ID {
		t.Fatalf("expected id %q, got %q", lib.ID, found.ID)
	}

	_, err = libraries.FindByPathForUser("", t.TempDir())
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected not found for other path, got %v", err)
	}
}
