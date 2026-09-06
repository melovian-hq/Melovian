// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type postgresDialect struct{}

func (postgresDialect) Name() DriverName { return DriverPostgres }

func (postgresDialect) Open(dsn string) (*sql.DB, error) {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	return conn, nil
}

func (postgresDialect) Setup(db *sql.DB) error {
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}

func (postgresDialect) Rebind(query string) string {
	return rebindPostgres(query)
}

func (postgresDialect) IsDuplicateColumn(err error) bool {
	return isDuplicateColumnMessage(err)
}

func (postgresDialect) AutoIncPK(column string) string {
	return column + " BIGSERIAL PRIMARY KEY"
}

func (postgresDialect) YearFromUnix(column string) string {
	return "CAST(EXTRACT(YEAR FROM to_timestamp(" + column + ")) AS INTEGER)"
}
