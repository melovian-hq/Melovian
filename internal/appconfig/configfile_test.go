// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package appconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"
)

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

// clearConfigEnv forces each env var to empty for the test and restores the
// original value on cleanup, so values LoadConfigFile sets cannot leak into
// other tests.
func clearConfigEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		t.Setenv(key, "")
	}
}

// isolateXDG points the xdg package at empty temp dirs so a developer
// machine's real ~/.config/melovian/config.toml cannot leak into tests.
// The xdg package reads its paths at init, so Reload is required.
func isolateXDG(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	xdg.Reload()
	t.Cleanup(xdg.Reload)
}

func TestLoadConfigFileAppliesValues(t *testing.T) {
	clearConfigEnv(t, "MELOVIAN_LISTEN", "MELOVIAN_DATA", "MELOVIAN_PUBLIC_URL", "MELOVIAN_AUTH_SECRET")
	path := writeConfigFile(t, `
listen = "0.0.0.0:9999"
data_dir = "/srv/melovian"
public_url = "https://music.example.com"
auth_secret = "abc123"
`)
	if err := LoadConfigFile(path); err != nil {
		t.Fatalf("LoadConfigFile: %v", err)
	}
	if got := os.Getenv("MELOVIAN_LISTEN"); got != "0.0.0.0:9999" {
		t.Fatalf("listen = %q", got)
	}
	if got := os.Getenv("MELOVIAN_DATA"); got != "/srv/melovian" {
		t.Fatalf("data dir = %q", got)
	}
	if got := os.Getenv("MELOVIAN_PUBLIC_URL"); got != "https://music.example.com" {
		t.Fatalf("public url = %q", got)
	}
	if got := os.Getenv("MELOVIAN_AUTH_SECRET"); got != "abc123" {
		t.Fatalf("auth secret = %q", got)
	}
}

func TestLoadConfigFileEnvWins(t *testing.T) {
	clearConfigEnv(t, "MELOVIAN_DATA")
	t.Setenv("MELOVIAN_LISTEN", "127.0.0.1:1111")
	path := writeConfigFile(t, `
listen = "0.0.0.0:9999"
data_dir = "/srv/melovian"
`)
	if err := LoadConfigFile(path); err != nil {
		t.Fatalf("LoadConfigFile: %v", err)
	}
	if got := os.Getenv("MELOVIAN_LISTEN"); got != "127.0.0.1:1111" {
		t.Fatalf("expected existing env to win, got %q", got)
	}
	if got := os.Getenv("MELOVIAN_DATA"); got != "/srv/melovian" {
		t.Fatalf("data dir = %q", got)
	}
}

func TestLoadConfigFileScalarRendering(t *testing.T) {
	clearConfigEnv(t,
		"CACHE_ENABLED", "MELOVIAN_DEMO_MODE", "MELOVIAN_DLNA_SERVER", "MELOVIAN_DLNA_PORT",
		"MELOVIAN_CONN_MIN_DELAY_MS", "MELOVIAN_CONN_BACKOFF_MULTIPLIER",
		"MELOVIAN_SENTRY_TRACES_SAMPLE_RATE",
	)
	path := writeConfigFile(t, `
cache_enabled = false
demo_mode = true

[dlna]
enabled = true
port = 8201

[connection]
min_delay_ms = 500
backoff_multiplier = 1.8

[sentry]
traces_sample_rate = 0.5
`)
	if err := LoadConfigFile(path); err != nil {
		t.Fatalf("LoadConfigFile: %v", err)
	}
	want := map[string]string{
		"CACHE_ENABLED":                      "false",
		"MELOVIAN_DEMO_MODE":                 "true",
		"MELOVIAN_DLNA_SERVER":               "true",
		"MELOVIAN_DLNA_PORT":                 "8201",
		"MELOVIAN_CONN_MIN_DELAY_MS":         "500",
		"MELOVIAN_CONN_BACKOFF_MULTIPLIER":   "1.8",
		"MELOVIAN_SENTRY_TRACES_SAMPLE_RATE": "0.5",
	}
	for env, expected := range want {
		if got := os.Getenv(env); got != expected {
			t.Fatalf("%s = %q, want %q", env, got, expected)
		}
	}
}

