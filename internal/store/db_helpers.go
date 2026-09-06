// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"database/sql"
	"strings"
)

func (db *DB) exec(query string, args ...any) (sql.Result, error) {
	return db.sql.Exec(db.dialect.Rebind(query), args...)
}

// execScript runs DDL that may contain multiple statements (Postgres needs one at a time).
func (db *DB) execScript(script string) error {
	for _, stmt := range splitSQLStatements(script) {
		if _, err := db.exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func splitSQLStatements(script string) []string {
	parts := strings.Split(script, ";")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		stmt := strings.TrimSpace(part)
		if stmt == "" {
			continue
		}
		out = append(out, stmt)
	}
	return out
}

func (db *DB) query(query string, args ...any) (*sql.Rows, error) {
	return db.sql.Query(db.dialect.Rebind(query), args...)
}

func (db *DB) queryRow(query string, args ...any) *sql.Row {
	return db.sql.QueryRow(db.dialect.Rebind(query), args...)
}

func (db *DB) begin() (*Tx, error) {
	tx, err := db.sql.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx, dialect: db.dialect}, nil
}

// Tx wraps sql.Tx with dialect rebinding.
type Tx struct {
	tx      *sql.Tx
	dialect Dialect
}

func (tx *Tx) Exec(query string, args ...any) (sql.Result, error) {
	return tx.tx.Exec(tx.dialect.Rebind(query), args...)
}

func (tx *Tx) Query(query string, args ...any) (*sql.Rows, error) {
	return tx.tx.Query(tx.dialect.Rebind(query), args...)
}

func (tx *Tx) QueryRow(query string, args ...any) *sql.Row {
	return tx.tx.QueryRow(tx.dialect.Rebind(query), args...)
}

func (tx *Tx) Commit() error { return tx.tx.Commit() }

func (tx *Tx) Rollback() error { return tx.tx.Rollback() }
