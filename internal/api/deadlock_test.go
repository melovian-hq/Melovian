// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"melovian/internal/store"
)

func TestConcurrentInstanceSwitchingCompletes(t *testing.T) {
	srv, db := newTestServer(t)
	instances := store.NewInstanceStore(db)

	ids := make([]string, 0, 4)
	for i := range 4 {
		inst, err := instances.Create(store.CreateInstanceInput{
			Name:      string(rune('A' + i)),
			ServerURL: "http://sub.example.com",
			Username:  "user",
			Password:  "pass",
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		ids = append(ids, inst.ID)
	}

	done := make(chan struct{})
	var wg sync.WaitGroup

	for range 8 {
		wg.Go(func() {
			for range 50 {
				for _, id := range ids {
					req := httptest.NewRequest(http.MethodPost, "/api/instances/"+id+"/activate", nil)
					rec := httptest.NewRecorder()
					srv.Handler().ServeHTTP(rec, req)
				}
			}
		})
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(60 * time.Second):
		t.Fatal("concurrent instance switching did not complete within timeout")
	}
}
