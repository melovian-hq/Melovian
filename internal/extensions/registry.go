// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	// DefaultRegistryURL is the published extension index. Override with the
	// MELOVIAN_EXTENSION_REGISTRY_URL environment variable.
	DefaultRegistryURL = "https://melovian-hq.github.io/Melovian-Extensions/registry.json"

	maxRegistryBytes = 2 << 20 // 2 MiB for the index
)

// RegistryEntry is one extension in the remote index.
type RegistryEntry struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Version      string               `json:"version"`
	Description  string               `json:"description,omitempty"`
	Author       string               `json:"author,omitempty"`
	Icon         string               `json:"icon,omitempty"`
	Image        string               `json:"image,omitempty"`
	Package      RegistryPackage      `json:"package"`
	Capabilities RegistryCapabilities `json:"capabilities"`
	Audit        RegistryAudit        `json:"audit"`
}

type RegistryPackage struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type RegistryCapabilities struct {
	Script      bool `json:"script"`
	Wasm        bool `json:"wasm"`
	Styles      int  `json:"styles"`
	AppTheme    bool `json:"appTheme"`
	TrackRules  int  `json:"trackRules"`
	PlayerHooks int  `json:"playerHooks"`
}

type RegistryAudit struct {
	Status   string   `json:"status"`
	Warnings []string `json:"warnings"`
}

// RegistryIndex is the remote registry.json document.
type RegistryIndex struct {
	Version     int             `json:"version"`
	GeneratedAt string          `json:"generatedAt"`
	Extensions  []RegistryEntry `json:"extensions"`
}

// RegistryURL returns the configured index URL.
func RegistryURL() string {
	if v := strings.TrimSpace(os.Getenv("MELOVIAN_EXTENSION_REGISTRY_URL")); v != "" {
		return v
	}
	return DefaultRegistryURL
}

// RegistryBaseURL returns the index URL trimmed to its directory so relative
// asset paths in entries resolve to absolute URLs.
func RegistryBaseURL(indexURL string) string {
	if i := strings.LastIndex(indexURL, "/"); i >= 0 {
		return indexURL[:i+1]
	}
	return indexURL + "/"
}

// ResolveRegistryAsset joins a registry-relative asset path onto the index URL.
func ResolveRegistryAsset(indexURL, rel string) string {
	rel = strings.TrimSpace(strings.ReplaceAll(rel, "\\", "/"))
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		return ""
	}
	base, err := url.Parse(RegistryBaseURL(indexURL))
	if err != nil {
		return ""
	}
	ref, err := url.Parse(rel)
	if err != nil {
		return ""
	}
	return base.ResolveReference(ref).String()
}

// remoteURLOK reports whether raw is a fetchable registry URL. https is
// required everywhere except loopback, where http is allowed for local dev
// and tests.
func remoteURLOK(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	if u.Scheme != "http" {
		return false
	}
	host := u.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

var registryHTTPClient = &http.Client{
	Timeout: 20 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return fmt.Errorf("too many redirects")
		}
		if !remoteURLOK(req.URL.String()) {
			return fmt.Errorf("redirect to disallowed URL")
		}
		return nil
	},
}

func fetchRemote(ctx context.Context, rawURL string, maxBytes int64) ([]byte, error) {
	if !remoteURLOK(rawURL) {
		return nil, fmt.Errorf("registry URLs must be https (http is allowed for loopback only)")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := registryHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry fetch failed with status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("response exceeds %d bytes", maxBytes)
	}
	return data, nil
}

// FetchRegistry downloads and parses the registry index.
func FetchRegistry(ctx context.Context) (RegistryIndex, string, error) {
	indexURL := RegistryURL()
	data, err := fetchRemote(ctx, indexURL, maxRegistryBytes)
	if err != nil {
		return RegistryIndex{}, indexURL, err
	}
	var index RegistryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return RegistryIndex{}, indexURL, fmt.Errorf("invalid registry index: %w", err)
	}
	if index.Version != 1 {
		return RegistryIndex{}, indexURL, fmt.Errorf("unsupported registry version %d", index.Version)
	}
	out := make([]RegistryEntry, 0, len(index.Extensions))
	seen := map[string]bool{}
	for _, entry := range index.Extensions {
		entry.ID = strings.TrimSpace(entry.ID)
		if !IsValidExtensionID(entry.ID) || seen[entry.ID] {
			continue
		}
		seen[entry.ID] = true
		out = append(out, entry)
	}
	index.Extensions = out
	return index, indexURL, nil
}

// FindRegistryEntry returns the index entry for id.
func FindRegistryEntry(index RegistryIndex, id string) (RegistryEntry, bool) {
	for _, entry := range index.Extensions {
		if entry.ID == id {
			return entry, true
		}
	}
	return RegistryEntry{}, false
}

// InstallFromRegistry downloads the package for id from the registry index,
// verifies its sha256, and installs it like an uploaded zip.
func InstallFromRegistry(ctx context.Context, dataDir, id string) (Manifest, error) {
	if !IsValidExtensionID(id) {
		return Manifest{}, fmt.Errorf("invalid extension id")
	}
	index, _, err := FetchRegistry(ctx)
	if err != nil {
		return Manifest{}, err
	}
	entry, ok := FindRegistryEntry(index, id)
	if !ok {
		return Manifest{}, fmt.Errorf("extension not found in registry")
	}
	pkgURL := strings.TrimSpace(entry.Package.URL)
	if pkgURL == "" {
		return Manifest{}, fmt.Errorf("registry entry has no package url")
	}
	if entry.Package.Bytes > maxPackageBytes {
		return Manifest{}, fmt.Errorf("package exceeds %d bytes", maxPackageBytes)
	}
	data, err := fetchRemote(ctx, pkgURL, maxPackageBytes)
	if err != nil {
		return Manifest{}, err
	}
	if entry.Package.Bytes > 0 && int64(len(data)) != entry.Package.Bytes {
		return Manifest{}, fmt.Errorf("package size mismatch")
	}
	want := strings.ToLower(strings.TrimSpace(entry.Package.SHA256))
	if want != "" {
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != want {
			return Manifest{}, fmt.Errorf("package checksum mismatch")
		}
	}
	return InstallFromZip(dataDir, data)
}
