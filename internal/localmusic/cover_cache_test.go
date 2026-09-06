// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package localmusic

import (
	"fmt"
	"sync"
	"testing"
)

func TestCoverCacheBoundsAndStats(t *testing.T) {
	c := NewCoverCache()
	payload := make([]byte, 256<<10)

	for i := range maxCoverCacheEntries + 64 {
		c.Set("lib", fmt.Sprintf("trk_%d", i), payload, "image/jpeg")
	}

	entries, total := c.Stats()
	if entries > maxCoverCacheEntries {
		t.Fatalf("entries=%d want <= %d", entries, maxCoverCacheEntries)
	}
	if total > maxCoverCacheBytes {
		t.Fatalf("totalBytes=%d want <= %d", total, maxCoverCacheBytes)
	}
}

func TestCoverCacheConcurrentAccess(t *testing.T) {
	c := NewCoverCache()
	var wg sync.WaitGroup
	for worker := range 16 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := range 100 {
				key := fmt.Sprintf("%d-%d", id, i)
				c.Set("lib", key, []byte(key), "image/png")
				_, _, _ = c.Get("lib", key)
			}
		}(worker)
	}
	wg.Wait()
}

func BenchmarkCoverCacheSetGet(b *testing.B) {
	c := NewCoverCache()
	payload := make([]byte, 8<<10)
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		id := fmt.Sprintf("t-%d", i%128)
		c.Set("lib", id, payload, "image/jpeg")
		_, _, _ = c.Get("lib", id)
	}
}
