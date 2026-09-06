// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func ExtensionsDir(dataDir string) string {
	return filepath.Join(dataDir, "extensions")
}

func List(dataDir string) ([]Entry, error) {
	root := ExtensionsDir(dataDir)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	out := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		manifestPath := filepath.Join(dir, ManifestName)
		data, err := os.ReadFile(manifestPath) //#nosec G304 -- path is under ExtensionsDir
		if err != nil {
			continue
		}
		manifest, err := ParseManifest(data)
		if err != nil || strings.TrimSpace(manifest.ID) == "" {
			continue
		}
		manifest.Styles = SanitizeStyles(manifest.Styles)
		enabled := !disabledMarker(dir)
		installedAt := ""
		if info, err := os.Stat(dir); err == nil {
			installedAt = info.ModTime().UTC().Format(time.RFC3339)
		}
		out = append(out, Entry{
			Manifest:    manifest,
			Dir:         dir,
			Enabled:     enabled,
			InstalledAt: installedAt,
		})
	}
	return out, nil
}

func disabledMarker(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".disabled"))
	return err == nil
}

func ReadScript(dataDir, id string) ([]byte, string, error) {
	entry, err := findByID(dataDir, id)
	if err != nil {
		return nil, "", err
	}
	scriptName := strings.TrimSpace(entry.Manifest.Script)
	if scriptName == "" {
		return nil, "", fmt.Errorf("extension has no script")
	}
	if strings.Contains(scriptName, "..") || filepath.IsAbs(scriptName) {
		return nil, "", fmt.Errorf("invalid script path")
	}
	path := filepath.Join(entry.Dir, scriptName)
	data, err := os.ReadFile(path) //#nosec G304 -- scriptName checked for traversal above
	if err != nil {
		return nil, "", err
	}
	return data, "application/javascript; charset=utf-8", nil
}

func findByID(dataDir, id string) (Entry, error) {
	items, err := List(dataDir)
	if err != nil {
		return Entry{}, err
	}
	for _, item := range items {
		if item.Manifest.ID == id {
			return item, nil
		}
	}
	return Entry{}, fmt.Errorf("extension not found")
}

func EnabledManifests(dataDir string) ([]Manifest, error) {
	items, err := List(dataDir)
	if err != nil {
		return nil, err
	}
	out := make([]Manifest, 0, len(items))
	for _, item := range items {
		if item.Enabled {
			out = append(out, item.Manifest)
		}
	}
	return out, nil
}

func SetEnabled(dataDir, id string, enabled bool) error {
	entry, err := findByID(dataDir, id)
	if err != nil {
		return err
	}
	marker := filepath.Join(entry.Dir, ".disabled")
	if enabled {
		return os.Remove(marker)
	}
	f, err := os.OpenFile(marker, os.O_CREATE|os.O_WRONLY, 0o600) //#nosec G304 -- marker path is under entry.Dir
	if err != nil {
		return err
	}
	return f.Close()
}

// IsEnabled reports whether an extension with the given id is installed and
// enabled in the user's extensions directory.
func IsEnabled(dataDir, id string) bool {
	dir := filepath.Join(ExtensionsDir(dataDir), id)
	if _, err := os.Stat(filepath.Join(dir, ManifestName)); err != nil {
		return false
	}
	return !disabledMarker(dir)
}
