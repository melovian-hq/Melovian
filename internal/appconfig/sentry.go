// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package appconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"melovian/internal/brand"
	"melovian/internal/compat"
)

// defaultSentryRelease follows the build version so crash reports group by release.
var defaultSentryRelease = brand.Slug + "@" + compat.Version

const SettingSentryServer = "sentry_server_settings"

type StoredSentrySettings struct {
	Enabled                bool    `json:"enabled"`
	DSN                    string  `json:"dsn,omitempty"`
	FrontendDSN            string  `json:"frontendDsn,omitempty"`
	Environment            string  `json:"environment,omitempty"`
	Release                string  `json:"release,omitempty"`
	TracesSampleRate       float64 `json:"tracesSampleRate,omitempty"`
	ClientReportingAllowed bool    `json:"clientReportingAllowed"`
}

func DefaultStoredSentrySettings() StoredSentrySettings {
	return StoredSentrySettings{
		ClientReportingAllowed: true,
	}
}

type SentryEnvLocks struct {
	DSN              bool `json:"dsn"`
	FrontendDSN      bool `json:"frontendDsn"`
	Environment      bool `json:"environment"`
	Release          bool `json:"release"`
	TracesSampleRate bool `json:"tracesSampleRate"`
}

type SentryConfig struct {
	DSN                    string
	FrontendDSN            string
	Environment            string
	Release                string
	TracesSampleRate       float64
	ClientReportingAllowed bool
}

func (c SentryConfig) Enabled() bool {
	return strings.TrimSpace(c.DSN) != ""
}

func (c SentryConfig) FrontendDSNEffective() string {
	if d := strings.TrimSpace(c.FrontendDSN); d != "" {
		return d
	}
	return strings.TrimSpace(c.DSN)
}

func LoadSentryEnvLocks() SentryEnvLocks {
	return SentryEnvLocks{
		DSN:              firstEnv("MELOVIAN_SENTRY_DSN", "SENTRY_DSN") != "",
		FrontendDSN:      firstEnv("MELOVIAN_SENTRY_FRONTEND_DSN", "SENTRY_FRONTEND_DSN") != "",
		Environment:      firstEnv("MELOVIAN_SENTRY_ENVIRONMENT", "SENTRY_ENVIRONMENT") != "",
		Release:          firstEnv("MELOVIAN_SENTRY_RELEASE", "SENTRY_RELEASE") != "",
		TracesSampleRate: envFloatFirst([]string{"MELOVIAN_SENTRY_TRACES_SAMPLE_RATE", "SENTRY_TRACES_SAMPLE_RATE"}, -1) >= 0,
	}
}

func loadSentryConfigFromEnv() SentryConfig {
	dsn := firstEnv("MELOVIAN_SENTRY_DSN", "SENTRY_DSN")
	frontendDSN := firstEnv("MELOVIAN_SENTRY_FRONTEND_DSN", "SENTRY_FRONTEND_DSN")
	environment := firstEnv("MELOVIAN_SENTRY_ENVIRONMENT", "SENTRY_ENVIRONMENT")
	release := firstEnv("MELOVIAN_SENTRY_RELEASE", "SENTRY_RELEASE")
	if release == "" {
		release = defaultSentryRelease
	}
	tracesSampleRate := envFloatFirst(
		[]string{"MELOVIAN_SENTRY_TRACES_SAMPLE_RATE", "SENTRY_TRACES_SAMPLE_RATE"},
		-1,
	)
	if tracesSampleRate < 0 {
		tracesSampleRate = 0
	}
	return SentryConfig{
		DSN:                    dsn,
		FrontendDSN:            frontendDSN,
		Environment:            environment,
		Release:                release,
		TracesSampleRate:       tracesSampleRate,
		ClientReportingAllowed: true,
	}
}

func MergeStoredSentrySettings(raw json.RawMessage) (StoredSentrySettings, error) {
	settings := DefaultStoredSentrySettings()
	if len(raw) == 0 {
		return settings, nil
	}
	if err := json.Unmarshal(raw, &settings); err != nil {
		return StoredSentrySettings{}, err
	}
	return settings, nil
}

func MergeSentryConfig(env SentryConfig, stored StoredSentrySettings, envLocks SentryEnvLocks) SentryConfig {
	merged := SentryConfig{
		ClientReportingAllowed: stored.ClientReportingAllowed,
	}

	if envLocks.DSN {
		merged.DSN = env.DSN
	} else if stored.Enabled {
		merged.DSN = strings.TrimSpace(stored.DSN)
	}

	if envLocks.FrontendDSN {
		merged.FrontendDSN = env.FrontendDSN
	} else {
		merged.FrontendDSN = strings.TrimSpace(stored.FrontendDSN)
	}

	if envLocks.Environment {
		merged.Environment = env.Environment
	} else {
		merged.Environment = strings.TrimSpace(stored.Environment)
	}

	if envLocks.Release {
		merged.Release = env.Release
	} else if stored.Release != "" {
		merged.Release = strings.TrimSpace(stored.Release)
	} else {
		merged.Release = defaultSentryRelease
	}

	if envLocks.TracesSampleRate {
		merged.TracesSampleRate = env.TracesSampleRate
	} else {
		merged.TracesSampleRate = stored.TracesSampleRate
		if merged.TracesSampleRate < 0 {
			merged.TracesSampleRate = 0
		}
	}

	if !merged.Enabled() {
		merged.ClientReportingAllowed = false
	}

	return merged
}

func loadSentryConfig() SentryConfig {
	return loadSentryConfigFromEnv()
}

func LoadSentryConfigFromEnv() SentryConfig {
	return loadSentryConfigFromEnv()
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func envFloatFirst(keys []string, fallback float64) float64 {
	for _, key := range keys {
		raw := strings.TrimSpace(os.Getenv(key))
		if raw == "" {
			continue
		}
		var value float64
		if _, err := fmt.Sscanf(raw, "%f", &value); err == nil {
			return value
		}
	}
	return fallback
}
