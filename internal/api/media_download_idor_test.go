// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"melovian/internal/store"
)

func TestMediaTrackDownloadDeniesOtherUsersLibrary(t *testing.T) {
	srv, _ := newAuthTestServer(t)
	alice, err := srv.auth.CreateUser("alice", "password123")
	if err != nil {
		t.Fatalf("CreateUser alice: %v", err)
	}
	bob, err := srv.auth.CreateUser("bob", "password123")
	if err != nil {
		t.Fatalf("CreateUser bob: %v", err)
	}

	dir := t.TempDir()
	libPath := filepath.Join(dir, "music")
	if err := os.MkdirAll(libPath, 0o750); err != nil {
		t.Fatal(err)
	}
	trackPath := filepath.Join(libPath, "song.mp3")
	payload := []byte("ID3owner-only-bytes")
	if err := os.WriteFile(trackPath, payload, 0o640); err != nil {
		t.Fatal(err)
	}

	lib, err := srv.localLibraries.CreateForUser(alice.ID, store.CreateLocalLibraryInput{
		Name: "Alice Lib",
		Path: libPath,
	})
	if err != nil {
		t.Fatalf("CreateForUser: %v", err)
	}
	track, err := srv.localTracks.Upsert(store.UpsertLocalTrackInput{
		LibraryID: lib.ID,
		RelPath:   "song.mp3",
		AbsPath:   trackPath,
		Title:     "Song",
		Artist:    "Artist",
		Format:    "mp3",
		Size:      int64(len(payload)),
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	bobTok, _, err := srv.auth.CreateSession(bob.ID)
	if err != nil {
		t.Fatalf("bob session: %v", err)
	}
	bobReq := httptest.NewRequest(http.MethodGet, "/api/media/tracks/"+track.ID+"/download", nil)
	bobReq.AddCookie(&http.Cookie{Name: store.SessionCookieName(), Value: bobTok})
	bobRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(bobRec, bobReq)
	if bobRec.Code != http.StatusNotFound {
		t.Fatalf("bob download status %d body %s", bobRec.Code, bobRec.Body.String())
	}

	aliceTok, _, err := srv.auth.CreateSession(alice.ID)
	if err != nil {
		t.Fatalf("alice session: %v", err)
	}
	aliceReq := httptest.NewRequest(http.MethodGet, "/api/media/tracks/"+track.ID+"/download", nil)
	aliceReq.AddCookie(&http.Cookie{Name: store.SessionCookieName(), Value: aliceTok})
	aliceRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(aliceRec, aliceReq)
	if aliceRec.Code != http.StatusOK {
		t.Fatalf("alice download status %d body %s", aliceRec.Code, aliceRec.Body.String())
	}
	if got := aliceRec.Body.Bytes(); string(got) != string(payload) {
		t.Fatalf("alice body mismatch")
	}
}
