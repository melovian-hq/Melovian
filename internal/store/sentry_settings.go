// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

const PrefKeySentryClient = "sentry_client_settings"

type SentryClientSettings struct {
	Enabled bool `json:"enabled"`
}

func DefaultSentryClientSettings() SentryClientSettings {
	return SentryClientSettings{Enabled: true}
}
