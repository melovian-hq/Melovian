// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"melovian/internal/appconfig"
)

func OpenTestDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := OpenDB(path, appconfig.Config{})
	if err != nil {
		t.Fatalf("OpenTestDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// OpenTestDBPostgres opens a Postgres database when MELOVIAN_TEST_DATABASE_URL is set.
// Skips the test otherwise.
func OpenTestDBPostgres(t *testing.T) *DB {
	t.Helper()
	url := strings.TrimSpace(os.Getenv("MELOVIAN_TEST_DATABASE_URL"))
	if url == "" {
		t.Skip("MELOVIAN_TEST_DATABASE_URL not set")
	}
	db, err := Open(appconfig.Config{DatabaseURL: url})
	if err != nil {
		t.Fatalf("OpenTestDBPostgres: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
