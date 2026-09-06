// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package smartplaylist

import "testing"

func TestMatchDraftJSONGenreContains(t *testing.T) {
	rules := `{
		"root": {
			"logic": "all",
			"rules": [{"field":"genre","operator":"contains","value":"jazz"}],
			"groups": []
		}
	}`
	track := TrackContext{Title: "Song", Genre: "Modern Jazz"}
	if !MatchDraftJSON(rules, track) {
		t.Fatal("expected jazz match")
	}
	track.Genre = "Rock"
	if MatchDraftJSON(rules, track) {
		t.Fatal("expected non-match")
	}
}
