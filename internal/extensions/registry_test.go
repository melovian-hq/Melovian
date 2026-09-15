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
	"encoding/json"
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

	index, indexURL, verified, err := FetchRegistry(context.Background(), t.TempDir())
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
	manifest, err := InstallFromRegistry(context.Background(), dir, "demo-remote", "")
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
	if _, err := InstallFromRegistry(context.Background(), t.TempDir(), "demo-remote", ""); err == nil {
		t.Fatal("expected checksum mismatch")
	}
}

func TestInstallFromRegistryMissingID(t *testing.T) {
	pkg := testZip(t, `{"id":"demo-remote","name":"Demo Remote","version":"1.0.0"}`)
	sum := sha256.Sum256(pkg)
	srv := serveRegistry(t, pkg, hex.EncodeToString(sum[:]))
	defer srv.Close()
	t.Setenv("MELOVIAN_EXTENSION_REGISTRY_URL", srv.URL+"/registry.json")
	if _, err := InstallFromRegistry(context.Background(), t.TempDir(), "nope", ""); err == nil {
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

	_, _, verified, err := FetchRegistry(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Fatal("expected signature verification")
	}
	if _, err := InstallFromRegistry(context.Background(), t.TempDir(), "demo-remote", ""); err != nil {
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
	if _, _, _, err := FetchRegistry(context.Background(), t.TempDir()); err == nil {
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
	if _, _, _, err := FetchRegistry(context.Background(), t.TempDir()); err == nil {
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
	if _, err := InstallFromRegistry(context.Background(), t.TempDir(), "demo-remote", ""); err == nil {
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
	if _, err := InstallFromRegistry(context.Background(), dir, "demo-remote", ""); err != nil {
		t.Fatal(err)
	}
	// Registry still serves 1.0.0 but a newer copy is installed.
	newer := testZip(t, `{"id":"demo-remote","name":"Demo Remote","version":"2.0.0"}`)
	if _, err := InstallFromZip(dir, newer); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallFromRegistry(context.Background(), dir, "demo-remote", ""); err == nil {
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

// serveRegistryEntry serves an unsigned registry with one entry whose
// extra fields come from entryExtra, injected into the entry object.
func serveRegistryEntry(t *testing.T, pkg []byte, entryExtra string) *httptest.Server {
	t.Helper()
	sum := sha256.Sum256(pkg)
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
    "version": "1.0.0"%s,
    "package": {"url": %q, "sha256": %q, "bytes": %d},
    "capabilities": {"script": false, "wasm": false, "styles": 0, "appTheme": false, "trackRules": 1, "playerHooks": 0},
    "audit": {"status": "pass", "warnings": []}
  }]
}`, entryExtra, pkgURL, hex.EncodeToString(sum[:]), len(pkg))
	})
	mux.HandleFunc("/demo-remote-1.0.0.zip", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(pkg)
	})
	return srv
}

func TestDelistedExtensionRefusesInstall(t *testing.T) {
	pkg := testZip(t, `{"id":"demo-remote","name":"Demo Remote","version":"1.0.0"}`)
	srv := serveRegistryEntry(t, pkg, `, "delisted": {"reason": "malware", "at": "2026-09-14"}`)
	defer srv.Close()
	t.Setenv("MELOVIAN_EXTENSION_REGISTRY_URL", srv.URL+"/registry.json")
	_, err := InstallFromRegistry(context.Background(), t.TempDir(), "demo-remote", "")
	if err == nil || !strings.Contains(err.Error(), "delisted") {
		t.Fatalf("expected delisted refusal, got %v", err)
	}
}

func TestMinAppVersionGate(t *testing.T) {
	pkg := testZip(t, `{"id":"demo-remote","name":"Demo Remote","version":"1.0.0"}`)
	srv := serveRegistryEntry(t, pkg, `, "minAppVersion": "99.0.0"`)
	defer srv.Close()
	t.Setenv("MELOVIAN_EXTENSION_REGISTRY_URL", srv.URL+"/registry.json")
	_, err := InstallFromRegistry(context.Background(), t.TempDir(), "demo-remote", "")
	if err == nil || !strings.Contains(err.Error(), "needs Melovian 99.0.0") {
		t.Fatalf("expected minAppVersion refusal, got %v", err)
	}
}

func TestRequiresGate(t *testing.T) {
	pkg := testZip(t, `{"id":"demo-remote","name":"Demo Remote","version":"1.0.0"}`)
	srv := serveRegistryEntry(t, pkg, `, "requires": ["base-ext"]`)
	defer srv.Close()
	t.Setenv("MELOVIAN_EXTENSION_REGISTRY_URL", srv.URL+"/registry.json")
	dir := t.TempDir()
	_, err := InstallFromRegistry(context.Background(), dir, "demo-remote", "")
	if err == nil || !strings.Contains(err.Error(), "requires base-ext") {
		t.Fatalf("expected requires refusal, got %v", err)
	}
	// Installing the dependency first satisfies the gate.
	base := testZip(t, `{"id":"base-ext","name":"Base","version":"1.0.0"}`)
	if _, err := InstallFromZip(dir, base); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallFromRegistry(context.Background(), dir, "demo-remote", ""); err != nil {
		t.Fatal(err)
	}
}

func TestRollbackToPublishedVersion(t *testing.T) {
	oldPkg := testZip(t, `{"id":"demo-remote","name":"Demo Remote","version":"1.0.0"}`)
	newPkg := testZip(t, `{"id":"demo-remote","name":"Demo Remote","version":"2.0.0"}`)
	oldSum := sha256.Sum256(oldPkg)
	newSum := sha256.Sum256(newPkg)
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	newURL := srv.URL + "/demo-remote-2.0.0.zip"
	oldURL := srv.URL + "/demo-remote-1.0.0.zip"
	mux.HandleFunc("/registry.json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{
  "version": 1,
  "generatedAt": "2026-09-14T00:00:00Z",
  "extensions": [{
    "id": "demo-remote",
    "name": "Demo Remote",
    "version": "2.0.0",
    "package": {"url": %q, "sha256": %q, "bytes": %d},
    "versions": [
      {"version": "2.0.0", "url": %q, "sha256": %q, "bytes": %d, "releasedAt": "2026-09-14"},
      {"version": "1.0.0", "url": %q, "sha256": %q, "bytes": %d, "releasedAt": "2026-09-01"}
    ],
    "capabilities": {"script": false, "wasm": false, "styles": 0, "appTheme": false, "trackRules": 1, "playerHooks": 0},
    "audit": {"status": "pass", "warnings": []}
  }]
}`, newURL, hex.EncodeToString(newSum[:]), len(newPkg),
			newURL, hex.EncodeToString(newSum[:]), len(newPkg),
			oldURL, hex.EncodeToString(oldSum[:]), len(oldPkg))
	})
	mux.HandleFunc("/demo-remote-2.0.0.zip", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(newPkg)
	})
	mux.HandleFunc("/demo-remote-1.0.0.zip", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(oldPkg)
	})
	defer srv.Close()
	t.Setenv("MELOVIAN_EXTENSION_REGISTRY_URL", srv.URL+"/registry.json")
	dir := t.TempDir()
	m, err := InstallFromRegistry(context.Background(), dir, "demo-remote", "")
	if err != nil || m.Version != "2.0.0" {
		t.Fatalf("expected 2.0.0 install, got %v %v", m.Version, err)
	}
	m, err = InstallFromRegistry(context.Background(), dir, "demo-remote", "1.0.0")
	if err != nil || m.Version != "1.0.0" {
		t.Fatalf("expected rollback to 1.0.0, got %v %v", m.Version, err)
	}
	if _, err := InstallFromRegistry(context.Background(), dir, "demo-remote", "9.9.9"); err == nil {
		t.Fatal("expected unknown version to fail")
	}
	// Latest after a rollback is an update again, so it must pass.
	m, err = InstallFromRegistry(context.Background(), dir, "demo-remote", "")
	if err != nil || m.Version != "2.0.0" {
		t.Fatalf("expected update back to 2.0.0, got %v %v", m.Version, err)
	}
}

