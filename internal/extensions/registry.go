// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
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

	// DefaultRegistryKey is the Ed25519 public key (hex) that signs the
	// official registry and its packages. Override with
	// MELOVIAN_EXTENSION_REGISTRY_KEY for a self-hosted registry.
	DefaultRegistryKey = "240f4df79f4ce2e137e29d223f11458b63d04c2f456031b12b139a992719a75e"

	maxRegistryBytes = 2 << 20 // 2 MiB for the index
	maxSignatureBytes = 4 << 10
)

// RegistryEntry is one extension in the remote index.
type RegistryEntry struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Version      string               `json:"version"`
	Description  string               `json:"description,omitempty"`
	Author       string               `json:"author,omitempty"`
	Homepage     string               `json:"homepage,omitempty"`
	License      string               `json:"license,omitempty"`
	Tags         []string             `json:"tags,omitempty"`
	Icon         string               `json:"icon,omitempty"`
	Image        string               `json:"image,omitempty"`
	Screenshots  []string             `json:"screenshots,omitempty"`
	Risk         string               `json:"risk,omitempty"`
	ExternalURLs []string             `json:"externalUrls,omitempty"`
	Versions     []RegistryVersion    `json:"versions,omitempty"`
	Changelog    []RegistryChangelog  `json:"changelog,omitempty"`
	Package      RegistryPackage      `json:"package"`
	Capabilities RegistryCapabilities `json:"capabilities"`
	Audit        RegistryAudit        `json:"audit"`
}

type RegistryPackage struct {
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Bytes     int64  `json:"bytes"`
	Signature string `json:"signature,omitempty"`
}

// RegistryVersion is one released version of an extension.
type RegistryVersion struct {
	Version   string `json:"version"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Bytes     int64  `json:"bytes"`
	Signature string `json:"signature,omitempty"`
	ReleasedAt string `json:"releasedAt,omitempty"`
	Notes     string `json:"notes,omitempty"`
}

// RegistryChangelog is one changelog entry.
type RegistryChangelog struct {
	Version string `json:"version"`
	Date    string `json:"date,omitempty"`
	Notes   string `json:"notes,omitempty"`
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

// registryKeyHex returns the configured Ed25519 public key for the active
// registry. The official registry always uses the compiled-in key. A custom
// registry URL pairs with MELOVIAN_EXTENSION_REGISTRY_KEY.
func registryKeyHex(indexURL string) string {
	if indexURL == DefaultRegistryURL {
		return DefaultRegistryKey
	}
	return strings.TrimSpace(os.Getenv("MELOVIAN_EXTENSION_REGISTRY_KEY"))
}

// registrySigURL returns the sidecar signature URL for an index URL.
func registrySigURL(indexURL string) string {
	return strings.TrimSuffix(indexURL, ".json") + ".sig"
}

func decodeRegistryKey(hexKey string) (ed25519.PublicKey, error) {
	raw, err := hex.DecodeString(strings.TrimSpace(hexKey))
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid registry public key")
	}
	return ed25519.PublicKey(raw), nil
}

// FetchRegistry downloads and parses the registry index. When a signing key
// is configured for the registry, the index must carry a valid registry.sig
// signature. For the official registry this check cannot be disabled.
// The returned bool reports whether the signature was verified.
func FetchRegistry(ctx context.Context) (RegistryIndex, string, bool, error) {
	indexURL := RegistryURL()
	data, err := fetchRemote(ctx, indexURL, maxRegistryBytes)
	if err != nil {
		return RegistryIndex{}, indexURL, false, err
	}
	verified := false
	if keyHex := registryKeyHex(indexURL); keyHex != "" {
		pub, err := decodeRegistryKey(keyHex)
		if err != nil {
			return RegistryIndex{}, indexURL, false, err
		}
		sigData, err := fetchRemote(ctx, registrySigURL(indexURL), maxSignatureBytes)
		if err != nil {
			return RegistryIndex{}, indexURL, false, fmt.Errorf("registry signature unavailable: %w", err)
		}
		sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sigData)))
		if err != nil || !ed25519.Verify(pub, data, sig) {
			return RegistryIndex{}, indexURL, false, fmt.Errorf("registry signature verification failed")
		}
		verified = true
	}
	var index RegistryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return RegistryIndex{}, indexURL, false, fmt.Errorf("invalid registry index: %w", err)
	}
	if index.Version != 1 {
		return RegistryIndex{}, indexURL, false, fmt.Errorf("unsupported registry version %d", index.Version)
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
	return index, indexURL, verified, nil
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

// CompareVersions compares dotted numeric versions, ignoring prerelease
// tags. Returns 1 when a is newer, -1 when older, 0 when equal.
func CompareVersions(a, b string) int {
	parts := func(v string) [3]int {
		var out [3]int
		v = strings.SplitN(strings.TrimSpace(v), "-", 2)[0]
		for i, seg := range strings.SplitN(v, ".", 4) {
			if i > 2 {
				break
			}
			n := 0
			for _, c := range seg {
				if c < '0' || c > '9' {
					break
				}
				n = n*10 + int(c-'0')
			}
			out[i] = n
		}
		return out
	}
	pa, pb := parts(a), parts(b)
	for i := range pa {
		if pa[i] != pb[i] {
			if pa[i] > pb[i] {
				return 1
			}
			return -1
		}
	}
	return 0
}

// verifyPackageSignature checks the zip against the Ed25519 signature in the
// registry entry. When the registry signature verified, a package signature
// is mandatory. For unsigned registries a present signature is still checked.
func verifyPackageSignature(indexURL string, verified bool, entry RegistryEntry, data []byte) error {
	keyHex := registryKeyHex(indexURL)
	sig64 := strings.TrimSpace(entry.Package.Signature)
	if sig64 == "" {
		if verified {
			return fmt.Errorf("package has no signature")
		}
		return nil
	}
	if keyHex == "" {
		return nil
	}
	pub, err := decodeRegistryKey(keyHex)
	if err != nil {
		return err
	}
	sig, err := base64.StdEncoding.DecodeString(sig64)
	if err != nil || !ed25519.Verify(pub, data, sig) {
		return fmt.Errorf("package signature verification failed")
	}
	return nil
}

// InstallFromRegistry downloads the package for id from the registry index,
// verifies its sha256 and signature, and installs it like an uploaded zip.
// It refuses to install a version older than the installed one.
func InstallFromRegistry(ctx context.Context, dataDir, id string) (Manifest, error) {
	if !IsValidExtensionID(id) {
		return Manifest{}, fmt.Errorf("invalid extension id")
	}
	index, indexURL, verified, err := FetchRegistry(ctx)
	if err != nil {
		return Manifest{}, err
	}
	entry, ok := FindRegistryEntry(index, id)
	if !ok {
		return Manifest{}, fmt.Errorf("extension not found in registry")
	}
	if cur, err := findByID(dataDir, id); err == nil {
		if CompareVersions(entry.Version, cur.Manifest.Version) < 0 {
			return Manifest{}, fmt.Errorf(
				"registry version %s is older than installed %s",
				entry.Version, cur.Manifest.Version)
		}
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
	if err := verifyPackageSignature(indexURL, verified, entry, data); err != nil {
		return Manifest{}, err
	}
	return InstallFromZip(dataDir, data)
}
