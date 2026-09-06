// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package localmusic

import (
	"fmt"
	"testing"
	"time"

	"melovian/internal/store"
)

func TestCatalogCacheHitAndInvalidate(t *testing.T) {
	cache := NewCatalogCache()
	lib := store.LocalLibrary{
		ID:             "lib_1",
		UpdatedAt:      time.Unix(100, 0),
		TrackCount:     10,
		MissingCount:   0,
		DuplicateCount: 0,
	}
	version := CatalogVersion(lib)
	catalog := Catalog{
		Songs: map[string]Song{"trk_1": {ID: "trk_1", Title: "Song"}},
	}

	cache.Set(lib.ID, version, catalog)
	got, ok := cache.Get(lib.ID, version)
	if !ok || len(got.Songs) != 1 {
		t.Fatalf("expected cached catalog, got ok=%v songs=%d", ok, len(got.Songs))
	}

	lib.TrackCount = 11
	if _, ok := cache.Get(lib.ID, CatalogVersion(lib)); ok {
		t.Fatal("expected cache miss after version change")
	}

	cache.Set(lib.ID, CatalogVersion(lib), catalog)
	cache.Invalidate(lib.ID)
	if _, ok := cache.Get(lib.ID, CatalogVersion(lib)); ok {
		t.Fatal("expected cache miss after invalidate")
	}
}

func TestCatalogCacheEvictsOldest(t *testing.T) {
	cache := NewCatalogCache()
	catalog := Catalog{Songs: map[string]Song{"trk_1": {ID: "trk_1"}}}

	for i := 1; i <= maxCatalogCacheEntries+1; i++ {
		id := fmt.Sprintf("lib_%d", i)
		cache.Set(id, uint64(i), catalog)
	}

	entries, _ := cache.Stats()
	if entries != maxCatalogCacheEntries {
		t.Fatalf("expected %d entries, got %d", maxCatalogCacheEntries, entries)
	}
	if _, ok := cache.Get("lib_1", 1); ok {
		t.Fatal("expected oldest catalog entry to be evicted")
	}
}