func TestRegistryKeyIDMismatchFails(t *testing.T) {
	pkg := testZip(t, `{"id":"demo-remote","name":"Demo Remote","version":"1.0.0"}`)
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(pkg)
	pkgSig := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, pkg))
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	pkgURL := srv.URL + "/demo-remote-1.0.0.zip"
	// keyId claims a different key than the one that actually signed.
	indexJSON := fmt.Sprintf(`{
  "version": 1,
  "keyId": "0000000000000000",
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
		_, _ = w.Write([]byte(indexJSON))
	})
	mux.HandleFunc("/registry.sig", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(base64.StdEncoding.EncodeToString(
			ed25519.Sign(priv, []byte(indexJSON)))))
	})
	defer srv.Close()
	t.Setenv("MELOVIAN_EXTENSION_REGISTRY_URL", srv.URL+"/registry.json")
	t.Setenv("MELOVIAN_EXTENSION_REGISTRY_KEY", hex.EncodeToString(pub))
	_, _, _, err = FetchRegistry(context.Background(), t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "keyId") {
		t.Fatalf("expected keyId mismatch failure, got %v", err)
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

func TestRegistryOverrideRoundTrip(t *testing.T) {
	dir := t.TempDir()
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	keyHex := hex.EncodeToString(pub)
	if err := SaveRegistryOverride(dir, "https://reg.example.com/registry.json", []string{keyHex}); err != nil {
		t.Fatalf("save: %v", err)
	}
	o, ok := LoadRegistryOverride(dir)
	if !ok || o.URL != "https://reg.example.com/registry.json" || len(o.Keys) != 1 {
		t.Fatalf("override = %+v ok=%v", o, ok)
	}
	if got := RegistryURL(dir); got != "https://reg.example.com/registry.json" {
		t.Fatalf("RegistryURL = %q", got)
	}
	if err := ClearRegistryOverride(dir); err != nil {
		t.Fatal(err)
	}
	if got := RegistryURL(dir); got != DefaultRegistryURL {
		t.Fatalf("after clear RegistryURL = %q", got)
	}
}

func TestRegistryOverrideValidation(t *testing.T) {
	dir := t.TempDir()
	if err := SaveRegistryOverride(dir, "http://evil.example.com/r.json", []string{"aa"}); err == nil {
		t.Fatal("expected http URL rejection")
	}
	if err := SaveRegistryOverride(dir, "https://reg.example.com/r.json", nil); err == nil {
		t.Fatal("expected missing key rejection")
	}
	if err := SaveRegistryOverride(dir, "https://reg.example.com/r.json", []string{"not-hex"}); err == nil {
		t.Fatal("expected bad key rejection")
	}
}

func TestRegistryOverrideFetchAndInstall(t *testing.T) {
	// A saved override routes fetches to the custom registry and trusts
	// the saved key for both the index and package signatures.
	dir := t.TempDir()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	pkg := testZip(t, `{"id":"demo-remote","name":"Demo","version":"1.0.0"}`)
	sum := sha256.Sum256(pkg)
	indexJSON, _ := json.Marshal(RegistryIndex{
		Version: 1,
		KeyID:   keyIDOf(pub),
		Extensions: []RegistryEntry{{
			ID: "demo-remote", Name: "Demo", Version: "1.0.0",
			Package: RegistryPackage{
				URL:       "__PKG__/demo-remote.zip",
				SHA256:    hex.EncodeToString(sum[:]),
				Bytes:     int64(len(pkg)),
				Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, pkg)),
			},
		}},
	})
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()
	indexJSON = bytes.ReplaceAll(indexJSON, []byte("__PKG__"), []byte(srv.URL))
	mux.HandleFunc("/registry.json", func(w http.ResponseWriter, r *http.Request) { w.Write(indexJSON) })
	mux.HandleFunc("/registry.sig", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(base64.StdEncoding.EncodeToString(ed25519.Sign(priv, indexJSON))))
	})
	mux.HandleFunc("/demo-remote.zip", func(w http.ResponseWriter, r *http.Request) { w.Write(pkg) })

	if err := SaveRegistryOverride(dir, srv.URL+"/registry.json", []string{hex.EncodeToString(pub)}); err != nil {
		t.Fatal(err)
	}
	manifest, err := InstallFromRegistry(context.Background(), dir, "demo-remote", "")
	if err != nil {
		t.Fatalf("install from custom registry: %v", err)
	}
	if manifest.ID != "demo-remote" {
		t.Fatalf("manifest id = %q", manifest.ID)
	}
}
