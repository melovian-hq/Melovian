// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestListDirectoriesRequiresSession(t *testing.T) {
	srv, _ := newAuthTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/filesystem/directories", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", rec.Code)
	}
}

func TestListDirectoriesWhenBrowseDisabled(t *testing.T) {
	srv, _ := newAuthTestServer(t)
	srv.cfg.LocalLibrary.Enabled = true
	srv.cfg.LocalLibrary.AllowCustomPath = false
	srv.cfg.LocalLibraryOverride = true

	cookie := setupAuthSession(t, srv)
	req := httptest.NewRequest(http.MethodGet, "/api/filesystem/directories", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when browse disabled, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestListDirectoriesListsChildren(t *testing.T) {
	srv, _ := newAuthTestServer(t)
	root := t.TempDir()
	child := filepath.Join(root, "Music")
	if err := os.Mkdir(child, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "readme.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".secret"), 0o750); err != nil {
		t.Fatal(err)
	}

	srv.cfg.LocalLibrary.Enabled = true
	srv.cfg.LocalLibrary.AllowCustomPath = true
	srv.cfg.LocalLibraryOverride = true
	srv.cfg.DataDir = root

	cookie := setupAuthSession(t, srv)
	req := httptest.NewRequest(http.MethodGet, "/api/filesystem/directories?path="+url.QueryEscape(root), nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Path    string `json:"path"`
		Entries []struct {
			Name string `json:"name"`
			Path string `json:"path"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Path != root {
		t.Fatalf("path=%q want %q", payload.Path, root)
	}
	if len(payload.Entries) != 1 || payload.Entries[0].Name != "Music" {
		t.Fatalf("entries=%+v", payload.Entries)
	}
}

func TestListDirectoriesRejectsOutsideBrowseRoots(t *testing.T) {
	srv, _ := newAuthTestServer(t)
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(outside, "Music"), 0o750); err != nil {
		t.Fatal(err)
	}

	srv.cfg.LocalLibrary.Enabled = true
	srv.cfg.LocalLibrary.AllowCustomPath = true
	srv.cfg.LocalLibraryOverride = true
	// DataDir stays the auth server temp dir, not `outside`.

	cookie := setupAuthSession(t, srv)
	req := httptest.NewRequest(http.MethodGet, "/api/filesystem/directories?path="+url.QueryEscape(outside), nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 outside roots, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func setupAuthSession(t *testing.T, srv *Server) *http.Cookie {
	t.Helper()
	setupBody := bytes.NewBufferString(`{"username":"admin","password":"password123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/setup", setupBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("setup status %d: %s", rec.Code, rec.Body.String())
	}
	cookie := authCookie(rec)
	if cookie == nil {
		t.Fatal("expected session cookie after setup")
	}
	return cookie
}
