// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestConcurrentMusicProgressUpdatesCompletes(t *testing.T) {
	srv, _ := newTestServer(t)
	trackID := "race-track-1"

	done := make(chan struct{})
	var wg sync.WaitGroup

	for worker := range 12 {
		wg.Go(func() {
			for i := range 40 {
				body := bytes.NewBufferString(`{
					"positionMs": 1000,
					"played": false,
					"trackTitle": "Race Track",
					"artistName": "Artist",
					"albumId": "album-1",
					"albumTitle": "Album",
					"durationMs": 180000,
					"coverArtId": "cover-1",
					"incrementPlay": false
				}`)
				putReq := httptest.NewRequest(http.MethodPut, "/api/music/items/"+trackID, body)
				putRec := httptest.NewRecorder()
				srv.Handler().ServeHTTP(putRec, putReq)

				getReq := httptest.NewRequest(http.MethodGet, "/api/music/items/"+trackID, nil)
				getRec := httptest.NewRecorder()
				srv.Handler().ServeHTTP(getRec, getReq)

				if i%10 == 0 {
					histReq := httptest.NewRequest(http.MethodGet, "/api/music/history?limit=5", nil)
					histRec := httptest.NewRecorder()
					srv.Handler().ServeHTTP(histRec, histReq)
				}
				_ = worker
			}
		})
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(12 * time.Second):
		t.Fatal("concurrent music progress updates did not complete within timeout")
	}
}

func TestConcurrentMusicPlaylistAndFavoriteAccessCompletes(t *testing.T) {
	srv, _ := newTestServer(t)

	createBody := bytes.NewBufferString(`{"name":"Race Playlist"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/music/playlists", createBody)
	createRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create playlist status %d", createRec.Code)
	}

	done := make(chan struct{})
	var wg sync.WaitGroup

	for range 10 {
		wg.Go(func() {
			for range 30 {
				listReq := httptest.NewRequest(http.MethodGet, "/api/music/playlists", nil)
				listRec := httptest.NewRecorder()
				srv.Handler().ServeHTTP(listRec, listReq)

				favReq := httptest.NewRequest(http.MethodGet, "/api/music/favorites", nil)
				favRec := httptest.NewRecorder()
				srv.Handler().ServeHTTP(favRec, favReq)

				statsReq := httptest.NewRequest(http.MethodGet, "/api/music/stats", nil)
				statsRec := httptest.NewRecorder()
				srv.Handler().ServeHTTP(statsRec, statsReq)
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
		t.Fatal("concurrent playlist/favorite reads did not complete within timeout")
	}
}
