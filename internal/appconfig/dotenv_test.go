// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package appconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("MELOVIAN_LISTEN=127.0.0.1:9999\n# comment\n\nQUOTED=\"hello world\"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	loaded, err := LoadDotEnv(path)
	if err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	if !loaded {
		t.Fatal("expected loaded=true")
	}
	if got := os.Getenv("MELOVIAN_LISTEN"); got != "127.0.0.1:9999" {
		t.Fatalf("expected listen from dotenv, got %q", got)
	}
	if got := os.Getenv("QUOTED"); got != "hello world" {
		t.Fatalf("expected unquoted value, got %q", got)
	}
}

func TestLoadDotEnvDoesNotOverrideExistingEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("MELOVIAN_LISTEN=127.0.0.1:1111\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv("MELOVIAN_LISTEN", "127.0.0.1:2222")

	if _, err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	if got := os.Getenv("MELOVIAN_LISTEN"); got != "127.0.0.1:2222" {
		t.Fatalf("expected existing env to win, got %q", got)
	}
}

func TestLoadDotEnvMissingFile(t *testing.T) {
	loaded, err := LoadDotEnv(filepath.Join(t.TempDir(), "missing.env"))
	if err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	if loaded {
		t.Fatal("expected loaded=false for missing file")
	}
}

func TestServerCLIApplyHostPort(t *testing.T) {
	cli := NewServerCLI()
	if err := cli.Parse([]string{"--host", "127.0.0.1", "--port", "9090"}); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	cfg := Config{ListenAddr: "127.0.0.1:17337"}
	if err := cli.Apply(&cfg); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if cfg.ListenAddr != "127.0.0.1:9090" {
		t.Fatalf("expected host/port listen addr, got %q", cfg.ListenAddr)
	}
}

func TestServerCLIApplyListenOverride(t *testing.T) {
	cli := NewServerCLI()
	if err := cli.Parse([]string{"--listen", "0.0.0.0:3000", "--host", "127.0.0.1", "--port", "9090"}); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	cfg := Config{}
	if err := cli.Apply(&cfg); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if cfg.ListenAddr != "0.0.0.0:3000" {
		t.Fatalf("expected --listen to win, got %q", cfg.ListenAddr)
	}
}

func TestServerCLIApplyDefaultServerListen(t *testing.T) {
	cli := NewServerCLI()
	if err := cli.Parse(nil); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	cfg := Config{ListenAddr: "127.0.0.1:17337"}
	if err := cli.Apply(&cfg); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if cfg.ListenAddr != "0.0.0.0:8080" {
		t.Fatalf("expected default server listen, got %q", cfg.ListenAddr)
	}
}

func TestServerCLILocalLibraryOverride(t *testing.T) {
	cli := NewServerCLI()
	if err := cli.Parse([]string{"--local-library", "--local-library-path", "/music"}); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	cfg := Config{ServerMode: true, AuthSecret: "secret"}
	if err := cli.Apply(&cfg); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	effective := cfg.LocalLibraryEffective()
	if !effective.Enabled {
		t.Fatal("expected local library enabled from flag")
	}
	if effective.DefaultPath != "/music" {
		t.Fatalf("expected default path /music, got %q", effective.DefaultPath)
	}
	if effective.AllowCustomPath {
		t.Fatal("expected custom paths disabled when default path is set")
	}
}
