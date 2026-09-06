// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type sqliteDialect struct{}

func (sqliteDialect) Name() DriverName { return DriverSQLite }

func (sqliteDialect) Open(dsn string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dsn), 0o750); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}
	conn, err := sql.Open("sqlite", dsn+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	return conn, nil
}

func (sqliteDialect) Setup(db *sql.DB) error {
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA temp_store=MEMORY",
		"PRAGMA mmap_size=67108864",
		"PRAGMA cache_size=-32000",
		"PRAGMA foreign_keys=ON",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("pragma %q: %w", p, err)
		}
	}
	return nil
}

func (sqliteDialect) Rebind(query string) string { return query }

func (sqliteDialect) IsDuplicateColumn(err error) bool {
	return isDuplicateColumnMessage(err)
}

func (sqliteDialect) AutoIncPK(column string) string {
	return column + " INTEGER PRIMARY KEY AUTOINCREMENT"
}

func (sqliteDialect) YearFromUnix(column string) string {
	return "CAST(strftime('%Y', " + column + ", 'unixepoch') AS INTEGER)"
}
