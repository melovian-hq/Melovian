// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"testing"

	"melovian/internal/appconfig"
)

func TestRebindPostgres(t *testing.T) {
	got := rebindPostgres(`INSERT INTO t (a, b) VALUES (?, ?) ON CONFLICT(a) DO UPDATE SET b = excluded.b`)
	want := `INSERT INTO t (a, b) VALUES ($1, $2) ON CONFLICT(a) DO UPDATE SET b = excluded.b`
	if got != want {
		t.Fatalf("rebind = %q, want %q", got, want)
	}
	if rebindPostgres(`SELECT 1`) != `SELECT 1` {
		t.Fatal("expected identity when no placeholders")
	}
}

func TestResolveDialectDefaultSQLite(t *testing.T) {
	d, dsn, err := resolveDialect(appconfig.Config{DatabasePath: "/tmp/x.db"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Name() != DriverSQLite {
		t.Fatalf("driver = %s", d.Name())
	}
	if dsn != "/tmp/x.db" {
		t.Fatalf("dsn = %q", dsn)
	}
}

func TestResolveDialectPostgresURL(t *testing.T) {
	d, dsn, err := resolveDialect(appconfig.Config{
		DatabaseURL: "postgres://user:pass@localhost:5432/melovian",
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.Name() != DriverPostgres {
		t.Fatalf("driver = %s", d.Name())
	}
	if dsn != "postgres://user:pass@localhost:5432/melovian" {
		t.Fatalf("dsn = %q", dsn)
	}
}

func TestPostgresSmoke(t *testing.T) {
	db := OpenTestDBPostgres(t)
	if db.Dialect().Name() != DriverPostgres {
		t.Fatalf("driver = %s", db.Dialect().Name())
	}
	if err := db.setSetting("multi_db_smoke", "ok"); err != nil {
		t.Fatal(err)
	}
	got, err := db.getSetting("multi_db_smoke")
	if err != nil {
		t.Fatal(err)
	}
	if got != "ok" {
		t.Fatalf("got %q", got)
	}
}
