// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestStaticAssetCacheHeaders(t *testing.T) {
	root := fstest.MapFS{
		"index.html":           &fstest.MapFile{Data: []byte("<html>ok</html>")},
		"assets/app-abc123.js": &fstest.MapFile{Data: []byte("console.log(1)")},
		"favicon.svg":          &fstest.MapFile{Data: []byte("<svg/>")},
	}
	handler := StaticAssetHandler(root)

	cases := []struct {
		path string
		want string
	}{
		{path: "/", want: "no-cache"},
		{path: "/index.html", want: "no-cache"},
		{path: "/assets/app-abc123.js", want: "public, max-age=31536000, immutable"},
		{path: "/favicon.svg", want: "public, max-age=3600"},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if got := rec.Header().Get("Cache-Control"); got != tc.want {
			t.Fatalf("%s Cache-Control=%q want %q", tc.path, got, tc.want)
		}
	}
}

func TestStaticAssetHandlerServesIndex(t *testing.T) {
	root := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>shell</html>")},
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	StaticAssetHandler(fs.FS(root)).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if body := rec.Body.String(); body != "<html>shell</html>" {
		t.Fatalf("body %q", body)
	}
}
