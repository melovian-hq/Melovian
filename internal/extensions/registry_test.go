// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
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

	index, indexURL, verified, err := FetchRegistry(context.Background())
	if verified {
		t.Fatal("unsigned test registry must not report verified")
	}
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

func signedRegistryServer(t *testing.T, pkg []byte) (*httptest.Server, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(pkg)
	pkgSig := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, pkg))
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	pkgURL := srv.URL + "/demo-remote-1.0.0.zip"
	indexJSON := fmt.Sprintf(`{
  "version": 1,
  "generatedAt": "2026-09-14T00:00:00Z",
  "extensions": [{
    "id": "demo-remote",
    "name": "Demo Remote",
    "version": "1.0.0",
    "package": {"url": %q, "sha256": %q, "bytes": %d, "signature": %q},
    "capabilities": {"script": false, "wasm": false, "styles": 0, "appTheme": false, "trackRules": 1, "playerHooks": 0},
    "audit": {"status": "pass", "warnings": []}
  }]
}`, pkgURL, hex.EncodeToString(sum[:]), len(pkg), pkgSig)
	mux.HandleFunc("/registry.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(indexJSON))
	})
	mux.HandleFunc("/registry.sig", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(base64.StdEncoding.EncodeToString(
			ed25519.Sign(priv, []byte(indexJSON)))))
	})
	mux.HandleFunc("/demo-remote-1.0.0.zip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(pkg)
	})
	t.Setenv("MELOVIAN_EXTENSION_REGISTRY_URL", srv.URL+"/registry.json")
	t.Setenv("MELOVIAN_EXTENSION_REGISTRY_KEY", hex.EncodeToString(pub))
	return srv, priv
}

func TestSignedRegistryInstalls(t *testing.T) {
	pkg := testZip(t, `{"id":"demo-remote","name":"Demo Remote","version":"1.0.0"}`)
	srv, _ := signedRegistryServer(t, pkg)
	defer srv.Close()

	_, _, verified, err := FetchRegistry(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Fatal("expected signature verification")
	}
	if _, err := InstallFromRegistry(context.Background(), t.TempDir(), "demo-remote"); err != nil {
		t.Fatal(err)
	}
}

func TestSignedRegistryMissingSignatureFails(t *testing.T) {
	pkg := testZip(t, `{"id":"demo-remote","name":"Demo Remote","version":"1.0.0"}`)
	srv, _ := signedRegistryServer(t, pkg)
	defer srv.Close()

	// A registry with a configured key but no registry.sig must fail closed.
	mux2 := http.NewServeMux()
	mux2.HandleFunc("/registry.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":1,"extensions":[]}`))
	})
	srv2 := httptest.NewServer(mux2)
	defer srv2.Close()
	t.Setenv("MELOVIAN_EXTENSION_REGISTRY_URL", srv2.URL+"/registry.json")
	if _, _, _, err := FetchRegistry(context.Background()); err == nil {
		t.Fatal("expected failure without registry.sig")
	}
}

func TestSignedRegistryTamperedIndexFails(t *testing.T) {
	pkg := testZip(t, `{"id":"demo-remote","name":"Demo Remote","version":"1.0.0"}`)
	srv, priv := signedRegistryServer(t, pkg)
	defer srv.Close()

	// Serve a tampered index alongside the signature of the original one.
	// The signature cannot match the new bytes, so verification must fail.
	origSig := ed25519.Sign(priv, []byte(`{"version":1,"extensions":[]}`))
	mux2 := http.NewServeMux()
	mux2.HandleFunc("/registry.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":1,"extensions":[{"id":"evil","name":"E","version":"9.9.9"}]}`))
	})
	mux2.HandleFunc("/registry.sig", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(base64.StdEncoding.EncodeToString(origSig)))
	})
	srv2 := httptest.NewServer(mux2)
	defer srv2.Close()
	t.Setenv("MELOVIAN_EXTENSION_REGISTRY_URL", srv2.URL+"/registry.json")
	if _, _, _, err := FetchRegistry(context.Background()); err == nil {
		t.Fatal("expected signature failure for tampered index")
	}
}

func TestSignedRegistryBadPackageSignatureFails(t *testing.T) {
	pkg := testZip(t, `{"id":"demo-remote","name":"Demo Remote","version":"1.0.0"}`)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(pkg)
	// Sign other bytes so the stored signature cannot match the package.
	bogusSig := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, []byte("other")))
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	pkgURL := srv.URL + "/demo-remote-1.0.0.zip"
	indexJSON := fmt.Sprintf(`{
  "version": 1,
  "generatedAt": "2026-09-14T00:00:00Z",
  "extensions": [{
    "id": "demo-remote",
    "name": "Demo Remote",
    "version": "1.0.0",
    "package": {"url": %q, "sha256": %q, "bytes": %d, "signature": %q},
    "capabilities": {"script": false, "wasm": false, "styles": 0, "appTheme": false, "trackRules": 1, "playerHooks": 0},
    "audit": {"status": "pass", "warnings": []}
  }]
}`, pkgURL, hex.EncodeToString(sum[:]), len(pkg), bogusSig)
	mux.HandleFunc("/registry.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(indexJSON))
	})
	mux.HandleFunc("/registry.sig", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(base64.StdEncoding.EncodeToString(
			ed25519.Sign(priv, []byte(indexJSON)))))
	})
	mux.HandleFunc("/demo-remote-1.0.0.zip", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(pkg)
	})
	defer srv.Close()
	t.Setenv("MELOVIAN_EXTENSION_REGISTRY_URL", srv.URL+"/registry.json")
	t.Setenv("MELOVIAN_EXTENSION_REGISTRY_KEY", hex.EncodeToString(pub))
	if _, err := InstallFromRegistry(context.Background(), t.TempDir(), "demo-remote"); err == nil {
		t.Fatal("expected package signature failure")
	}
}

func TestInstallFromRegistryDowngradeBlocked(t *testing.T) {
	pkg := testZip(t, `{"id":"demo-remote","name":"Demo Remote","version":"1.0.0"}`)
	sum := sha256.Sum256(pkg)
	srv := serveRegistry(t, pkg, hex.EncodeToString(sum[:]))
	defer srv.Close()
	t.Setenv("MELOVIAN_EXTENSION_REGISTRY_URL", srv.URL+"/registry.json")
	dir := t.TempDir()
	if _, err := InstallFromRegistry(context.Background(), dir, "demo-remote"); err != nil {
		t.Fatal(err)
	}
	// Registry still serves 1.0.0 but a newer copy is installed.
	newer := testZip(t, `{"id":"demo-remote","name":"Demo Remote","version":"2.0.0"}`)
	if _, err := InstallFromZip(dir, newer); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallFromRegistry(context.Background(), dir, "demo-remote"); err == nil {
		t.Fatal("expected downgrade to be blocked")
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

// Packages may ship a .ts source next to the compiled .js the manifest
// points at. The installer accepts both file types and the runtime only
// executes the compiled script.
func TestInstallRegistryBuiltZipWithTypeScriptSource(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "genre-palette-1.1.0.zip"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	manifest, err := InstallFromZip(dir, data)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Script != "script.js" {
		t.Fatalf("expected compiled script.js, got %q", manifest.Script)
	}
	if _, err := os.Stat(filepath.Join(dir, "extensions", "genre-palette", "script.ts")); err != nil {
		t.Fatal("expected script.ts source to be installed alongside script.js")
	}
}
