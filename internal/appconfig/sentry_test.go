// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package appconfig

import (
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
