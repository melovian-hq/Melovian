// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"melovian/internal/store"
)

func TestSharePublicPlaylistAndPassword(t *testing.T) {
	srv, db := newTestServer(t)

	dir := t.TempDir()
	trackPath := filepath.Join(dir, "song.mp3")
	if err := os.WriteFile(trackPath, []byte("share-audio"), 0o640); err != nil {
		t.Fatal(err)
	}
	libs := store.NewLocalLibraryStore(db)
	lib, err := libs.Create(store.CreateLocalLibraryInput{Name: "Lib", Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	trackStore := store.NewLocalTrackStore(db)
	track, err := trackStore.Upsert(store.UpsertLocalTrackInput{
		LibraryID: lib.ID,
		RelPath:   "song.mp3",
		AbsPath:   trackPath,
		Title:     "Song One",
		Artist:    "Artist",
		Format:    "mp3",
		Size:      11,
	})
	if err != nil {
		t.Fatal(err)
	}

	listen := store.NewListenStore(db)
	pl, err := listen.CreatePlaylist("local", "Road Mix")
	if err != nil {
		t.Fatal(err)
	}
	if err := listen.SetPlaylistTracks("local", pl.ID, []store.PlaylistTrack{
		{
			TrackID:    track.ID,
			TrackTitle: "Song One",
			ArtistName: "Artist",
			AlbumTitle: "Album",
			DurationMs: 180000,
		},
	}); err != nil {
		t.Fatal(err)
	}

	createBody := `{"resourceType":"playlist","resourceId":"` + pl.ID + `","description":"Road Mix","accessMode":"public"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/music/shares", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create share status %d body %s", createRec.Code, createRec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	token, _ := created["token"].(string)
	if token == "" {
		t.Fatal("missing token")
	}
	url, _ := created["url"].(string)
	if !strings.Contains(url, "/share/"+token) {
		t.Fatalf("share url %q missing /share/ path", url)
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/s/"+token, nil)
	detailRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(detailRec, detailReq)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("public detail status %d body %s", detailRec.Code, detailRec.Body.String())
	}
	var detail map[string]any
	if err := json.Unmarshal(detailRec.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	tracks, ok := detail["tracks"].([]any)
	if !ok || len(tracks) != 1 {
		t.Fatalf("expected 1 track, got %#v", detail["tracks"])
	}

	pwBody := `{"resourceType":"playlist","resourceId":"` + pl.ID + `","accessMode":"password","password":"secret"}`
	pwReq := httptest.NewRequest(http.MethodPost, "/api/music/shares", strings.NewReader(pwBody))
	pwReq.Header.Set("Content-Type", "application/json")
	pwRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(pwRec, pwReq)
	if pwRec.Code != http.StatusCreated {
		t.Fatalf("password share status %d body %s", pwRec.Code, pwRec.Body.String())
	}
	var pwShare map[string]any
	_ = json.Unmarshal(pwRec.Body.Bytes(), &pwShare)
	pwToken, _ := pwShare["token"].(string)

	deniedReq := httptest.NewRequest(http.MethodGet, "/s/"+pwToken, nil)
	deniedRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(deniedRec, deniedReq)
	if deniedRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 before unlock, got %d", deniedRec.Code)
	}

	unlockReq := httptest.NewRequest(http.MethodPost, "/s/"+pwToken+"/unlock", strings.NewReader(`{"password":"wrong"}`))
	unlockReq.Header.Set("Content-Type", "application/json")
	unlockRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(unlockRec, unlockReq)
	if unlockRec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password status %d", unlockRec.Code)
	}

	unlockOk := httptest.NewRequest(http.MethodPost, "/s/"+pwToken+"/unlock", strings.NewReader(`{"password":"secret"}`))
	unlockOk.Header.Set("Content-Type", "application/json")
	unlockOkRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(unlockOkRec, unlockOk)
	if unlockOkRec.Code != http.StatusOK {
		t.Fatalf("unlock status %d body %s", unlockOkRec.Code, unlockOkRec.Body.String())
	}
	cookies := unlockOkRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected unlock cookie")
	}

	authedReq := httptest.NewRequest(http.MethodGet, "/s/"+pwToken, nil)
	for _, c := range cookies {
		authedReq.AddCookie(c)
	}
	authedRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(authedRec, authedReq)
	if authedRec.Code != http.StatusOK {
		t.Fatalf("authed detail status %d body %s", authedRec.Code, authedRec.Body.String())
	}
}

func TestMediaTrackDownloadLocal(t *testing.T) {
	srv, db := newTestServer(t)
	dir := t.TempDir()
	libPath := filepath.Join(dir, "music")
	if err := os.MkdirAll(libPath, 0o750); err != nil {
		t.Fatal(err)
	}
	trackPath := filepath.Join(libPath, "song.mp3")
	payload := []byte("ID3fake-audio-bytes")
	if err := os.WriteFile(trackPath, payload, 0o640); err != nil {
		t.Fatal(err)
	}

	libs := store.NewLocalLibraryStore(db)
	lib, err := libs.Create(store.CreateLocalLibraryInput{Name: "Test", Path: libPath})
	if err != nil {
		t.Fatal(err)
	}
	tracks := store.NewLocalTrackStore(db)
	track, err := tracks.Upsert(store.UpsertLocalTrackInput{
		LibraryID: lib.ID,
		RelPath:   "song.mp3",
		AbsPath:   trackPath,
		Title:     "Song",
		Artist:    "Artist",
		Format:    "mp3",
		Size:      int64(len(payload)),
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/media/tracks/"+track.ID+"/download", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("download status %d body %s", rec.Code, rec.Body.String())
	}
	disposition := rec.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "attachment") {
		t.Fatalf("missing attachment disposition: %q", disposition)
	}
	if !bytes.Equal(rec.Body.Bytes(), payload) {
		t.Fatalf("body mismatch")
	}
}
