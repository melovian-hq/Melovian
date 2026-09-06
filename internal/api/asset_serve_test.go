// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCombinedHandlerServesDistAssets(t *testing.T) {
	t.Setenv("FRONTEND_DEVSERVER_URL", "")

	dist := filepath.Join("..", "..", "frontend", "dist")
	if _, err := os.Stat(dist); err != nil {
		t.Skip("frontend dist not built")
	}

	var jsPath, cssPath string
	entries, err := os.ReadDir(filepath.Join(dist, "assets"))
	if err != nil {
		t.Fatalf("read dist assets: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		switch {
		case jsPath == "" && strings.HasSuffix(name, ".js"):
			jsPath = "/assets/" + name
		case cssPath == "" && strings.HasSuffix(name, ".css"):
			cssPath = "/assets/" + name
		}
	}
	if jsPath == "" || cssPath == "" {
		t.Fatal("dist assets missing js or css bundle")
	}

	h := &CombinedHandler{
		API:    http.NotFoundHandler(),
		Assets: http.FileServer(http.Dir(dist)),
	}

	paths := []string{jsPath, cssPath, "/login", "/api/config", "/favicon.svg"}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		ct := rec.Header().Get("Content-Type")
		t.Logf("%s -> %d ct=%q len=%d", path, rec.Code, ct, rec.Body.Len())
		if path == "/api/config" {
			if rec.Code != http.StatusNotFound {
				t.Fatalf("%s: expected API passthrough 404, got %d", path, rec.Code)
			}
			continue
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d body=%.200q", path, rec.Code, rec.Body.String())
		}
		if path == jsPath {
			body := rec.Body.Bytes()
			if len(body) > 0 && body[0] == '<' {
				t.Fatalf("%s: got HTML instead of JS", path)
			}
			if ct != "" && !strings.Contains(ct, "javascript") && !strings.Contains(ct, "ecmascript") {
				t.Fatalf("%s: unexpected content type %q", path, ct)
			}
		}
		if path == "/login" {
			body := rec.Body.Bytes()
			if len(body) == 0 || body[0] != '<' {
				t.Fatalf("%s: expected SPA fallback HTML", path)
			}
		}
	}
}
