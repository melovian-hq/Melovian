// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestResponseCacheConcurrentGetSet(t *testing.T) {
	c := NewResponseCache()
	const workers = 32
	const ops = 200

	var wg sync.WaitGroup
	wg.Add(workers)
	for worker := range workers {
		go func(id int) {
			defer wg.Done()
			for i := range ops {
				key := Key("GET", "/Items", string(rune('a'+id%26))+string(rune('0'+i%10)))
				c.Set(key, Entry{
					StatusCode: http.StatusOK,
					Body:       []byte("ok"),
					ExpiresAt:  time.Now().Add(time.Minute),
				})
				c.Get(key)
			}
		}(worker)
	}
	wg.Wait()
}

func TestResponseCacheBoundedEntries(t *testing.T) {
	c := NewResponseCache()
	now := time.Now()

	for i := range MaxEntries + 128 {
		key := Key("GET", "/Items/bounded", string(rune(i)))
		c.Set(key, Entry{
			StatusCode: http.StatusOK,
			Body:       []byte("payload"),
			ExpiresAt:  now.Add(time.Hour),
		})
	}

	c.mu.Lock()
	size := len(c.entries)
	c.mu.Unlock()

	if size > MaxEntries {
		t.Fatalf("cache grew beyond MaxEntries: got %d want <= %d", size, MaxEntries)
	}
}

func TestResponseCacheEvictsExpiredBeforeOldest(t *testing.T) {
	c := NewResponseCache()
	now := time.Now()

	for i := range MaxEntries {
		key := Key("GET", "/Expired", string(rune(i)))
		c.Set(key, Entry{
			StatusCode: http.StatusOK,
			Body:       []byte("old"),
			ExpiresAt:  now.Add(-time.Minute),
		})
	}

	c.Set("GET /fresh", Entry{
		StatusCode: http.StatusOK,
		Body:       []byte("fresh"),
		ExpiresAt:  now.Add(time.Hour),
	})

	if _, ok := c.Get("GET /fresh"); !ok {
		t.Fatal("expected fresh entry to remain accessible")
	}
}

func TestResponseCacheBoundedTotalBytes(t *testing.T) {
	c := NewResponseCache()
	now := time.Now()
	body := make([]byte, 1<<20)

	for i := range MaxTotalBytes/len(body) + 32 {
		key := Key("GET", "/Large", string(rune(i)))
		c.Set(key, Entry{
			StatusCode: http.StatusOK,
			Body:       body,
			ExpiresAt:  now.Add(time.Hour),
		})
	}

	c.mu.Lock()
	total := c.totalBytes
	c.mu.Unlock()

	if total > MaxTotalBytes {
		t.Fatalf("cache grew beyond MaxTotalBytes: got %d want <= %d", total, MaxTotalBytes)
	}
}
