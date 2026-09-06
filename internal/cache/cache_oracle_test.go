// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestResponseCacheGetMissesAfterExpiryOracle(t *testing.T) {
	c := NewResponseCache()
	key := "GET /oracle/expired"
	c.Set(key, Entry{
		StatusCode: http.StatusOK,
		Body:       []byte("stale"),
		ExpiresAt:  time.Now().Add(-time.Second),
	})
	if _, ok := c.Get(key); ok {
		t.Fatal("expected miss after expiry")
	}
}

func TestResponseCacheSetRejectsOversizedBodyOracle(t *testing.T) {
	c := NewResponseCache()
	key := "GET /oracle/large"
	c.Set(key, Entry{
		StatusCode: http.StatusOK,
		Body:       make([]byte, MaxBodyBytes+1),
		ExpiresAt:  time.Now().Add(time.Minute),
	})
	if _, ok := c.Get(key); ok {
		t.Fatal("expected oversized body to be rejected")
	}
	entries, total := c.Stats()
	if entries != 0 || total != 0 {
		t.Fatalf("expected empty cache after reject, entries=%d total=%d", entries, total)
	}
}

func TestResponseCacheCapacityEvictionOracle(t *testing.T) {
	c := NewResponseCache()
	now := time.Now()
	body := make([]byte, 32<<10)

	for i := range MaxEntries + 64 {
		c.Set(fmt.Sprintf("GET /oracle/cap/%d", i), Entry{
			StatusCode: http.StatusOK,
			Body:       body,
			ExpiresAt:  now.Add(time.Hour),
		})
	}

	entries, total := c.Stats()
	if entries > MaxEntries {
		t.Fatalf("entries %d exceed MaxEntries %d", entries, MaxEntries)
	}
	if total > MaxTotalBytes {
		t.Fatalf("totalBytes %d exceed MaxTotalBytes %d", total, MaxTotalBytes)
	}
}
