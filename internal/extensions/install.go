// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const (
	maxPackageBytes = 32 << 20 // 32 MiB
	maxAssetBytes   = 8 << 20  // 8 MiB per file inside a package
)

var extensionIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

var allowedPackageExts = map[string]bool{
	".json":  true,
	".js":    true,
	".mjs":   true,
	".wasm":  true,
	".css":   true,
	".txt":   true,
	".md":    true,
	".png":   true,
	".jpg":   true,
	".jpeg":  true,
	".webp":  true,
	".gif":   true,
	".svg":   true,
	".woff":  true,
	".woff2": true,
	".ttf":   true,
	".otf":   true,
}

// IsValidExtensionID reports whether id is a safe directory name.
func IsValidExtensionID(id string) bool {
	return extensionIDPattern.MatchString(id)
}

func uninstalledMarkerPath(dataDir, id string) string {
	return filepath.Join(ExtensionsDir(dataDir), ".uninstalled", id)
}

// IsUninstalled reports whether a bundled extension was removed by the user.
func IsUninstalled(dataDir, id string) bool {
	_, err := os.Stat(uninstalledMarkerPath(dataDir, id))
	return err == nil
}

func markUninstalled(dataDir, id string) error {
	dir := filepath.Join(ExtensionsDir(dataDir), ".uninstalled")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	return os.WriteFile(uninstalledMarkerPath(dataDir, id), []byte{}, 0o600)
}

func clearUninstalled(dataDir, id string) error {
	err := os.Remove(uninstalledMarkerPath(dataDir, id))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// BundledIDs returns ids shipped in the embedded bundle.
func BundledIDs() ([]string, error) {
	entries, err := fs.ReadDir(bundledFS, bundledRoot)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			out = append(out, entry.Name())
		}
	}
	return out, nil
}

// IsBundled reports whether id is a shipped extension.
func IsBundled(id string) bool {
	ids, err := BundledIDs()
	if err != nil {
		return false
	}
	return slices.Contains(ids, id)
}

// ReadBundledManifest loads a shipped extension manifest by id.
func ReadBundledManifest(id string) (Manifest, error) {
	if !IsValidExtensionID(id) || !IsBundled(id) {
		return Manifest{}, fmt.Errorf("bundled extension not found")
	}
	data, err := bundledFS.ReadFile(filepath.ToSlash(filepath.Join(bundledRoot, id, ManifestName)))
	if err != nil {
		return Manifest{}, err
	}
	manifest, err := ParseManifest(data)
	if err != nil {
		return Manifest{}, err
	}
	manifest.Styles = SanitizeStyles(manifest.Styles)
	return manifest, nil
}

