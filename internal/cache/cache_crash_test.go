// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResponseCacheCrashSafeEmptyInputs(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	c := NewResponseCache()
	c.Set("", Entry{})
	c.Set("GET /", Entry{StatusCode: http.StatusOK, Body: nil, ExpiresAt: time.Now().Add(time.Minute)})
	_, _ = c.Get("")
	_, _ = c.Get("missing")
	_ = c.InvalidatePrefix("")
	_ = c.InvalidatePrefix("nope")
	_ = Key("", "", "")
	_ = Key("GET", "/x", "")
	_ = ShouldCacheRequest(http.MethodGet, "/stream")
	_ = ShouldCacheRequest("", "")
	WriteCachedResponse(httptest.NewRecorder(), Entry{StatusCode: 200, Body: []byte("ok")}, "HIT")
}
