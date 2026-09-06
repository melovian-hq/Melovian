// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"net/http"
	"testing"
	"time"
)

func TestResponseCacheStoreThenFetchAcceptance(t *testing.T) {
	c := NewResponseCache()
	key := Key("GET", "/Items/media-1", "fields=PrimaryImage")
	header := http.Header{}
	header.Set("Content-Type", "application/json")
	header.Set("ETag", `"abc"`)

	body := []byte(`{"Id":"media-1","Name":"Track"}`)
	c.Set(key, Entry{
		StatusCode: http.StatusOK,
		Header:     CloneHeader(header),
		Body:       body,
		ExpiresAt:  time.Now().Add(TTL("/Items/media-1")),
	})

	got, ok := c.Get(key)
	if !ok {
		t.Fatal("expected cache hit after store")
	}
	if got.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want %d", got.StatusCode, http.StatusOK)
	}
	if string(got.Body) != string(body) {
		t.Fatalf("body mismatch: %q", got.Body)
	}
	if got.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("missing content type: %v", got.Header)
	}
	if got.Header.Get("ETag") != `"abc"` {
		t.Fatalf("missing etag: %v", got.Header)
	}
}
