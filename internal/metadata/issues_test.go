// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"testing"

	"melovian/internal/store"
)

func TestTrackIssues(t *testing.T) {
	track := store.LocalTrack{
		Title:  "Song",
		Artist: "Unknown Artist",
		Album:  "Real Album",
	}
	issues := TrackIssues(track)
	if len(issues) != 1 || issues[0] != IssueUnknownArtist {
		t.Fatalf("issues = %#v, want unknown-artist only", issues)
	}
}

func TestTrackHasIssueAny(t *testing.T) {
	track := store.LocalTrack{Title: "", Artist: "Artist", Album: "Album"}
	if !TrackHasIssue(track, IssueAny) {
		t.Fatal("expected any issue")
	}
}
