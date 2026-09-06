// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import "testing"

func TestSourceViewModeGlobalPersistence(t *testing.T) {
	db := OpenTestDB(t)
	prefs := NewPreferencesStore(db)

	mode, err := prefs.GetSourceViewMode("")
	if err != nil {
		t.Fatalf("GetSourceViewMode: %v", err)
	}
	if mode != SourceViewSubsonic {
		t.Fatalf("default mode = %q, want subsonic", mode)
	}

	if err := prefs.SetSourceViewMode("", SourceViewLocal); err != nil {
		t.Fatalf("SetSourceViewMode: %v", err)
	}
	mode, err = prefs.GetSourceViewMode("")
	if err != nil {
		t.Fatalf("GetSourceViewMode after set: %v", err)
	}
	if mode != SourceViewLocal {
		t.Fatalf("mode = %q, want local", mode)
	}
}

func TestSourceViewModePerUserPersistence(t *testing.T) {
	db := OpenTestDB(t)
	prefs := NewPreferencesStore(db)

	if err := prefs.SetSourceViewMode("user-1", SourceViewUnified); err != nil {
		t.Fatalf("SetSourceViewMode: %v", err)
	}
	mode, err := prefs.GetSourceViewMode("user-1")
	if err != nil {
		t.Fatalf("GetSourceViewMode: %v", err)
	}
	if mode != SourceViewUnified {
		t.Fatalf("mode = %q, want unified", mode)
	}
}