// InstallFromZip installs or replaces an extension from a zip archive.
// The zip must contain melovian-extension.json at the root or inside one top-level folder.
func InstallFromZip(dataDir string, zipData []byte) (Manifest, error) {
	if len(zipData) == 0 {
		return Manifest{}, fmt.Errorf("empty package")
	}
	if len(zipData) > maxPackageBytes {
		return Manifest{}, fmt.Errorf("package exceeds %d bytes", maxPackageBytes)
	}

	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return Manifest{}, fmt.Errorf("invalid zip: %w", err)
	}

	manifestName := strings.ToLower(ManifestName)
	var manifestFile *zip.File
	prefix := ""
	for _, file := range reader.File {
		name := filepath.ToSlash(file.Name)
		if strings.HasPrefix(name, "__MACOSX/") || strings.HasSuffix(name, "/") {
			continue
		}
		base := strings.ToLower(filepath.Base(name))
		if base != manifestName {
			continue
		}
		depth := strings.Count(strings.Trim(name, "/"), "/")
		if depth > 1 {
			continue
		}
		manifestFile = file
		if depth == 1 {
			prefix = strings.SplitN(name, "/", 2)[0] + "/"
		}
		break
	}
	if manifestFile == nil {
		return Manifest{}, fmt.Errorf("zip missing %s", ManifestName)
	}

	manifestBytes, err := readZipFile(manifestFile, maxAssetBytes)
	if err != nil {
		return Manifest{}, err
	}
	manifest, err := ParseManifest(manifestBytes)
	if err != nil {
		return Manifest{}, fmt.Errorf("invalid manifest: %w", err)
	}
	id := strings.TrimSpace(manifest.ID)
	if !IsValidExtensionID(id) {
		return Manifest{}, fmt.Errorf("invalid extension id")
	}
	if strings.TrimSpace(manifest.Name) == "" || strings.TrimSpace(manifest.Version) == "" {
		return Manifest{}, fmt.Errorf("manifest requires name and version")
	}

	dest := filepath.Join(ExtensionsDir(dataDir), id)
	if err := os.MkdirAll(ExtensionsDir(dataDir), 0o750); err != nil {
		return Manifest{}, err
	}
	tmp, err := os.MkdirTemp(ExtensionsDir(dataDir), id+".install-*")
	if err != nil {
		return Manifest{}, err
	}
	defer os.RemoveAll(tmp)

	for _, file := range reader.File {
		name := filepath.ToSlash(file.Name)
		if strings.HasPrefix(name, "__MACOSX/") || strings.HasSuffix(name, "/") {
			continue
		}
		rel := name
		if prefix != "" {
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			rel = strings.TrimPrefix(name, prefix)
		}
		if rel == "" || strings.Contains(rel, "..") {
			return Manifest{}, fmt.Errorf("invalid path in zip: %s", name)
		}
		ext := strings.ToLower(filepath.Ext(rel))
		if !allowedPackageExts[ext] && !strings.EqualFold(filepath.Base(rel), ManifestName) {
			return Manifest{}, fmt.Errorf("disallowed file type: %s", rel)
		}
		if file.UncompressedSize64 > maxAssetBytes {
			return Manifest{}, fmt.Errorf("file too large: %s", rel)
		}
		target := filepath.Join(tmp, filepath.FromSlash(rel))
		if !strings.HasPrefix(target, tmp+string(os.PathSeparator)) && target != tmp {
			return Manifest{}, fmt.Errorf("zip slip blocked: %s", rel)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return Manifest{}, err
		}
		data, err := readZipFile(file, maxAssetBytes)
		if err != nil {
			return Manifest{}, err
		}
		if err := os.WriteFile(target, data, 0o600); err != nil {
			return Manifest{}, err
		}
	}

	if err := os.RemoveAll(dest); err != nil {
		return Manifest{}, err
	}
	if err := os.Rename(tmp, dest); err != nil {
		return Manifest{}, err
	}
	if err := clearUninstalled(dataDir, id); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func readZipFile(file *zip.File, limit int64) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("file too large: %s", file.Name)
	}
	return data, nil
}

// Uninstall removes an installed extension. Bundled ids are marked so InstallBundled
// will not restore them until ReinstallBundled is called.
func Uninstall(dataDir, id string) error {
	if !IsValidExtensionID(id) {
		return fmt.Errorf("invalid extension id")
	}
	entry, err := findByID(dataDir, id)
	if err != nil {
		return err
	}
	if IsBundled(id) {
		if err := markUninstalled(dataDir, id); err != nil {
			return err
		}
	}
	return os.RemoveAll(entry.Dir)
}

// ReinstallBundled clears the uninstall marker and copies the shipped extension again.
func ReinstallBundled(dataDir, id string) error {
	if !IsValidExtensionID(id) || !IsBundled(id) {
		return fmt.Errorf("bundled extension not found")
	}
	if err := clearUninstalled(dataDir, id); err != nil {
		return err
	}
	return installBundledID(dataDir, id)
}

// ResolveAssetPath returns a safe absolute path for an asset under an extension directory.
// If rel is missing at the extension root, assets/rel is tried so short names still resolve.
func ResolveAssetPath(dataDir, id, rel string) (string, error) {
	if !IsValidExtensionID(id) {
		return "", fmt.Errorf("invalid extension id")
	}
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" || strings.Contains(rel, "..") {
		return "", fmt.Errorf("invalid asset path")
	}
	entry, err := findByID(dataDir, id)
	if err != nil {
		return "", err
	}
	candidates := []string{rel}
	if !strings.HasPrefix(rel, "assets/") {
		candidates = append(candidates, "assets/"+rel)
	}
	dir := filepath.Clean(entry.Dir)
	var lastErr error
	for _, candidate := range candidates {
		full := filepath.Clean(filepath.Join(entry.Dir, filepath.FromSlash(candidate)))
		if full != dir && !strings.HasPrefix(full, dir+string(os.PathSeparator)) {
			lastErr = fmt.Errorf("invalid asset path")
			continue
		}
		info, err := os.Stat(full)
		if err != nil {
			lastErr = err
			continue
		}
		if info.IsDir() {
			lastErr = fmt.Errorf("not a file")
			continue
		}
		return full, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("not found")
	}
	return "", lastErr
}

// HasWasm reports whether the extension directory contains any .wasm file.
func HasWasm(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ".wasm") {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found
}
