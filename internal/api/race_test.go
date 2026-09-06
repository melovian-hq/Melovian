// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"melovian/internal/store"
)

func TestConcurrentActivateAndMusicReads(t *testing.T) {
	srv, db := newTestServer(t)
	instances := store.NewInstanceStore(db)

	instA, err := instances.Create(store.CreateInstanceInput{
		Name: "A", ServerURL: "http://a.example.com", Username: "u", Password: "p",
	})
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	instB, err := instances.Create(store.CreateInstanceInput{
		Name: "B", ServerURL: "http://b.example.com", Username: "u", Password: "p",
	})
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(2)
		go func(id string) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/instances/"+id+"/activate", nil)
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)
		}(instA.ID)
		go func(id string) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/api/music/history", nil)
			req.Header.Set("X-Instance-Id", id)
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)
		}(instB.ID)
	}
	wg.Wait()
}
