// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package apishared

import (
	"melovian/internal/store"
)

// SourceViewModeForUser returns the source view mode for the user, falling
// back to the subsonic view when preferences cannot be read.
func SourceViewModeForUser(preferences *store.PreferencesStore, userID string) string {
	mode, err := preferences.GetSourceViewMode(userID)
	if err != nil {
		return store.SourceViewSubsonic
	}
	return mode
}

// ShouldKeepOtherSourceOnActivate reports whether activating a source must
// leave the other source active, which is the case in unified view.
func ShouldKeepOtherSourceOnActivate(preferences *store.PreferencesStore, userID string) bool {
	return store.IsUnifiedSourceView(SourceViewModeForUser(preferences, userID))
}
