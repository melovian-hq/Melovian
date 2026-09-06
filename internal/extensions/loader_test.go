// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListExtensions(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "extensions", "demo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"id":"demo","name":"Demo","version":"1.0.0","trackRules":[]}`
	if err := os.WriteFile(filepath.Join(root, ManifestName), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Manifest.ID != "demo" {
		t.Fatalf("unexpected items: %#v", items)
	}
	if items[0].InstalledAt == "" {
		t.Fatal("expected InstalledAt from folder mtime")
	}
}
