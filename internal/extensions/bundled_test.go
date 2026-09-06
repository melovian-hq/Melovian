// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallBundledFeatureExtensions(t *testing.T) {
	dir := t.TempDir()
	if err := InstallBundled(dir); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"lyrics", "metadata"} {
		manifest := filepath.Join(ExtensionsDir(dir), id, ManifestName)
		if _, err := os.Stat(manifest); err != nil {
			t.Fatalf("expected bundled manifest for %s: %v", id, err)
		}
	}
	icon := filepath.Join(ExtensionsDir(dir), "lyrics", "assets", "icon.svg")
	if _, err := os.Stat(icon); err != nil {
		t.Fatalf("expected lyrics icon asset: %v", err)
	}
	short, err := ResolveAssetPath(dir, "lyrics", "icon.svg")
	if err != nil {
		t.Fatalf("short asset path: %v", err)
	}
	if short != icon {
		t.Fatalf("short path = %q, want %q", short, icon)
	}
	mc, err := ReadBundledManifest("lyrics")
	if err != nil {
		t.Fatal(err)
	}
	if mc.ID != "lyrics" || mc.Icon == "" {
		t.Fatalf("bundled manifest = %#v, want lyrics with icon", mc)
	}

	// Optional bundled ids (scrobblers, whisper) are not installed until the
	// user picks them in the extensions settings.
	for _, id := range []string{"lyrics-whisper", "lastfm", "listenbrainz", "rocksky"} {
		if extensionInstalled(dir, id) {
			t.Fatalf("optional bundled %s should not auto-install", id)
		}
	}

	disabled := filepath.Join(ExtensionsDir(dir), "lyrics", ".disabled")
	if err := os.WriteFile(disabled, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := InstallBundled(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(disabled); err != nil {
		t.Fatalf("expected .disabled to survive reinstall: %v", err)
	}

	items, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, item := range items {
		found[item.Manifest.ID] = true
		if item.Manifest.ID == "lyrics" && item.Enabled {
			t.Fatal("expected lyrics disabled after marker")
		}
		if item.Manifest.ID == "metadata" && !item.Enabled {
			t.Fatal("expected metadata enabled by default")
		}
	}
	for _, id := range []string{"lyrics", "metadata"} {
		if !found[id] {
			t.Fatalf("bundled %s missing from List", id)
		}
	}
}
