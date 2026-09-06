// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestConcurrentMusicSettingsReadsDoNotDeadlock(t *testing.T) {
	srv, _ := newTestServer(t)

	done := make(chan struct{})
	var wg sync.WaitGroup

	for range 8 {
		wg.Go(func() {
			for range 60 {
				eqReq := httptest.NewRequest(http.MethodGet, "/api/music/settings/eq", nil)
				eqRec := httptest.NewRecorder()
				srv.Handler().ServeHTTP(eqRec, eqReq)

				connReq := httptest.NewRequest(http.MethodGet, "/api/music/settings/connection", nil)
				connRec := httptest.NewRecorder()
				srv.Handler().ServeHTTP(connRec, connReq)
			}
		})
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("concurrent music settings reads did not complete within timeout")
	}
}
