// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

const PrefKeySentryClient = "sentry_client_settings"

// Consent states for the first-start telemetry prompt. Empty and "unset"
// both mean the user has not answered yet.
const (
	SentryChoiceUnset    = ""
	SentryChoiceAccepted = "accepted"
	SentryChoiceDeclined = "declined"
)

type SentryClientSettings struct {
	Enabled bool `json:"enabled"`
	// Choice is the recorded consent answer. Legacy rows that only stored
	// enabled:true are treated as accepted by NormalizeSentryChoice.
	Choice string `json:"choice,omitempty"`
}

func DefaultSentryClientSettings() SentryClientSettings {
	return SentryClientSettings{Enabled: false}
}

// NormalizeSentryChoice maps stored state onto a consent answer. Legacy
// installs that already enabled reporting count as accepted so they are not
// re-prompted.
func NormalizeSentryChoice(s SentryClientSettings) string {
	switch s.Choice {
	case SentryChoiceAccepted, SentryChoiceDeclined:
		return s.Choice
	default:
		if s.Enabled {
			return SentryChoiceAccepted
		}
		return SentryChoiceUnset
	}
}
