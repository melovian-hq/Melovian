// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"melovian/internal/appconfig"
	ext "melovian/internal/extensions"
)

func newTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	dataDir := t.TempDir()
	h := New(appconfig.Config{DataDir: dataDir})
	mux := http.NewServeMux()
	h.Register(mux)
	return httptest.NewServer(mux), dataDir
}

// plantExtension drops a manifest folder under the data dir so List
// picks it up as installed.
func plantExtension(t *testing.T, dataDir string, manifestJSON string) {
	t.Helper()
	dir := filepath.Join(ext.ExtensionsDir(dataDir), "test-ext")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ext.ManifestName), []byte(manifestJSON), 0o644); err != nil {
		t.Fatal(err)
	}
}

const testExtManifest = `{
  "id": "test-ext",
  "name": "Test",
  "version": "1.0.0",
  "settings": [
    {"key": "muted", "type": "boolean", "default": false},
    {"key": "accent", "type": "choice", "options": ["red", "blue"], "default": "red"}
  ]
}`

func TestGetExtensionSettings(t *testing.T) {
	srv, dataDir := newTestServer(t)
	defer srv.Close()
	plantExtension(t, dataDir, testExtManifest)

	resp, err := http.Get(srv.URL + "/api/extensions/test-ext/settings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body struct {
		Schema   []ext.SettingField `json:"schema"`
		Settings map[string]any     `json:"settings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Schema) != 2 {
		t.Fatalf("schema = %v", body.Schema)
	}
	if body.Settings["muted"] != false || body.Settings["accent"] != "red" {
		t.Fatalf("defaults = %v", body.Settings)
	}
}

func TestPutExtensionSettings(t *testing.T) {
	srv, dataDir := newTestServer(t)
	defer srv.Close()
	plantExtension(t, dataDir, testExtManifest)

	req, _ := http.NewRequest(http.MethodPut,
		srv.URL+"/api/extensions/test-ext/settings",
		strings.NewReader(`{"settings": {"muted": true, "accent": "blue"}}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var body struct {
		Settings map[string]any `json:"settings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Settings["muted"] != true || body.Settings["accent"] != "blue" {
		t.Fatalf("settings = %v", body.Settings)
	}

	// Invalid values are rejected and do not clobber stored state.
	req, _ = http.NewRequest(http.MethodPut,
		srv.URL+"/api/extensions/test-ext/settings",
		strings.NewReader(`{"settings": {"accent": "green"}}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid choice status = %d", resp.StatusCode)
	}
	if got := ext.LoadSettings(dataDir, mustFind(t, dataDir)); got["accent"] != "blue" {
		t.Fatalf("rejected save clobbered state: %v", got)
	}
}

func TestExtensionSettingsNotInstalled(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/extensions/ghost/settings")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestPutExtensionSettingsBadJSON(t *testing.T) {
	srv, dataDir := newTestServer(t)
	defer srv.Close()
	plantExtension(t, dataDir, testExtManifest)

	req, _ := http.NewRequest(http.MethodPut,
		srv.URL+"/api/extensions/test-ext/settings",
		strings.NewReader(`{oops`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func mustFind(t *testing.T, dataDir string) ext.Manifest {
	t.Helper()
	entry, err := ext.FindInstalled(dataDir, "test-ext")
	if err != nil {
		t.Fatal(err)
	}
	return entry.Manifest
}
