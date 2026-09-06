// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"melovian/internal/appconfig"
)

// DriverName identifies a supported database engine.
type DriverName string

const (
	DriverSQLite   DriverName = "sqlite"
	DriverPostgres DriverName = "postgres"
)

// Dialect abstracts engine-specific open, setup, and SQL helpers.
type Dialect interface {
	Name() DriverName
	Open(dsn string) (*sql.DB, error)
	Setup(db *sql.DB) error
	Rebind(query string) string
	IsDuplicateColumn(err error) bool
	AutoIncPK(column string) string
	YearFromUnix(column string) string
}

func resolveDialect(cfg appconfig.Config) (Dialect, string, error) {
	raw := strings.TrimSpace(cfg.DatabaseURL)
	if raw == "" {
		return sqliteDialect{}, cfg.DatabasePath, nil
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, "", fmt.Errorf("parse database url: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "postgres", "postgresql":
		return postgresDialect{}, raw, nil
	case "sqlite", "file":
		path := u.Path
		if u.Scheme == "file" {
			path = raw
			if after, ok := strings.CutPrefix(path, "file:"); ok {
				path = after
				if after, ok := strings.CutPrefix(path, "//"); ok {
					path = after
				}
			}
		}
		if path == "" {
			path = cfg.DatabasePath
		}
		return sqliteDialect{}, path, nil
	default:
		return nil, "", fmt.Errorf("unsupported database url scheme %q", u.Scheme)
	}
}

func rebindPostgres(query string) string {
	if !strings.Contains(query, "?") {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b.WriteByte('$')
			fmt.Fprintf(&b, "%d", n)
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

func isDuplicateColumnMessage(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column") ||
		strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "duplicate_column")
}
