// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallFromZipAndUninstall(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	manifest := `{
  "id": "demo-pack",
  "name": "Demo Pack",
  "version": "1.2.3",
  "icon": "assets/icon.png",
  "image": "assets/banner.png"
}`
	w, err := zw.Create("melovian-extension.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(manifest)); err != nil {
		t.Fatal(err)
	}
	w, err = zw.Create("assets/icon.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte{0x89, 0x50, 0x4e, 0x47}); err != nil {
		t.Fatal(err)
	}
	w, err = zw.Create("plugin.wasm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte{0x00, 0x61, 0x73, 0x6d}); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := InstallFromZip(dir, buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "demo-pack" || got.Icon != "assets/icon.png" {
		t.Fatalf("unexpected manifest: %#v", got)
	}
	asset, err := ResolveAssetPath(dir, "demo-pack", "assets/icon.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(asset); err != nil {
		t.Fatal(err)
	}
	extDir := filepath.Join(ExtensionsDir(dir), "demo-pack")
	if !HasWasm(extDir) {
		t.Fatal("expected wasm detection")
	}
	if err := Uninstall(dir, "demo-pack"); err != nil {
		t.Fatal(err)
	}
	items, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty list, got %#v", items)
	}
}

func TestBundledUninstallAndReinstall(t *testing.T) {
	dir := t.TempDir()
	if err := InstallBundled(dir); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(dir, "lyrics"); err != nil {
		t.Fatal(err)
	}
	if !IsUninstalled(dir, "lyrics") {
		t.Fatal("expected uninstalled marker")
	}
	if err := InstallBundled(dir); err != nil {
		t.Fatal(err)
	}
	items, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.Manifest.ID == "lyrics" {
			t.Fatal("bundled extension should stay uninstalled")
		}
	}
	if err := ReinstallBundled(dir, "lyrics"); err != nil {
		t.Fatal(err)
	}
	items, err = List(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range items {
		if item.Manifest.ID == "lyrics" {
			found = true
			if item.Manifest.Icon == "" {
				t.Fatal("expected icon on bundled manifest")
			}
		}
	}
	if !found {
		t.Fatal("expected lyrics after reinstall")
	}
}

func TestInstallRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("melovian-extension.json")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte(`{"id":"evil","name":"Evil","version":"1"}`))
	w, err = zw.Create("../escape.js")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte(`bad`))
	_ = zw.Close()
	if _, err := InstallFromZip(dir, buf.Bytes()); err == nil {
		t.Fatal("expected traversal to fail")
	}
}
