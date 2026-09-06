// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"testing"
)

func TestTrackVideoLinkUpsertAndGet(t *testing.T) {
	db := OpenTestDB(t)
	links := NewTrackVideoLinkStore(db)

	saved, err := links.Upsert(UpsertTrackVideoLinkInput{
		UserID:  "local",
		TrackID: "trk_1",
		Source:  VideoSourceInvidious,
		VideoID: "abc123",
		Title:   "Official Video",
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if saved.VideoID != "abc123" || saved.Source != VideoSourceInvidious {
		t.Fatalf("unexpected link: %+v", saved)
	}

	got, err := links.Get("local", "trk_1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Official Video" {
		t.Fatalf("title=%q", got.Title)
	}

	if err := links.Delete("local", "trk_1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
