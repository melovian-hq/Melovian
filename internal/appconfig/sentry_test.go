// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package appconfig

import (
	"encoding/json"
	"testing"
)

func TestLoadSentryConfig(t *testing.T) {
	t.Setenv("MELOVIAN_SENTRY_DSN", "")
	t.Setenv("SENTRY_DSN", "")
	t.Setenv("MELOVIAN_SENTRY_FRONTEND_DSN", "")
	t.Setenv("SENTRY_FRONTEND_DSN", "")
	t.Setenv("MELOVIAN_SENTRY_ENVIRONMENT", "")
	t.Setenv("SENTRY_ENVIRONMENT", "")
	t.Setenv("MELOVIAN_SENTRY_RELEASE", "")
	t.Setenv("SENTRY_RELEASE", "")
	t.Setenv("MELOVIAN_SENTRY_TRACES_SAMPLE_RATE", "")
	t.Setenv("SENTRY_TRACES_SAMPLE_RATE", "")

	cfg := loadSentryConfig()
	if cfg.Enabled() {
		t.Fatal("expected disabled without DSN")
	}
	if cfg.Release != defaultSentryRelease {
		t.Fatalf("release %q, want %q", cfg.Release, defaultSentryRelease)
	}
}

func TestLoadSentryConfigEnv(t *testing.T) {
	t.Setenv("MELOVIAN_SENTRY_DSN", "https://glitchtip.example/1")
	t.Setenv("SENTRY_DSN", "")
	t.Setenv("MELOVIAN_SENTRY_FRONTEND_DSN", "https://glitchtip.example/2")
	t.Setenv("SENTRY_FRONTEND_DSN", "")
	t.Setenv("MELOVIAN_SENTRY_ENVIRONMENT", "staging")
	t.Setenv("SENTRY_ENVIRONMENT", "")
	t.Setenv("MELOVIAN_SENTRY_RELEASE", "melovian@test")
	t.Setenv("SENTRY_RELEASE", "")
	t.Setenv("MELOVIAN_SENTRY_TRACES_SAMPLE_RATE", "0.25")
	t.Setenv("SENTRY_TRACES_SAMPLE_RATE", "")

	cfg := loadSentryConfig()
	if !cfg.Enabled() {
		t.Fatal("expected enabled")
	}
	if cfg.DSN != "https://glitchtip.example/1" {
		t.Fatalf("dsn %q", cfg.DSN)
	}
	if cfg.FrontendDSNEffective() != "https://glitchtip.example/2" {
		t.Fatalf("frontend dsn %q", cfg.FrontendDSNEffective())
	}
	if cfg.Environment != "staging" {
		t.Fatalf("environment %q", cfg.Environment)
	}
	if cfg.Release != "melovian@test" {
		t.Fatalf("release %q", cfg.Release)
	}
	if cfg.TracesSampleRate != 0.25 {
		t.Fatalf("traces sample rate %v", cfg.TracesSampleRate)
	}
}

func TestLoadSentryConfigFallbackDSN(t *testing.T) {
	t.Setenv("MELOVIAN_SENTRY_DSN", "")
	t.Setenv("SENTRY_DSN", "https://sentry.example/3")
	t.Setenv("MELOVIAN_SENTRY_FRONTEND_DSN", "")
	t.Setenv("SENTRY_FRONTEND_DSN", "")

	cfg := loadSentryConfig()
	if cfg.DSN != "https://sentry.example/3" {
		t.Fatalf("dsn %q", cfg.DSN)
	}
	if cfg.FrontendDSNEffective() != "https://sentry.example/3" {
		t.Fatalf("frontend dsn %q", cfg.FrontendDSNEffective())
	}
}

func TestMergeStoredSentrySettingsMigratesFrontendDSN(t *testing.T) {
	raw := json.RawMessage(`{
		"enabled": true,
		"dsn": "",
		"frontendDsn": "https://abc123@glitchtip.example/2",
		"clientReportingAllowed": true
	}`)
	stored, err := MergeStoredSentrySettings(raw)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if stored.DSN != "https://abc123@glitchtip.example/2" {
		t.Fatalf("dsn %q", stored.DSN)
	}
	if stored.FrontendDSN != "" {
		t.Fatalf("frontend dsn not migrated: %q", stored.FrontendDSN)
	}
}

func TestMergeStoredSentrySettingsKeepsBackendDSN(t *testing.T) {
	raw := json.RawMessage(`{
		"enabled": true,
		"dsn": "https://backend@glitchtip.example/1",
		"frontendDsn": "https://frontend@glitchtip.example/2"
	}`)
	stored, err := MergeStoredSentrySettings(raw)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if stored.DSN != "https://backend@glitchtip.example/1" {
		t.Fatalf("dsn %q", stored.DSN)
	}
	if stored.FrontendDSN != "" {
		t.Fatalf("frontend dsn not cleared: %q", stored.FrontendDSN)
	}
}

func TestFrontendDSNOrDefault(t *testing.T) {
	cfg := SentryConfig{}
	if cfg.FrontendDSNOrDefault() == "" {
		t.Fatal("expected the built-in telemetry DSN when nothing configured")
	}
	cfg.DSN = "https://configured@glitchtip.example/1"
	if cfg.FrontendDSNOrDefault() != "https://configured@glitchtip.example/1" {
		t.Fatalf("expected configured dsn, got %q", cfg.FrontendDSNOrDefault())
	}
}

func TestMergeSentryConfigKeepsClientPolicyWithoutBackendDSN(t *testing.T) {
	stored := DefaultStoredSentrySettings()
	merged := MergeSentryConfig(SentryConfig{}, stored, SentryEnvLocks{})
	if !merged.ClientReportingAllowed {
		t.Fatal("client reporting policy must survive a disabled backend DSN")
	}
}

func TestFirstEnv(t *testing.T) {
	t.Setenv("MELOVIAN_SENTRY_DSN", "first")
	t.Setenv("SENTRY_DSN", "second")
	if got := firstEnv("MELOVIAN_SENTRY_DSN", "SENTRY_DSN"); got != "first" {
		t.Fatalf("got %q", got)
	}
	t.Setenv("MELOVIAN_SENTRY_DSN", "")
	if got := firstEnv("MELOVIAN_SENTRY_DSN", "SENTRY_DSN"); got != "second" {
		t.Fatalf("got %q", got)
	}
}
