// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"database/sql"
	"errors"
	"testing"
)

func TestDownloadStoreLifecycle(t *testing.T) {
	db := OpenTestDB(t)
	s := NewDownloadStore(db)

	if _, err := s.Get("inst-1", "track-1"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected ErrNoRows for missing entry, got %v", err)
	}

	entry := DownloadedTrack{
		InstanceID:  "inst-1",
		TrackID:     "track-1",
		Path:        "/tmp/track-1",
		ContentType: "audio/mpeg",
		Size:        2048,
		TrackTitle:  "Test Track",
		ArtistName:  "Test Artist",
	}
	if err := s.Upsert(entry); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := s.Get("inst-1", "track-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Path != entry.Path || got.Size != entry.Size {
		t.Fatalf("unexpected entry: %+v", got)
	}
	if got.CreatedAt == 0 {
		t.Fatal("expected CreatedAt to be populated")
	}

	// Upsert again updates fields without duplicating rows.
	entry.Size = 4096
	if err := s.Upsert(entry); err != nil {
		t.Fatalf("Upsert update: %v", err)
	}

	list, err := s.List("inst-1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(list))
	}
	if list[0].Size != 4096 {
		t.Fatalf("expected updated size 4096, got %d", list[0].Size)
	}

	// Downloads are scoped per instance.
	if other, err := s.List("inst-2"); err != nil || len(other) != 0 {
		t.Fatalf("expected empty list for other instance, got %d (%v)", len(other), err)
	}

	path, err := s.Delete("inst-1", "track-1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if path != "/tmp/track-1" {
		t.Fatalf("expected returned path, got %q", path)
	}

	if _, err := s.Get("inst-1", "track-1"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected entry removed, got %v", err)
	}

	// Deleting a missing entry is a no-op with empty path.
	if path, err := s.Delete("inst-1", "track-1"); err != nil || path != "" {
		t.Fatalf("expected no-op delete, got path=%q err=%v", path, err)
	}
}

func TestDownloadStoreEvictUntil(t *testing.T) {
	db := OpenTestDB(t)
	s := NewDownloadStore(db)

	base := nowUnix()
	for i, size := range []int64{100, 200, 300} {
		id := []string{"a", "b", "c"}[i]
		if err := s.Upsert(DownloadedTrack{
			InstanceID: "inst-1",
			TrackID:    "track-" + id,
			Path:       "/tmp/track-" + id,
			Size:       size,
			CreatedAt:  base + int64(i),
		}); err != nil {
			t.Fatalf("Upsert %d: %v", i, err)
		}
	}

	paths, err := s.EvictUntil("inst-1", 300)
	if err != nil {
		t.Fatalf("EvictUntil: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 evicted paths, got %d", len(paths))
	}

	total, count, err := s.Stats("inst-1")
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if count != 1 || total != 300 {
		t.Fatalf("expected one 300-byte track left, got count=%d total=%d", count, total)
	}
}
