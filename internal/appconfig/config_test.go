// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package appconfig

import (
	"testing"
)

func TestAuthEnabled(t *testing.T) {
	if (Config{AuthSecret: ""}).AuthEnabled() {
		t.Fatal("empty secret should disable auth")
	}
	if (Config{AuthSecret: "   "}).AuthEnabled() {
		t.Fatal("whitespace secret should disable auth")
	}
	if !(Config{AuthSecret: "secret"}).AuthEnabled() {
		t.Fatal("non-empty secret should enable auth")
	}
}

func TestOIDCEnabled(t *testing.T) {
	base := Config{
		AuthSecret: "secret",
		OIDC: OIDCConfig{
			Issuer:   "https://issuer.example",
			ClientID: "client",
		},
	}
	if !base.OIDCEnabled() {
		t.Fatal("expected oidc enabled with auth secret, issuer, and client id")
	}

	cases := []Config{
		{AuthSecret: "", OIDC: base.OIDC},
		{AuthSecret: "secret", OIDC: OIDCConfig{Issuer: "", ClientID: "client"}},
		{AuthSecret: "secret", OIDC: OIDCConfig{Issuer: "https://issuer.example", ClientID: ""}},
	}
	for i, cfg := range cases {
		if cfg.OIDCEnabled() {
			t.Fatalf("case %d: expected oidc disabled", i)
		}
	}
}

func TestLoadConfigConnectionDefaults(t *testing.T) {
	t.Setenv("MELOVIAN_CONN_MIN_DELAY_MS", "5000")
	t.Setenv("MELOVIAN_CONN_MAX_DELAY_MS", "90000")
	t.Setenv("MELOVIAN_CONN_BACKOFF_MULTIPLIER", "2.0")
	t.Setenv("MELOVIAN_CONN_HEALTH_CHECK_MS", "45000")
	t.Setenv("MELOVIAN_CONN_OFFLINE_POLL_MS", "3000")
	t.Setenv("MELOVIAN_CONN_SELF_HEAL_MS", "120000")
	t.Setenv("MELOVIAN_CONN_MAX_HISTORY", "64")
	t.Setenv("CACHE_ENABLED", "false")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	d := cfg.ConnectionDefaults
	if d.MinDelayMs != 5000 || d.MaxDelayMs != 90000 {
		t.Fatalf("unexpected delay defaults: %+v", d)
	}
	if d.BackoffMultiplier != 2.0 {
		t.Fatalf("expected backoff 2.0, got %v", d.BackoffMultiplier)
	}
	if d.HealthCheckIntervalMs != 45000 || d.OfflinePollMs != 3000 {
		t.Fatalf("unexpected poll defaults: %+v", d)
	}
	if d.SelfHealIntervalMs != 120000 || d.MaxHistoryEntries != 64 {
		t.Fatalf("unexpected heal/history defaults: %+v", d)
	}
	if cfg.CacheEnabled {
		t.Fatal("expected cache disabled when CACHE_ENABLED=false")
	}
}

func TestLoadOIDCScopesDefault(t *testing.T) {
	t.Setenv("MELOVIAN_OIDC_SCOPES", "")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.OIDC.Scopes) != 3 {
		t.Fatalf("expected default scopes, got %v", cfg.OIDC.Scopes)
	}
}

func TestLocalLibraryDefaults(t *testing.T) {
	t.Setenv("MELOVIAN_LOCAL_LIBRARY", "")
	t.Setenv("MELOVIAN_LOCAL_LIBRARY_PATH", "")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.LocalLibrary.Enabled {
		t.Fatal("expected local library enabled by default on desktop")
	}
	if cfg.LocalLibrary.AllowCustomPath != true {
		t.Fatal("expected custom paths when no default path is set")
	}
}

