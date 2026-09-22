// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"melovian/internal/api/realtime"
	"melovian/internal/seo"
	"melovian/internal/store"
)

var pngMagic = []byte{0x89, 'P', 'N', 'G'}

func createShare(t *testing.T, srv *Server, input store.CreateShareInput) store.Share {
	t.Helper()
	share, err := srv.shares.Create(input)
	if err != nil {
		t.Fatalf("create share: %v", err)
	}
	return share
}

func TestPageMetaSharePublic(t *testing.T) {
	srv, _ := newTestServer(t)
	share := createShare(t, srv, store.CreateShareInput{
		UserID:       "local",
		ResourceType: "song",
		ResourceID:   "ext-song-1",
		Description:  "Never Gonna Give You Up",
	})

	fallback := seo.ForPath("/share/" + share.Token)
	page := srv.PageMeta(httptest.NewRequest(http.MethodGet, "/share/"+share.Token, nil), "/share/"+share.Token, fallback)

	if page.Title != "Never Gonna Give You Up" {
		t.Fatalf("title = %q, want share description", page.Title)
	}
	if page.Image != "/og/share/"+share.Token+".png" {
		t.Fatalf("image = %q", page.Image)
	}
	if page.Type != "music.song" {
		t.Fatalf("type = %q, want music.song", page.Type)
	}
	if !page.Index {
		t.Fatal("public share should be indexable")
	}
}

func TestPageMetaShareGatedDoesNotLeak(t *testing.T) {
	srv, _ := newTestServer(t)
	share := createShare(t, srv, store.CreateShareInput{
		UserID:       "local",
		ResourceType: "song",
		ResourceID:   "ext-song-1",
		Description:  "Secret Title",
		AccessMode:   store.ShareAccessPassword,
		Password:     "hunter2",
	})

	fallback := seo.ForPath("/share/" + share.Token)
	page := srv.PageMeta(httptest.NewRequest(http.MethodGet, "/share/"+share.Token, nil), "/share/"+share.Token, fallback)

	if strings.Contains(page.Title, "Secret") || strings.Contains(page.Description, "Secret") {
		t.Fatalf("gated share meta leaked title: %+v", page)
	}
	if page.Index {
		t.Fatal("gated share should not be indexable")
	}
	if strings.Contains(page.Image, share.Token) {
		t.Fatalf("gated share should not expose an entity image: %q", page.Image)
	}
}

func TestPageMetaShareUnknownToken(t *testing.T) {
	srv, _ := newTestServer(t)
	fallback := seo.ForPath("/share/nope")
	page := srv.PageMeta(httptest.NewRequest(http.MethodGet, "/share/nope", nil), "/share/nope", fallback)
	if page.Index {
		t.Fatal("unknown share should not be indexable")
	}
}

func TestPageMetaLibraryRouteUnchanged(t *testing.T) {
	srv, _ := newTestServer(t)
	fallback := seo.ForPath("/music/album/abc")
	page := srv.PageMeta(httptest.NewRequest(http.MethodGet, "/music/album/abc", nil), "/music/album/abc", fallback)
	if page.Title != fallback.Title {
		t.Fatalf("library route meta should be untouched, got %q", page.Title)
	}
}

