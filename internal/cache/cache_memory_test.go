// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestResponseCacheStatsTracksBytes(t *testing.T) {
	c := NewResponseCache()
	now := time.Now()
	body := []byte("hello-world-payload")
	c.Set("GET /a", Entry{
		StatusCode: 200,
		Body:       body,
		ExpiresAt:  now.Add(time.Hour),
	})

	entries, total := c.Stats()
	if entries != 1 {
		t.Fatalf("entries=%d want 1", entries)
	}
	if total != len(body) {
		t.Fatalf("totalBytes=%d want %d", total, len(body))
	}
}

func TestResponseCacheMemoryBudgetUnderPressure(t *testing.T) {
	c := NewResponseCache()
	now := time.Now()
	body := make([]byte, 256<<10)

	var before runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	for i := range 512 {
		c.Set(fmt.Sprintf("GET /mem/%d", i), Entry{
			StatusCode: 200,
			Body:       body,
			ExpiresAt:  now.Add(time.Hour),
		})
	}

	entries, total := c.Stats()
	if entries > MaxEntries {
		t.Fatalf("entries exceeded MaxEntries: %d", entries)
	}
	if total > MaxTotalBytes {
		t.Fatalf("totalBytes exceeded MaxTotalBytes: %d", total)
	}

	var after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&after)
	growth := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	const softLimit = int64(MaxTotalBytes) * 4
	if growth > softLimit {
		t.Fatalf("heap growth %d exceeds soft limit %d", growth, softLimit)
	}
}

func TestResponseCacheReplaceDoesNotLeakBytes(t *testing.T) {
	c := NewResponseCache()
	now := time.Now()
	key := "GET /replace"
	c.Set(key, Entry{StatusCode: 200, Body: []byte("tiny"), ExpiresAt: now.Add(time.Hour)})
	c.Set(key, Entry{StatusCode: 200, Body: []byte("much-longer-body"), ExpiresAt: now.Add(time.Hour)})

	entries, total := c.Stats()
	if entries != 1 {
		t.Fatalf("entries=%d want 1", entries)
	}
	if total != len("much-longer-body") {
		t.Fatalf("totalBytes=%d want %d", total, len("much-longer-body"))
	}
}
