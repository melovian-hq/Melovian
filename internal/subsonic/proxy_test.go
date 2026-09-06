// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonic

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"melovian/internal/cache"
)

func TestProxyBypassesCoverArtCache(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "getCoverArt") {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("fake-jpeg"))
	}))
	t.Cleanup(upstream.Close)

	responseCache := cache.NewResponseCache()
	handler := NewProxy(func(context.Context) *Client {
		return NewClient(upstream.URL, "alice", "secret")
	}, responseCache, true, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/subsonic/rest/getCoverArt.view?id=1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Cache"); got != "BYPASS" && got != "" {
		// Cover art is not cacheable, so response may stream without X-Cache or with BYPASS.
		if got == "HIT" || got == "MISS" {
			t.Fatalf("cover art should not be cached, got X-Cache=%q", got)
		}
	}
	entries, _ := responseCache.Stats()
	if entries != 0 {
		t.Fatalf("expected cover art to bypass cache, entries=%d", entries)
	}
}

func TestProxyLimitReaderBypassesOversizedBodies(t *testing.T) {
	payload := strings.Repeat("x", cache.MaxBodyBytes+4096)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, payload)
	}))
	t.Cleanup(upstream.Close)

	responseCache := cache.NewResponseCache()
	handler := NewProxy(func(context.Context) *Client {
		return NewClient(upstream.URL, "alice", "secret")
	}, responseCache, true, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/subsonic/rest/getAlbum.view?id=1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec.Header().Get("X-Cache") != "BYPASS" {
		t.Fatalf("expected BYPASS for oversized body, got %q", rec.Header().Get("X-Cache"))
	}
	if rec.Body.Len() != len(payload) {
		t.Fatalf("body length=%d want %d", rec.Body.Len(), len(payload))
	}
	entries, _ := responseCache.Stats()
	if entries != 0 {
		t.Fatalf("oversized body should not be cached, entries=%d", entries)
	}
}

func TestProxyCachesSmallJSON(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	t.Cleanup(upstream.Close)

	responseCache := cache.NewResponseCache()
	handler := NewProxy(func(context.Context) *Client {
		return NewClient(upstream.URL, "alice", "secret")
	}, responseCache, true, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/subsonic/rest/ping.view", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("X-Cache") != "MISS" {
		t.Fatalf("first request want MISS, got %q", rec.Header().Get("X-Cache"))
	}

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req)
	if rec2.Header().Get("X-Cache") != "HIT" {
		t.Fatalf("second request want HIT, got %q", rec2.Header().Get("X-Cache"))
	}
}

func TestProxyForwardsNavidromeShareImagesWithoutAuth(t *testing.T) {
	var gotPath, gotRawQuery string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("img"))
	}))
	t.Cleanup(upstream.Close)

	handler := NewProxy(func(context.Context) *Client {
		return NewClient(upstream.URL, "alice", "secret")
	}, cache.NewResponseCache(), false, nil)

	token := "eyJhbGciOiJIUzI1NiJ9.token"
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/subsonic/share/img/"+token+"?size=600&_instance=inst-1",
		nil,
	)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if gotPath != "/share/img/"+token {
		t.Fatalf("path=%q", gotPath)
	}
	if strings.Contains(gotRawQuery, "u=") || strings.Contains(gotRawQuery, "t=") {
		t.Fatalf("share image must not receive subsonic auth, query=%q", gotRawQuery)
	}
	if !strings.Contains(gotRawQuery, "size=600") {
		t.Fatalf("expected size query preserved, got %q", gotRawQuery)
	}
	if strings.Contains(gotRawQuery, "_instance") {
		t.Fatalf("_instance must be stripped before upstream, query=%q", gotRawQuery)
	}
}
