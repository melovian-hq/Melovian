// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"melovian/internal/store"
)

// Guarantee: a public share token only exposes tracks and covers that belong to
// the shared resource and the share owner's libraries. Hostile track/cover ids
// outside that set must be refused.

func TestShareTrackIDORDenied(t *testing.T) {
	srv, db := newTestServer(t)

	ownerDir := t.TempDir()
	victimDir := t.TempDir()
	ownerFile := filepath.Join(ownerDir, "owner.mp3")
	victimFile := filepath.Join(victimDir, "victim.mp3")
	if err := os.WriteFile(ownerFile, []byte("owner-audio"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(victimFile, []byte("VICTIM-SECRET-AUDIO"), 0o640); err != nil {
		t.Fatal(err)
	}

	libs := store.NewLocalLibraryStore(db)
	ownerLib, err := libs.CreateForUser("owner", store.CreateLocalLibraryInput{Name: "Owner", Path: ownerDir})
	if err != nil {
		t.Fatal(err)
	}
	victimLib, err := libs.CreateForUser("victim", store.CreateLocalLibraryInput{Name: "Victim", Path: victimDir})
	if err != nil {
		t.Fatal(err)
	}

	tracks := store.NewLocalTrackStore(db)
	ownerTrack, err := tracks.Upsert(store.UpsertLocalTrackInput{
		LibraryID: ownerLib.ID,
		RelPath:   "owner.mp3",
		AbsPath:   ownerFile,
		Title:     "Owner Song",
		Artist:    "Owner",
		Format:    "mp3",
		Size:      11,
	})
	if err != nil {
		t.Fatal(err)
	}
	victimTrack, err := tracks.Upsert(store.UpsertLocalTrackInput{
		LibraryID: victimLib.ID,
		RelPath:   "victim.mp3",
		AbsPath:   victimFile,
		Title:     "Victim Song",
		Artist:    "Victim",
		Format:    "mp3",
		Size:      18,
	})
	if err != nil {
		t.Fatal(err)
	}

	listen := store.NewListenStore(db)
	pl, err := listen.CreatePlaylist("user:owner", "Attack Mix")
	if err != nil {
		t.Fatal(err)
	}
	// Attacker plants a foreign track id into their playlist snapshot.
	if err := listen.SetPlaylistTracks("user:owner", pl.ID, []store.PlaylistTrack{
		{
			TrackID:    ownerTrack.ID,
			TrackTitle: "Owner Song",
			ArtistName: "Owner",
			DurationMs: 1000,
		},
		{
			TrackID:    victimTrack.ID,
			TrackTitle: "Victim Song",
			ArtistName: "Victim",
			DurationMs: 1000,
		},
	}); err != nil {
		t.Fatal(err)
	}

	share, err := store.NewShareStore(db).Create(store.CreateShareInput{
		UserID:       "owner",
		ResourceType: "playlist",
		ResourceID:   pl.ID,
		AccessMode:   store.ShareAccessPublic,
		OwnerScope:   "user:owner",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Owner track in share owner's library may stream.
	okReq := httptest.NewRequest(http.MethodGet, "/s/"+share.Token+"/tracks/"+ownerTrack.ID+"/stream", nil)
	okRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(okRec, okReq)
	if okRec.Code != http.StatusOK {
		t.Fatalf("owner track stream status %d body %s", okRec.Code, okRec.Body.String())
	}
	if got := okRec.Body.String(); got != "owner-audio" {
		t.Fatalf("owner body %q", got)
	}

	// Foreign library track must not stream even if listed on the playlist.
	badReq := httptest.NewRequest(http.MethodGet, "/s/"+share.Token+"/tracks/"+victimTrack.ID+"/stream", nil)
	badRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(badRec, badReq)
	if badRec.Code == http.StatusOK && strings.Contains(badRec.Body.String(), "VICTIM-SECRET-AUDIO") {
		t.Fatalf("SHARE_TRACK_IDOR_PROVED: foreign track streamed via public share")
	}
	if badRec.Code == http.StatusOK {
		t.Fatalf("expected deny for foreign track, got 200 body %q", badRec.Body.String())
	}
	t.Log("SHARE_TRACK_IDOR_DENIED_PROVED")
}

func TestShareCoverIDORDenied(t *testing.T) {
	srv, db := newTestServer(t)

	ownerDir := t.TempDir()
	ownerShared := filepath.Join(ownerDir, "shared.mp3")
	ownerSecret := filepath.Join(ownerDir, "secret.mp3")
	_ = os.WriteFile(ownerShared, []byte("aaa"), 0o640)
	_ = os.WriteFile(ownerSecret, []byte("bbb"), 0o640)

	libs := store.NewLocalLibraryStore(db)
	ownerLib, _ := libs.CreateForUser("owner", store.CreateLocalLibraryInput{Name: "O", Path: ownerDir})
	tracks := store.NewLocalTrackStore(db)
	sharedTrack, _ := tracks.Upsert(store.UpsertLocalTrackInput{
		LibraryID: ownerLib.ID, RelPath: "shared.mp3", AbsPath: ownerShared, Title: "Shared", Format: "mp3", Size: 3,
	})
	secretTrack, _ := tracks.Upsert(store.UpsertLocalTrackInput{
		LibraryID: ownerLib.ID, RelPath: "secret.mp3", AbsPath: ownerSecret, Title: "Secret", Format: "mp3", Size: 3,
	})

	listen := store.NewListenStore(db)
	pl, _ := listen.CreatePlaylist("user:owner", "Cover Mix")
	_ = listen.SetPlaylistTracks("user:owner", pl.ID, []store.PlaylistTrack{
		{TrackID: sharedTrack.ID, TrackTitle: "Shared", DurationMs: 1000, CoverArtID: sharedTrack.ID},
	})
	share, err := store.NewShareStore(db).Create(store.CreateShareInput{
		UserID: "owner", ResourceType: "playlist", ResourceID: pl.ID,
		AccessMode: store.ShareAccessPublic, OwnerScope: "user:owner",
	})
	if err != nil {
		t.Fatal(err)
	}

	if srv.shareCoverAllowed(share, secretTrack.ID) {
		t.Fatalf("SHARE_COVER_IDOR_PROVED: non-shared cover id allowed")
	}
	if !srv.shareCoverAllowed(share, sharedTrack.ID) {
		t.Fatalf("shared track cover should be allowed")
	}

	req := httptest.NewRequest(http.MethodGet, "/s/"+share.Token+"/cover?id="+secretTrack.ID, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("SHARE_COVER_IDOR_PROVED: non-shared cover served status %d", rec.Code)
	}
	t.Log("SHARE_COVER_IDOR_DENIED_PROVED")
}

func TestShareLegacyStreamTrackSwapDenied(t *testing.T) {
	srv, db := newTestServer(t)
	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.mp3")
	bPath := filepath.Join(dir, "b.mp3")
	_ = os.WriteFile(aPath, []byte("audio-a"), 0o640)
	_ = os.WriteFile(bPath, []byte("audio-b-SECRET"), 0o640)

	libs := store.NewLocalLibraryStore(db)
	lib, _ := libs.CreateForUser("owner", store.CreateLocalLibraryInput{Name: "L", Path: dir})
	tracks := store.NewLocalTrackStore(db)
	a, _ := tracks.Upsert(store.UpsertLocalTrackInput{
		LibraryID: lib.ID, RelPath: "a.mp3", AbsPath: aPath, Title: "A", Format: "mp3", Size: 7,
	})
	b, _ := tracks.Upsert(store.UpsertLocalTrackInput{
		LibraryID: lib.ID, RelPath: "b.mp3", AbsPath: bPath, Title: "B", Format: "mp3", Size: 14,
	})

	share, err := store.NewShareStore(db).Create(store.CreateShareInput{
		UserID: "owner", ResourceType: "song", ResourceID: a.ID,
		AccessMode: store.ShareAccessPublic, OwnerScope: "user:owner",
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/s/"+share.Token+"/stream?trackId="+b.ID, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), "audio-b-SECRET") {
		t.Fatalf("legacy stream trackId swap allowed")
	}
	if rec.Code == http.StatusOK {
		t.Fatalf("expected deny, got 200")
	}
	t.Log("SHARE_LEGACY_TRACK_SWAP_DENIED_PROVED")
}

func TestShareAccessModeUnknownDenied(t *testing.T) {
	srv, _ := newTestServer(t)
	share := store.Share{
		Token:        "tok",
		AccessMode:   "not-a-real-mode",
		PasswordHash: "",
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if srv.shareAccessAllowed(req, share) {
		t.Fatalf("unknown access mode must deny, got allow")
	}
	t.Log("SHARE_UNKNOWN_MODE_DENIED_PROVED")
}

func TestShareRestrictedRequiresLogin(t *testing.T) {
	srv, db := newAuthTestServer(t)
	listen := store.NewListenStore(db)
	pl, err := listen.CreatePlaylist("user:owner", "Private")
	if err != nil {
		t.Fatal(err)
	}
	share, err := store.NewShareStore(db).Create(store.CreateShareInput{
		UserID: "owner", ResourceType: "playlist", ResourceID: pl.ID,
		AccessMode: store.ShareAccessRestricted, OwnerScope: "user:owner",
		RecipientIDs: []string{"recipient"},
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/s/"+share.Token, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["requiresLogin"] != true {
		t.Fatalf("expected requiresLogin, got %#v", body)
	}
	t.Log("SHARE_RESTRICTED_LOGIN_REQUIRED_PROVED")
}