func TestLocalLibraryMultiUserDefaults(t *testing.T) {
	t.Setenv("MELOVIAN_LOCAL_LIBRARY", "")
	t.Setenv("MELOVIAN_LOCAL_LIBRARY_PATH", "/srv/music")
	cfg := Config{AuthSecret: "secret", ServerMode: true}
	effective := cfg.LocalLibraryEffective()
	if effective.Enabled {
		t.Fatal("expected local library disabled by default in multi-user server mode")
	}
	if effective.DefaultPath != "/srv/music" {
		t.Fatalf("expected default path, got %q", effective.DefaultPath)
	}
	if effective.AllowCustomPath {
		t.Fatal("expected custom paths disabled when default path is set in multi-user mode")
	}
}

func TestLocalLibraryExplicitEnable(t *testing.T) {
	t.Setenv("MELOVIAN_LOCAL_LIBRARY", "true")
	cfg := Config{AuthSecret: "secret", ServerMode: true}
	effective := cfg.LocalLibraryEffective()
	if !effective.Enabled {
		t.Fatal("expected explicit enable to override multi-user default")
	}
}

func TestParseAllowedIPs(t *testing.T) {
	prefixes, err := parseAllowedIPs("192.168.1.5,10.0.0.0/8")
	if err != nil {
		t.Fatalf("parseAllowedIPs: %v", err)
	}
	if len(prefixes) != 2 {
		t.Fatalf("expected 2 prefixes, got %d", len(prefixes))
	}
	if prefixes[0].String() != "192.168.1.5/32" {
		t.Fatalf("unexpected first prefix: %s", prefixes[0])
	}
	if prefixes[1].String() != "10.0.0.0/8" {
		t.Fatalf("unexpected second prefix: %s", prefixes[1])
	}
}

func TestParseAllowedIPsEmpty(t *testing.T) {
	prefixes, err := parseAllowedIPs("")
	if err != nil {
		t.Fatalf("parseAllowedIPs: %v", err)
	}
	if len(prefixes) != 0 {
		t.Fatalf("expected empty slice, got %v", prefixes)
	}
}

func TestParseAllowedIPsInvalid(t *testing.T) {
	if _, err := parseAllowedIPs("not-an-ip"); err == nil {
		t.Fatal("expected error for invalid IP")
	}
}

func TestLoadConfigDemoAndAllowlist(t *testing.T) {
	t.Setenv("MELOVIAN_DEMO_MODE", "true")
	t.Setenv("MELOVIAN_ALLOWED_IPS", "127.0.0.1,::1")
	t.Setenv("MELOVIAN_TRUST_PROXY", "yes")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.DemoMode {
		t.Fatal("expected demo mode enabled")
	}
	if !cfg.TrustProxy {
		t.Fatal("expected trust proxy enabled")
	}
	if len(cfg.AllowedIPs) != 2 {
		t.Fatalf("expected 2 allowed IPs, got %d", len(cfg.AllowedIPs))
	}
}

func TestDemoModeEffective(t *testing.T) {
	if (Config{DemoMode: true}).DemoModeEffective() {
		t.Fatal("demo mode should require server mode")
	}
	if !(Config{DemoMode: true, ServerMode: true}).DemoModeEffective() {
		t.Fatal("expected demo effective in server mode")
	}
}

func TestAuthDisabledInDemoMode(t *testing.T) {
	cfg := Config{AuthSecret: "secret", DemoMode: true, ServerMode: true}
	if cfg.AuthEnabled() {
		t.Fatal("auth should be disabled in demo mode")
	}
}

func TestValidateServerConfigDemo(t *testing.T) {
	cfg := Config{
		DemoMode:     true,
		ServerMode:   true,
		LegacyServer: "https://music.example",
		LegacyUser:   "demo",
		LegacyPass:   "pass",
	}
	if err := ValidateServerConfig(cfg); err != nil {
		t.Fatalf("ValidateServerConfig: %v", err)
	}

	cfg.LegacyServer = ""
	if err := ValidateServerConfig(cfg); err != nil {
		t.Fatalf("empty server should allow built-in fake catalog: %v", err)
	}

	cfg.LegacyServer = "fake://melovian-demo"
	if err := ValidateServerConfig(cfg); err != nil {
		t.Fatalf("fake catalog URL should validate: %v", err)
	}

	cfg.LegacyServer = "https://music.example"
	cfg.LegacyUser = ""
	if err := ValidateServerConfig(cfg); err == nil {
		t.Fatal("expected error when real server lacks NAVIDROME_USER")
	}
}

