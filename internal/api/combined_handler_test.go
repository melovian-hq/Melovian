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

func TestNeedsSPAFallback(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/", want: false},
		{path: "/login", want: true},
		{path: "/music", want: true},
		{path: "/music/album/abc", want: true},
		{path: "/assets/index.js", want: false},
		{path: "/favicon.svg", want: false},
		{path: "/style.css", want: false},
	}

	for _, tc := range tests {
		if got := needsSPAFallback(tc.path); got != tc.want {
			t.Fatalf("needsSPAFallback(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestIsStaticAssetPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/assets/index.js", want: true},
		{path: "/wails/runtime.js", want: true},
		{path: "/@vite/client", want: true},
		{path: "/src/main.ts", want: true},
		{path: "/node_modules/.vite/deps/svelte.js", want: true},
		{path: "/favicon.svg", want: true},
		{path: "/login", want: false},
		{path: "/music", want: false},
	}

	for _, tc := range tests {
		if got := isStaticAssetPath(tc.path); got != tc.want {
			t.Fatalf("isStaticAssetPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestCombinedHandlerDevProxyPassthrough(t *testing.T) {
	t.Setenv("FRONTEND_DEVSERVER_URL", "http://127.0.0.1:9245")

	var gotPath string
	handler := &CombinedHandler{
		API: http.NotFoundHandler(),
		Assets: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = w.Write([]byte("export {};"))
		}),
	}

	req := httptest.NewRequest(http.MethodGet, "/@vite/client", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotPath != "/@vite/client" {
		t.Fatalf("assets path %q, want /@vite/client", gotPath)
	}
	if rec.Header().Get("Content-Type") != "application/javascript" {
		t.Fatalf("content type %q, want application/javascript", rec.Header().Get("Content-Type"))
	}
}

func TestCombinedHandlerSPAFallback(t *testing.T) {
	t.Setenv("FRONTEND_DEVSERVER_URL", "")
	assets := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>app</html>")},
	}

	handler := &CombinedHandler{
		API: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		}),
		Assets: http.FileServer(http.FS(assets)),
	}

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); body != "<html>app</html>" {
		t.Fatalf("body %q, want index.html content", body)
	}
}

func TestCombinedHandlerAPIPassthrough(t *testing.T) {
	handler := &CombinedHandler{
		API: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
		Assets: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d, want 204", rec.Code)
	}
}

func TestCombinedHandlerStaticAsset(t *testing.T) {
	t.Setenv("FRONTEND_DEVSERVER_URL", "")
	assets := fstest.MapFS{
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('ok')")},
	}

	handler := &CombinedHandler{
		API:    http.NotFoundHandler(),
		Assets: http.StripPrefix("/", http.FileServer(http.FS(assets))),
	}

	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	if rec.Body.String() != "console.log('ok')" {
		t.Fatalf("unexpected body %q", rec.Body.String())
	}
}

func TestCombinedHandlerBindingsAsset(t *testing.T) {
	t.Setenv("FRONTEND_DEVSERVER_URL", "")

	var gotPath string
	handler := &CombinedHandler{
		API: http.NotFoundHandler(),
		Assets: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
		}),
	}

	req := httptest.NewRequest(http.MethodGet, "/bindings/melovian/services/index.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotPath != "/bindings/melovian/services/index.js" {
		t.Fatalf("assets path %q, want /bindings/melovian/services/index.js", gotPath)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
}

var _ fs.FS = fstest.MapFS{}
