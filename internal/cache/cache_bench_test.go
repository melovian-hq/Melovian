// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func BenchmarkCacheKey(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = Key("GET", "/Items/abc", "limit=10&sort=name")
	}
}

func BenchmarkResponseCacheGetHit(b *testing.B) {
	c := NewResponseCache()
	key := "GET /Items/abc"
	c.Set(key, Entry{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       []byte(`{"ok":true}`),
		ExpiresAt:  time.Now().Add(time.Minute),
	})

	b.ReportAllocs()
	for b.Loop() {
		_, _ = c.Get(key)
	}
}

func BenchmarkWriteCachedResponse(b *testing.B) {
	entry := Entry{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}, "X-Test": {"a", "b"}},
		Body:       []byte(`{"ok":true,"items":[]}`),
	}

	b.ReportAllocs()
	for b.Loop() {
		rec := httptest.NewRecorder()
		WriteCachedResponse(rec, entry, "HIT")
	}
}