func TestValidateServerConfigAllowsNoAuth(t *testing.T) {
	cfg := Config{ServerMode: true}
	if err := ValidateServerConfig(cfg); err != nil {
		t.Fatalf("ValidateServerConfig: %v", err)
	}
}

func TestGenerateAuthSecret(t *testing.T) {
	secret, err := GenerateAuthSecret()
	if err != nil {
		t.Fatalf("GenerateAuthSecret: %v", err)
	}
	if len(secret) != 64 {
		t.Fatalf("expected 64-char hex secret, got len %d", len(secret))
	}
	other, err := GenerateAuthSecret()
	if err != nil {
		t.Fatalf("GenerateAuthSecret: %v", err)
	}
	if secret == other {
		t.Fatal("expected unique secrets")
	}
}

func TestPrepareServerAuthGeneratesSecret(t *testing.T) {
	cfg := Config{ServerMode: true}
	res, err := PrepareServerAuth(&cfg, ServerStartupOptions{})
	if err != nil {
		t.Fatalf("PrepareServerAuth: %v", err)
	}
	if !res.Generated || res.GeneratedSecret == "" {
		t.Fatal("expected generated secret")
	}
	if cfg.AuthSecret != res.GeneratedSecret {
		t.Fatal("expected config auth secret to match generated value")
	}
	if !cfg.AuthEnabled() {
		t.Fatal("expected auth enabled after secret generation")
	}
}

func TestPrepareServerAuthNoAuth(t *testing.T) {
	cfg := Config{ServerMode: true, AuthSecret: "existing"}
	res, err := PrepareServerAuth(&cfg, ServerStartupOptions{NoAuth: true})
	if err != nil {
		t.Fatalf("PrepareServerAuth: %v", err)
	}
	if !res.NoAuth || !res.IgnoredSecret {
		t.Fatalf("expected no-auth with ignored secret, got %+v", res)
	}
	if cfg.AuthEnabled() {
		t.Fatal("expected auth disabled")
	}
}

func TestPrepareServerAuthNoAuthEnv(t *testing.T) {
	t.Setenv("MELOVIAN_NO_AUTH", "true")
	cfg := Config{ServerMode: true}
	res, err := PrepareServerAuth(&cfg, ServerStartupOptions{})
	if err != nil {
		t.Fatalf("PrepareServerAuth: %v", err)
	}
	if !res.NoAuth {
		t.Fatal("expected no-auth from env")
	}
}

func TestPrepareServerAuthKeepsProvidedSecret(t *testing.T) {
	cfg := Config{ServerMode: true, AuthSecret: "provided-secret"}
	res, err := PrepareServerAuth(&cfg, ServerStartupOptions{})
	if err != nil {
		t.Fatalf("PrepareServerAuth: %v", err)
	}
	if res.Generated {
		t.Fatal("expected no generation when secret is provided")
	}
	if cfg.AuthSecret != "provided-secret" {
		t.Fatalf("expected provided secret, got %q", cfg.AuthSecret)
	}
}

func TestPrepareServerAuthSkipsDemoMode(t *testing.T) {
	cfg := Config{ServerMode: true, DemoMode: true}
	res, err := PrepareServerAuth(&cfg, ServerStartupOptions{})
	if err != nil {
		t.Fatalf("PrepareServerAuth: %v", err)
	}
	if res.Generated || cfg.AuthSecret != "" {
		t.Fatalf("expected no auth setup in demo mode, got %+v secret=%q", res, cfg.AuthSecret)
	}
}

func TestDefaultDataDirUsesMelovianDataEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MELOVIAN_DATA", dir)
	if got := DefaultDataDir(); got != dir {
		t.Fatalf("DefaultDataDir() = %q, want %q", got, dir)
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DataDir != dir {
		t.Fatalf("LoadConfig DataDir = %q, want %q", cfg.DataDir, dir)
	}
}
