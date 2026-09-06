// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed all:bundled
var bundledFS embed.FS

const bundledRoot = "bundled"

// BundledOptional lists bundled extension ids that are not installed
// automatically. Users can install them from the extensions settings.
var BundledOptional = map[string]bool{
	"rocksky":        true,
	"lastfm":         true,
	"listenbrainz":   true,
	"lyrics-whisper": true,
}

func extensionInstalled(dataDir, id string) bool {
	_, err := os.Stat(filepath.Join(ExtensionsDir(dataDir), id, ManifestName))
	return err == nil
}

// InstallBundled copies shipped extensions into the user extensions directory.
// Existing .disabled markers are preserved. Bundled files are refreshed so
// asset and manifest updates land on upgrade. Extensions the user uninstalled
// are skipped until ReinstallBundled clears the marker.
func InstallBundled(dataDir string) error {
	root := ExtensionsDir(dataDir)
	if err := os.MkdirAll(root, 0o750); err != nil {
		return err
	}
	ids, err := BundledIDs()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if IsUninstalled(dataDir, id) {
			continue
		}
		if BundledOptional[id] && !extensionInstalled(dataDir, id) {
			continue
		}
		if err := installBundledID(dataDir, id); err != nil {
			return err
		}
	}
	return nil
}

func installBundledID(dataDir, id string) error {
	root := ExtensionsDir(dataDir)
	srcRoot := filepath.ToSlash(filepath.Join(bundledRoot, id))
	return fs.WalkDir(bundledFS, srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(filepath.Join(root, id), 0o750)
		}
		dest := filepath.Join(root, id, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(dest, 0o750)
		}
		if strings.EqualFold(filepath.Base(dest), ".disabled") {
			return nil
		}
		data, err := bundledFS.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0o600)
	})
}