func TestLoadConfigFileNestedTables(t *testing.T) {
	clearConfigEnv(t,
		"NAVIDROME_SERVER", "NAVIDROME_USER", "NAVIDROME_PASSWORD",
		"MELOVIAN_OIDC_ISSUER", "MELOVIAN_OIDC_PROVIDER_NAME", "MELOVIAN_OIDC_AUTH_URL",
		"MELOVIAN_LOCAL_LIBRARY", "MELOVIAN_LOCAL_LIBRARY_PATH",
	)
	path := writeConfigFile(t, `
[subsonic]
server_url = "https://navidrome.example.com"
username = "demo"
password = "secret"

[oidc]
issuer = "https://id.example.com"
provider_name = "Pocket ID"

[local_library]
enabled = true
path = "/srv/music"
`)
	if err := LoadConfigFile(path); err != nil {
		t.Fatalf("LoadConfigFile: %v", err)
	}
	if got := os.Getenv("NAVIDROME_SERVER"); got != "https://navidrome.example.com" {
		t.Fatalf("subsonic server = %q", got)
	}
	if got := os.Getenv("NAVIDROME_USER"); got != "demo" {
		t.Fatalf("subsonic user = %q", got)
	}
	if got := os.Getenv("MELOVIAN_OIDC_ISSUER"); got != "https://id.example.com" {
		t.Fatalf("oidc issuer = %q", got)
	}
	if got := os.Getenv("MELOVIAN_LOCAL_LIBRARY"); got != "true" {
		t.Fatalf("local library = %q", got)
	}
	if got := os.Getenv("MELOVIAN_LOCAL_LIBRARY_PATH"); got != "/srv/music" {
		t.Fatalf("local library path = %q", got)
	}
}

func TestLoadConfigFileArrays(t *testing.T) {
	clearConfigEnv(t,
		"MELOVIAN_CORS_ORIGINS", "MELOVIAN_OIDC_SCOPES", "MELOVIAN_LOCAL_LIBRARY_ROOTS",
	)
	path := writeConfigFile(t, `
cors_origins = ["https://a.example", "https://b.example"]

[oidc]
scopes = ["openid", "profile"]

[local_library]
roots = ["/music", "/mnt/media"]
`)
	if err := LoadConfigFile(path); err != nil {
		t.Fatalf("LoadConfigFile: %v", err)
	}
	if got := os.Getenv("MELOVIAN_CORS_ORIGINS"); got != "https://a.example,https://b.example" {
		t.Fatalf("cors origins = %q", got)
	}
	if got := os.Getenv("MELOVIAN_OIDC_SCOPES"); got != "openid profile" {
		t.Fatalf("oidc scopes = %q", got)
	}
	wantRoots := "/music" + string(filepath.ListSeparator) + "/mnt/media"
	if got := os.Getenv("MELOVIAN_LOCAL_LIBRARY_ROOTS"); got != wantRoots {
		t.Fatalf("local library roots = %q, want %q", got, wantRoots)
	}
}

func TestLoadConfigFileStringListPassesThrough(t *testing.T) {
	clearConfigEnv(t, "MELOVIAN_CORS_ORIGINS", "MELOVIAN_OIDC_SCOPES", "MELOVIAN_LOCAL_LIBRARY_ROOTS")
	path := writeConfigFile(t, `
cors_origins = "https://a.example, https://b.example"

[oidc]
scopes = "openid profile email"

[local_library]
roots = "/music:/mnt/media"
`)
	if err := LoadConfigFile(path); err != nil {
		t.Fatalf("LoadConfigFile: %v", err)
	}
	if got := os.Getenv("MELOVIAN_CORS_ORIGINS"); got != "https://a.example, https://b.example" {
		t.Fatalf("cors origins = %q", got)
	}
	if got := os.Getenv("MELOVIAN_OIDC_SCOPES"); got != "openid profile email" {
		t.Fatalf("oidc scopes = %q", got)
	}
	if got := os.Getenv("MELOVIAN_LOCAL_LIBRARY_ROOTS"); got != "/music:/mnt/media" {
		t.Fatalf("local library roots = %q", got)
	}
}

func TestLoadConfigFileUnknownKey(t *testing.T) {
	path := writeConfigFile(t, "listne = \"0.0.0.0:9999\"\n")
	err := LoadConfigFile(path)
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), "listne") {
		t.Fatalf("error should name the key, got %v", err)
	}
}

func TestLoadConfigFileUnknownNestedKey(t *testing.T) {
	path := writeConfigFile(t, "[bogus]\nkey = 1\n")
	err := LoadConfigFile(path)
	if err == nil {
		t.Fatal("expected error for unknown table")
	}
	if !strings.Contains(err.Error(), "bogus.key") {
		t.Fatalf("error should name the dotted key, got %v", err)
	}
}

