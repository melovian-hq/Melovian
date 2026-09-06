// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonicserver

import (
	"context"
	"crypto/md5" //#nosec G501 -- Subsonic token auth requires MD5 per protocol spec
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"melovian/internal/localmusic"
	"melovian/internal/store"
)

type stubProvider struct{}

func (stubProvider) Catalog(context.Context, string) (localmusic.Catalog, error) {
	return localmusic.Catalog{}, nil
}

func (stubProvider) Track(context.Context, string, string) (store.LocalTrack, store.LocalLibrary, error) {
	return store.LocalTrack{}, store.LocalLibrary{}, nil
}

func (stubProvider) Cover(context.Context, string, string) ([]byte, string, error) {
	return nil, "", nil
}

func (stubProvider) StreamPath(context.Context, string, store.LocalTrack, store.LocalLibrary) (string, error) {
	return "", nil
}

type openAuth struct{}

func (openAuth) AuthRequired() bool                          { return false }
func (openAuth) Authenticate(string, string) (string, error) { return "", nil }
func (openAuth) AuthenticateToken(string, string, string) (string, error) {
	return "", nil
}

func TestPingJSON(t *testing.T) {
	server := New(stubProvider{}, openAuth{}, true)
	req := httptest.NewRequest(http.MethodGet, "/rest/ping.view?f=json", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ok") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestVerifyToken(t *testing.T) {
	password := "sesame"
	salt := "c19bede9c58ae026"
	sum := md5.Sum([]byte(password + salt)) //#nosec G401 -- Subsonic API mandated token = md5(password + salt)
	token := hex.EncodeToString(sum[:])
	if !VerifyToken(password, token, salt) {
		t.Fatal("expected token verification to pass")
	}
}

func TestDisabledServer(t *testing.T) {
	server := New(stubProvider{}, openAuth{}, false)
	req := httptest.NewRequest(http.MethodGet, "/rest/ping.view?f=json", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
