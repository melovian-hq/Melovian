// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package localmusic

import (
	"fmt"
	"sync"
	"testing"
)

func TestCatalogCacheConcurrentSetGet(t *testing.T) {
	cache := NewCatalogCache()
	catalog := Catalog{Songs: map[string]Song{"a": {ID: "a"}}}

	var wg sync.WaitGroup
	for i := range 32 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("lib_%d", n%maxCatalogCacheEntries)
			cache.Set(id, uint64(n), catalog)
			_, _ = cache.Get(id, uint64(n))
			_, _ = cache.Stats()
		}(i)
	}
	wg.Wait()

	entries, _ := cache.Stats()
	if entries > maxCatalogCacheEntries {
		t.Fatalf("entries=%d want <= %d", entries, maxCatalogCacheEntries)
	}
}

func BenchmarkCatalogCacheSetGet(b *testing.B) {
	cache := NewCatalogCache()
	catalog := Catalog{Songs: map[string]Song{"a": {ID: "a", Title: "Song"}}}
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		id := fmt.Sprintf("lib_%d", i%maxCatalogCacheEntries)
		cache.Set(id, uint64(i), catalog)
		_, _ = cache.Get(id, uint64(i))
	}
}