func TestPageMetaListenSession(t *testing.T) {
	srv, _ := newTestServer(t)
	host := &realtime.WSClient{Send: make(chan []byte, 16)}
	host.SetDeviceID("host-1")
	host.SetScopeKey("server:local")
	srv.devices.Register(host, "Host", "Linux")
	sessionID := srv.devices.CreateSession(host)
	if sessionID == "" {
		t.Fatal("no session created")
	}
	srv.devices.UpdatePlayback(host, realtime.PlaybackSnapshot{
		TrackID:    "trk-9",
		TrackTitle: "Midnight Rendezvous",
		ArtistName: "Some Artist",
	})
	_, token, ok := srv.devices.EnsureInviteToken("server:local", "host-1")
	if !ok || token == "" {
		t.Fatal("EnsureInviteToken failed")
	}

	fallback := seo.ForPath("/listen/" + token)
	page := srv.PageMeta(httptest.NewRequest(http.MethodGet, "/listen/"+token, nil), "/listen/"+token, fallback)
	if page.Index {
		t.Fatal("listen invite tokens are credentials; they must not be indexed")
	}
	if page.Image != "/og/listen/"+token+".png" {
		t.Fatalf("image = %q", page.Image)
	}
	if !strings.Contains(page.Description, "Midnight Rendezvous") {
		t.Fatalf("description = %q, want now-playing info", page.Description)
	}

	bad := srv.PageMeta(httptest.NewRequest(http.MethodGet, "/listen/nope", nil), "/listen/nope", seo.ForPath("/listen/nope"))
	if bad.Index {
		t.Fatal("dead invite should not be indexable")
	}
}

func TestOGImageSharePublic(t *testing.T) {
	srv, _ := newTestServer(t)
	share := createShare(t, srv, store.CreateShareInput{
		UserID:       "local",
		ResourceType: "playlist",
		ResourceID:   "pl-missing",
		Description:  "Late Night Driving Mix",
	})

	req := httptest.NewRequest(http.MethodGet, "/og/share/"+share.Token+".png", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("content type %q, want image/png", ct)
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), pngMagic) {
		t.Fatal("body is not a PNG")
	}
	if rec.Header().Get("Cache-Control") == "" {
		t.Fatal("missing Cache-Control")
	}

	rec2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/og/share/"+share.Token+".png", nil))
	if rec2.Code != http.StatusOK || rec2.Header().Get("X-Cache") != "HIT" {
		t.Fatalf("second request should hit cache, code %d x-cache %q", rec2.Code, rec2.Header().Get("X-Cache"))
	}
}

func TestOGImageShareGated(t *testing.T) {
	srv, _ := newTestServer(t)
	share := createShare(t, srv, store.CreateShareInput{
		UserID:       "local",
		ResourceType: "song",
		ResourceID:   "ext-song-1",
		Description:  "Secret Title",
		AccessMode:   store.ShareAccessPassword,
		Password:     "hunter2",
	})

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/og/share/"+share.Token+".png", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("gated share should still render a generic card, got %d", rec.Code)
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), pngMagic) {
		t.Fatal("body is not a PNG")
	}
}

func TestOGImageNotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	for _, path := range []string{
		"/og/share/nope.png",
		"/og/listen/nope.png",
		"/og/unknown/x.png",
		"/og/share/",
		"/og/",
	} {
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s: status %d, want 404", path, rec.Code)
		}
	}
}

func TestOGImageMethodNotAllowed(t *testing.T) {
	srv, _ := newTestServer(t)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/og/share/x.png", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d, want 405", rec.Code)
	}
}

func TestCombinedHandlerShareMetaInjection(t *testing.T) {
	t.Setenv("FRONTEND_DEVSERVER_URL", "")
	srv, _ := newTestServer(t)
	share := createShare(t, srv, store.CreateShareInput{
		UserID:       "local",
		ResourceType: "album",
		ResourceID:   "ext-alb-1",
		Description:  "Blue Lines",
	})
	assets := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html><head><title>Shell</title></head><body>app</body></html>")},
	}
	handler := &CombinedHandler{
		API:    http.NotFoundHandler(),
		Assets: http.FileServer(http.FS(assets)),
		Shell:  assets,
		Meta:   srv.PageMeta,
	}

	req := httptest.NewRequest(http.MethodGet, "/share/"+share.Token, nil)
	req.Host = "music.example"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"<title>Blue Lines · Melovian</title>",
		`property="og:title" content="Blue Lines · Melovian"`,
		`property="og:type" content="music.album"`,
		`property="og:image" content="http://music.example/og/share/` + share.Token + `.png"`,
		`rel="canonical" href="http://music.example/share/` + share.Token + `"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}
