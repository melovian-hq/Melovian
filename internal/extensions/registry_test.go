// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testZip(t *testing.T, manifest string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("demo-remote/melovian-extension.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(manifest)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func serveRegistry(t *testing.T, pkg []byte, sum string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	pkgURL := srv.URL + "/demo-remote-1.0.0.zip"
	mux.HandleFunc("/registry.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
  "version": 1,
  "generatedAt": "2026-09-14T00:00:00Z",
  "extensions": [{
    "id": "demo-remote",
    "name": "Demo Remote",
    "version": "1.0.0",
    "package": {"url": %q, "sha256": %q, "bytes": %d},
    "capabilities": {"script": false, "wasm": false, "styles": 0, "appTheme": false, "trackRules": 1, "playerHooks": 0},
    "audit": {"status": "pass", "warnings": []}
  }]
}`, pkgURL, sum, len(pkg))
	})
	mux.HandleFunc("/demo-remote-1.0.0.zip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(pkg)
	})
	return srv
}

func TestFetchRegistryAndInstall(t *testing.T) {
	pkg := testZip(t, `{"id":"demo-remote","name":"Demo Remote","version":"1.0.0"}`)
	sum := sha256.Sum256(pkg)
	srv := serveRegistry(t, pkg, hex.EncodeToString(sum[:]))
	defer srv.Close()
	t.Setenv("MELOVIAN_EXTENSION_REGISTRY_URL", srv.URL+"/registry.json")

	index, indexURL, err := FetchRegistry(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if indexURL != srv.URL+"/registry.json" {
		t.Fatalf("unexpected index url %q", indexURL)
	}
	if len(index.Extensions) != 1 || index.Extensions[0].ID != "demo-remote" {
		t.Fatalf("unexpected index %#v", index)
	}

	dir := t.TempDir()
	manifest, err := InstallFromRegistry(context.Background(), dir, "demo-remote")
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ID != "demo-remote" {
		t.Fatalf("unexpected manifest %#v", manifest)
	}
	items, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected installed extension, got %#v", items)
	}
}

func TestInstallFromRegistryChecksumMismatch(t *testing.T) {
	pkg := testZip(t, `{"id":"demo-remote","name":"Demo Remote","version":"1.0.0"}`)
	srv := serveRegistry(t, pkg, strings.Repeat("0", 64))
	defer srv.Close()
	t.Setenv("MELOVIAN_EXTENSION_REGISTRY_URL", srv.URL+"/registry.json")
	if _, err := InstallFromRegistry(context.Background(), t.TempDir(), "demo-remote"); err == nil {
		t.Fatal("expected checksum mismatch")
	}
}

func TestInstallFromRegistryMissingID(t *testing.T) {
	pkg := testZip(t, `{"id":"demo-remote","name":"Demo Remote","version":"1.0.0"}`)
	sum := sha256.Sum256(pkg)
	srv := serveRegistry(t, pkg, hex.EncodeToString(sum[:]))
	defer srv.Close()
	t.Setenv("MELOVIAN_EXTENSION_REGISTRY_URL", srv.URL+"/registry.json")
	if _, err := InstallFromRegistry(context.Background(), t.TempDir(), "nope"); err == nil {
		t.Fatal("expected unknown id to fail")
	}
}

func TestRemoteURLPolicy(t *testing.T) {
	for _, tc := range []struct {
		url  string
		want bool
	}{
		{"https://melovian-hq.github.io/Melovian-Extensions/registry.json", true},
		{"http://example.com/registry.json", false},
		{"http://127.0.0.1:8080/registry.json", true},
		{"http://localhost:9000/registry.json", true},
		{"http://[::1]:9000/registry.json", true},
		{"ftp://example.com/x", false},
		{"file:///etc/passwd", false},
		{"", false},
		{"not a url", false},
	} {
		if got := remoteURLOK(tc.url); got != tc.want {
			t.Errorf("remoteURLOK(%q) = %v, want %v", tc.url, got, tc.want)
		}
	}
}

func TestResolveRegistryAsset(t *testing.T) {
	base := "https://melovian-hq.github.io/Melovian-Extensions/registry.json"
	got := ResolveRegistryAsset(base, "extensions/podcast-style/assets/icon.svg")
	want := "https://melovian-hq.github.io/Melovian-Extensions/extensions/podcast-style/assets/icon.svg"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if ResolveRegistryAsset(base, "") != "" {
		t.Fatal("empty rel should produce empty url")
	}
}

// The registry builds zips with a store-method writer, fixed timestamp,
// and files under a top-level folder. This fixture came out of
// Melovian-Extensions tools/build.mjs and checks the installer accepts it.
func TestInstallRegistryBuiltZip(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "podcast-style-1.0.0.zip"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := InstallFromZip(t.TempDir(), data)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ID != "podcast-style" || manifest.Version != "1.0.0" {
		t.Fatalf("unexpected manifest %#v", manifest)
	}
}