func TestLoadConfigFileUnknownEmptyTable(t *testing.T) {
	path := writeConfigFile(t, "listen = \"0.0.0.0:9999\"\n\n[bogus]\n")
	err := LoadConfigFile(path)
	if err == nil {
		t.Fatal("expected error for unknown empty table")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Fatalf("error should name the table, got %v", err)
	}
}

func TestLoadConfigFileWrongValueType(t *testing.T) {
	path := writeConfigFile(t, "listen = 8080\n")
	err := LoadConfigFile(path)
	if err == nil {
		t.Fatal("expected error for non-string listen")
	}
	if !strings.Contains(err.Error(), "listen") {
		t.Fatalf("error should name the key, got %v", err)
	}
}

func TestLoadConfigFileWrongBoolType(t *testing.T) {
	path := writeConfigFile(t, "demo_mode = \"yes\"\n")
	if err := LoadConfigFile(path); err == nil {
		t.Fatal("expected error for string boolean")
	}
}

func TestLoadConfigFileUnsupportedValueType(t *testing.T) {
	path := writeConfigFile(t, "listen = 1979-05-27\n")
	if err := LoadConfigFile(path); err == nil {
		t.Fatal("expected error for datetime value")
	}
}

func TestLoadConfigFileNonStringArray(t *testing.T) {
	path := writeConfigFile(t, "cors_origins = [1, 2]\n")
	if err := LoadConfigFile(path); err == nil {
		t.Fatal("expected error for non-string array")
	}
}

func TestLoadConfigFileMissingFile(t *testing.T) {
	if err := LoadConfigFile(filepath.Join(t.TempDir(), "missing.toml")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestResolveConfigPathExplicit(t *testing.T) {
	path := writeConfigFile(t, "listen = \"0.0.0.0:9999\"\n")
	got, ok := ResolveConfigPath(path)
	if !ok || got != path {
		t.Fatalf("ResolveConfigPath = %q, %v; want %q, true", got, ok, path)
	}
}

func TestResolveConfigPathExplicitMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.toml")
	if _, ok := ResolveConfigPath(missing); ok {
		t.Fatal("expected not-found for missing explicit path")
	}
	if _, err := LoadConfigFileIfResolved(missing); err == nil {
		t.Fatal("expected error for missing explicit path")
	}
}

func TestResolveConfigPathSearchOrder(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"melovian.toml", "config.toml"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(""), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	// Keep the data dir and XDG candidates empty so they cannot shadow the
	// local files.
	t.Setenv("MELOVIAN_DATA", filepath.Join(t.TempDir(), "data"))
	isolateXDG(t)
	t.Chdir(dir)

	got, ok := ResolveConfigPath("")
	if !ok || got != "melovian.toml" {
		t.Fatalf("expected melovian.toml first, got %q, %v", got, ok)
	}
	if err := os.Remove(filepath.Join(dir, "melovian.toml")); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	got, ok = ResolveConfigPath("")
	if !ok || got != "config.toml" {
		t.Fatalf("expected config.toml second, got %q, %v", got, ok)
	}
}

func TestLoadConfigFileIfResolvedEnvOverride(t *testing.T) {
	clearConfigEnv(t, "MELOVIAN_LISTEN")
	path := writeConfigFile(t, "listen = \"0.0.0.0:7777\"\n")
	t.Setenv("MELOVIAN_CONFIG", path)

	loaded, err := LoadConfigFileIfResolved("")
	if err != nil {
		t.Fatalf("LoadConfigFileIfResolved: %v", err)
	}
	if loaded != path {
		t.Fatalf("loaded = %q, want %q", loaded, path)
	}
	if got := os.Getenv("MELOVIAN_LISTEN"); got != "0.0.0.0:7777" {
		t.Fatalf("listen = %q", got)
	}
}

func TestLoadConfigFileIfResolvedNone(t *testing.T) {
	t.Setenv("MELOVIAN_CONFIG", "")
	t.Setenv("MELOVIAN_DATA", filepath.Join(t.TempDir(), "data"))
	isolateXDG(t)
	t.Chdir(t.TempDir())

	loaded, err := LoadConfigFileIfResolved("")
	if err != nil {
		t.Fatalf("LoadConfigFileIfResolved: %v", err)
	}
	if loaded != "" {
		t.Fatalf("expected no config file, got %q", loaded)
	}
}
