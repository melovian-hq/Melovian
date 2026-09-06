// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestResponseCacheGetExpired(t *testing.T) {
	c := NewResponseCache()
	key := "GET /test"
	c.Set(key, Entry{
		StatusCode: http.StatusOK,
		Body:       []byte("gone"),
		ExpiresAt:  time.Now().Add(-time.Minute),
	})

	if _, ok := c.Get(key); ok {
		t.Fatal("expected expired entry to miss")
	}
	if _, ok := c.Get(key); ok {
		t.Fatal("expected expired entry to be removed on second get")
	}
}

func TestResponseCacheRejectsOversizeBody(t *testing.T) {
	c := NewResponseCache()
	key := "GET /large"
	body := make([]byte, MaxBodyBytes+1)
	c.Set(key, Entry{
		StatusCode: http.StatusOK,
		Body:       body,
		ExpiresAt:  time.Now().Add(time.Minute),
	})

	if _, ok := c.Get(key); ok {
		t.Fatal("expected oversize entry to be rejected")
	}
}

func TestResponseCacheInvalidatePrefix(t *testing.T) {
	c := NewResponseCache()
	c.Set("user:1:instance:a|GET /rest/getArtist", Entry{
		StatusCode: http.StatusOK,
		Body:       []byte("artist"),
		ExpiresAt:  time.Now().Add(time.Minute),
	})
	c.Set("user:1:instance:b|GET /rest/getArtist", Entry{
		StatusCode: http.StatusOK,
		Body:       []byte("other"),
		ExpiresAt:  time.Now().Add(time.Minute),
	})

	if removed := c.InvalidatePrefix("user:1:instance:a|"); removed != 1 {
		t.Fatalf("InvalidatePrefix() = %d, want 1", removed)
	}
	if _, ok := c.Get("user:1:instance:a|GET /rest/getArtist"); ok {
		t.Fatal("expected scoped entry to be removed")
	}
	if _, ok := c.Get("user:1:instance:b|GET /rest/getArtist"); !ok {
		t.Fatal("expected other scoped entry to remain")
	}
}

func TestKeyAndTTL(t *testing.T) {
	if got := Key("GET", "/Items/abc", "limit=10"); got != "GET /Items/abc?limit=10" {
		t.Fatalf("unexpected key %q", got)
	}
	if got := Key("GET", "/Items/abc", ""); got != "GET /Items/abc" {
		t.Fatalf("unexpected key %q", got)
	}

	if TTL("/Images/cover") != 24*time.Hour {
		t.Fatalf("expected cover art ttl 24h, got %v", TTL("/Images/cover"))
	}
	if TTL("/Search/query") != 90*time.Second {
		t.Fatalf("expected search ttl 90s, got %v", TTL("/Search/query"))
	}
	if TTL("/Unknown") != 2*time.Minute {
		t.Fatalf("expected default ttl 2m, got %v", TTL("/Unknown"))
	}
}

func TestShouldCacheRequest(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   bool
	}{
		{http.MethodGet, "/Items/abc", true},
		{http.MethodPost, "/Items/abc", false},
		{http.MethodGet, "/Videos/abc", false},
		{http.MethodGet, "/Sessions/abc", false},
		{http.MethodGet, "/Items/abc/PlaybackInfo", false},
		{http.MethodGet, "/rest/stream.view", false},
		{http.MethodGet, "/playlist.m3u8", false},
		{http.MethodGet, "/segment.ts", false},
	}

	for _, tc := range tests {
		if got := ShouldCacheRequest(tc.method, tc.path); got != tc.want {
			t.Fatalf("ShouldCacheRequest(%q, %q) = %v, want %v", tc.method, tc.path, got, tc.want)
		}
	}
}

func TestWriteCachedResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	entry := Entry{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       []byte(`{"ok":true}`),
	}
	WriteCachedResponse(rec, entry, "HIT")

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if rec.Header().Get("X-Cache") != "HIT" {
		t.Fatalf("expected X-Cache HIT, got %q", rec.Header().Get("X-Cache"))
	}
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("unexpected body %q", rec.Body.String())
	}
}
