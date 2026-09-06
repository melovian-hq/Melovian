// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMusicHistoryContract(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/music/history?limit=5", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Items == nil {
		t.Fatal("expected items array in response")
	}
}

func TestMusicListenEventsContract(t *testing.T) {
	srv, _ := newTestServer(t)

	putBody := bytes.NewBufferString(`{
		"positionMs": 0,
		"incrementPlay": true,
		"trackTitle": "History Track",
		"artistName": "Artist",
		"albumTitle": "Album",
		"durationMs": 180000,
		"coverArtId": "cover-1"
	}`)
	putReq := httptest.NewRequest(http.MethodPut, "/api/music/items/event-track-1", putBody)
	putRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("upsert status %d: %s", putRec.Code, putRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/music/listen-events?limit=10", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Items   []map[string]any `json:"items"`
		HasMore bool             `json:"hasMore"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Items) == 0 {
		t.Fatal("expected at least one listen event")
	}
	if payload.Items[0]["trackTitle"] != "History Track" {
		t.Fatalf("unexpected track title %v", payload.Items[0]["trackTitle"])
	}
}

func TestMusicClearListenEventsContract(t *testing.T) {
	srv, _ := newTestServer(t)

	putBody := bytes.NewBufferString(`{
		"positionMs": 0,
		"incrementPlay": true,
		"trackTitle": "History Track",
		"artistName": "Artist",
		"albumTitle": "Album",
		"durationMs": 180000
	}`)
	putReq := httptest.NewRequest(http.MethodPut, "/api/music/items/clear-track-1", putBody)
	putRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("upsert status %d: %s", putRec.Code, putRec.Body.String())
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/api/music/listen-events", nil)
	delRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("clear status %d: %s", delRec.Code, delRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/music/listen-events?limit=10", nil)
	getRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("list status %d: %s", getRec.Code, getRec.Body.String())
	}
	var payload struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Items) != 0 {
		t.Fatalf("expected empty history, got %+v", payload.Items)
	}
}

func TestMusicProgressUpsertAndGetContract(t *testing.T) {
	srv, _ := newTestServer(t)
	trackID := "track-contract-1"

	body := bytes.NewBufferString(`{
		"positionMs": 120000,
		"played": false,
		"trackTitle": "Test Track",
		"artistName": "Test Artist",
		"albumId": "album-1",
		"albumTitle": "Test Album",
		"durationMs": 240000,
		"coverArtId": "cover-1"
	}`)
	putReq := httptest.NewRequest(http.MethodPut, "/api/music/items/"+trackID, body)
	putRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("upsert status %d: %s", putRec.Code, putRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/music/items/"+trackID, nil)
	getRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status %d: %s", getRec.Code, getRec.Body.String())
	}

	var item map[string]any
	if err := json.Unmarshal(getRec.Body.Bytes(), &item); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if item["trackId"] != trackID {
		t.Fatalf("expected trackId %q, got %v", trackID, item["trackId"])
	}
	if item["trackTitle"] != "Test Track" {
		t.Fatalf("expected track title preserved, got %v", item["trackTitle"])
	}
}

func TestMusicResumeInvalidLimitUsesFallback(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/music/resume?limit=not-a-number", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMusicPlaylistLifecycleContract(t *testing.T) {
	srv, _ := newTestServer(t)

	createBody := bytes.NewBufferString(`{"name":"Fuzz Playlist"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/music/playlists", createBody)
	createRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status %d: %s", createRec.Code, createRec.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	playlistID, _ := created["id"].(string)
	if playlistID == "" {
		t.Fatal("expected playlist id")
	}

	addBody := bytes.NewBufferString(`{
		"trackId":"track-1",
		"trackTitle":"Song",
		"artistName":"Artist",
		"albumId":"album-1",
		"albumTitle":"Album",
		"durationMs":180000,
		"coverArtId":"cover-1"
	}`)
	addReq := httptest.NewRequest(http.MethodPost, "/api/music/playlists/"+playlistID+"/tracks", addBody)
	addRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(addRec, addReq)
	if addRec.Code != http.StatusOK {
		t.Fatalf("add track status %d: %s", addRec.Code, addRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/music/playlists/"+playlistID, nil)
	getRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get playlist status %d: %s", getRec.Code, getRec.Body.String())
	}

	var playlist map[string]any
	if err := json.Unmarshal(getRec.Body.Bytes(), &playlist); err != nil {
		t.Fatalf("decode playlist: %v", err)
	}
	tracks, ok := playlist["tracks"].([]any)
	if !ok || len(tracks) != 1 {
		t.Fatalf("expected one track in playlist, got %#v", playlist["tracks"])
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/api/music/playlists/"+playlistID, nil)
	delRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("delete status %d: %s", delRec.Code, delRec.Body.String())
	}
}

func TestMusicFavoriteContract(t *testing.T) {
	srv, _ := newTestServer(t)
	trackID := "favorite-track-1"

	addReq := httptest.NewRequest(http.MethodPost, "/api/music/favorites/"+trackID, bytes.NewBufferString(`{
		"trackTitle":"Favorite Song",
		"artistName":"Artist",
		"albumId":"album-1",
		"albumTitle":"Album",
		"durationMs":180000,
		"coverArtId":"cover-1"
	}`))
	addRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(addRec, addReq)
	if addRec.Code != http.StatusNoContent {
		t.Fatalf("add favorite status %d: %s", addRec.Code, addRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/music/favorites", nil)
	listRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list favorites status %d: %s", listRec.Code, listRec.Body.String())
	}

	var payload struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Items) == 0 {
		t.Fatal("expected at least one favorite")
	}

	removeReq := httptest.NewRequest(http.MethodDelete, "/api/music/favorites/"+trackID, nil)
	removeRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(removeRec, removeReq)
	if removeRec.Code != http.StatusNoContent {
		t.Fatalf("remove favorite status %d: %s", removeRec.Code, removeRec.Body.String())
	}
}
