// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonic

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"melovian/internal/cache"
)

func BenchmarkProxyCacheHit(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	b.Cleanup(upstream.Close)

	handler := NewProxy(func(context.Context) *Client {
		return NewClient(upstream.URL, "alice", "secret")
	}, cache.NewResponseCache(), true, nil)

	warm := httptest.NewRequest(http.MethodGet, "/api/subsonic/rest/ping.view", nil)
	handler.ServeHTTP(httptest.NewRecorder(), warm)

	b.ReportAllocs()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/api/subsonic/rest/ping.view", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
