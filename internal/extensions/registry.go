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

	"melovian/internal/compat"
)

const (
	// DefaultRegistryURL is the published extension index. Override with the
	// MELOVIAN_EXTENSION_REGISTRY_URL environment variable.
	DefaultRegistryURL = "https://melovian-hq.github.io/Melovian-Extensions/registry.json"

	// DefaultRegistryKey is the Ed25519 public key (hex) that signs the
	// official registry and its packages. Override with
	// MELOVIAN_EXTENSION_REGISTRY_KEY for a self-hosted registry.
	DefaultRegistryKey = "240f4df79f4ce2e137e29d223f11458b63d04c2f456031b12b139a992719a75e"

	maxRegistryBytes  = 2 << 20 // 2 MiB for the index
	maxSignatureBytes = 4 << 10
)

// RegistryEntry is one extension in the remote index.
type RegistryEntry struct {
	ID            string               `json:"id"`
	Name          string               `json:"name"`
	Version       string               `json:"version"`
	Description   string               `json:"description,omitempty"`
	Author        string               `json:"author,omitempty"`
	Homepage      string               `json:"homepage,omitempty"`
	License       string               `json:"license,omitempty"`
	Tags          []string             `json:"tags,omitempty"`
	Icon          string               `json:"icon,omitempty"`
	Image         string               `json:"image,omitempty"`
	Screenshots   []string             `json:"screenshots,omitempty"`
	Permissions   []string             `json:"permissions,omitempty"`
	MinAppVersion string               `json:"minAppVersion,omitempty"`
	Requires      []string             `json:"requires,omitempty"`
	Delisted      *RegistryDelisted    `json:"delisted,omitempty"`
	Risk          string               `json:"risk,omitempty"`
	ExternalURLs  []string             `json:"externalUrls,omitempty"`
	Versions      []RegistryVersion    `json:"versions,omitempty"`
	Changelog     []RegistryChangelog  `json:"changelog,omitempty"`
	Package       RegistryPackage      `json:"package"`
	Capabilities  RegistryCapabilities `json:"capabilities"`
	Audit         RegistryAudit        `json:"audit"`
}

// RegistryDelisted marks an extension pulled for cause. New installs are
// refused and installed copies surface a warning.
type RegistryDelisted struct {
	Reason string `json:"reason"`
	At     string `json:"at"`
}

type RegistryPackage struct {
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Bytes     int64  `json:"bytes"`
	Signature string `json:"signature,omitempty"`
}

// RegistryVersion is one released version of an extension.
type RegistryVersion struct {
	Version    string `json:"version"`
	URL        string `json:"url"`
	SHA256     string `json:"sha256"`
	Bytes      int64  `json:"bytes"`
	Signature  string `json:"signature,omitempty"`
	ReleasedAt string `json:"releasedAt,omitempty"`
	Notes      string `json:"notes,omitempty"`
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
	KeyID       string          `json:"keyId,omitempty"`
	Extensions  []RegistryEntry `json:"extensions"`
}

// RegistryURL returns the configured index URL. Environment variables win
// over the saved override so an admin can pin a registry the UI cannot
// change.
func RegistryURL(dataDir string) string {
	if v := strings.TrimSpace(os.Getenv("MELOVIAN_EXTENSION_REGISTRY_URL")); v != "" {
		return v
	}
	if o, ok := LoadRegistryOverride(dataDir); ok {
		return o.URL
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

// registryKeys returns the trusted Ed25519 public keys for the active
// registry, hex encoded. The official registry trusts the compiled-in key
// plus any retired-but-still-valid keys listed in
// MELOVIAN_EXTENSION_REGISTRY_EXTRA_KEYS so a rotation does not strand
// installed clients mid-swap. A custom registry URL pairs with the keys
// saved in the override file, or MELOVIAN_EXTENSION_REGISTRY_KEY (single)
// and MELOVIAN_EXTENSION_REGISTRY_KEYS (comma separated) for env setups.
func registryKeys(indexURL, dataDir string) []string {
	if indexURL == DefaultRegistryURL {
		keys := []string{DefaultRegistryKey}
		for _, k := range strings.Split(os.Getenv("MELOVIAN_EXTENSION_REGISTRY_EXTRA_KEYS"), ",") {
			if k = strings.TrimSpace(k); k != "" {
				keys = append(keys, k)
			}
		}
		return keys
	}
	if multi := strings.TrimSpace(os.Getenv("MELOVIAN_EXTENSION_REGISTRY_KEYS")); multi != "" {
		var keys []string
		for _, k := range strings.Split(multi, ",") {
			if k = strings.TrimSpace(k); k != "" {
				keys = append(keys, k)
			}
		}
		return keys
	}
	if single := strings.TrimSpace(os.Getenv("MELOVIAN_EXTENSION_REGISTRY_KEY")); single != "" {
		return []string{single}
	}
	if o, ok := LoadRegistryOverride(dataDir); ok && o.URL == indexURL {
		return o.Keys
	}
	return nil
}

// keyIDOf derives the registry keyId for a public key: the first 16 hex
// chars of sha256(raw public key). The index carries keyId so clients can
// confirm the signing key across rotations.
func keyIDOf(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	return hex.EncodeToString(sum[:])[:16]
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
func FetchRegistry(ctx context.Context, dataDir string) (RegistryIndex, string, bool, error) {
	indexURL := RegistryURL(dataDir)
	data, err := fetchRemote(ctx, indexURL, maxRegistryBytes)
	if err != nil {
		return RegistryIndex{}, indexURL, false, err
	}
	verified := false
	var verifyKey ed25519.PublicKey
	if keys := registryKeys(indexURL, dataDir); len(keys) > 0 {
		sigData, err := fetchRemote(ctx, registrySigURL(indexURL), maxSignatureBytes)
		if err != nil {
			return RegistryIndex{}, indexURL, false, fmt.Errorf("registry signature unavailable: %w", err)
		}
		sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sigData)))
		if err != nil {
			return RegistryIndex{}, indexURL, false, fmt.Errorf("registry signature verification failed")
		}
		for _, keyHex := range keys {
			pub, err := decodeRegistryKey(keyHex)
			if err != nil {
				return RegistryIndex{}, indexURL, false, err
			}
			if ed25519.Verify(pub, data, sig) {
				verifyKey = pub
				break
			}
		}
		if verifyKey == nil {
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
	// A signed index must carry keyId identifying which trusted key signed
	// it. Binding keyId to the key that verified blocks a stale key from
	// being replayed against a registry that rotated past it.
	if verified {
		if index.KeyID == "" {
			if indexURL == DefaultRegistryURL {
				return RegistryIndex{}, indexURL, false, fmt.Errorf("signed registry is missing keyId")
			}
		} else if index.KeyID != keyIDOf(verifyKey) {
			return RegistryIndex{}, indexURL, false, fmt.Errorf("registry keyId does not match the signing key")
		}
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
func verifyPackageSignature(indexURL, dataDir string, verified bool, pkg RegistryPackage, data []byte) error {
	keys := registryKeys(indexURL, dataDir)
	sig64 := strings.TrimSpace(pkg.Signature)
	if sig64 == "" {
		if verified {
			return fmt.Errorf("package has no signature")
		}
		return nil
	}
	if len(keys) == 0 {
		return nil
	}
	sig, err := base64.StdEncoding.DecodeString(sig64)
	if err != nil {
		return fmt.Errorf("package signature verification failed")
	}
	for _, keyHex := range keys {
		pub, err := decodeRegistryKey(keyHex)
		if err != nil {
			return err
		}
		if ed25519.Verify(pub, data, sig) {
			return nil
		}
	}
	return fmt.Errorf("package signature verification failed")
}

// selectPackage picks the package coordinates for wantVersion. An empty
// wantVersion resolves to the entry's latest. Older entries come from the
// version history so explicit rollbacks stay signed and checksummed.
func selectPackage(entry RegistryEntry, wantVersion string) (RegistryPackage, string, error) {
	want := strings.TrimSpace(wantVersion)
	if want == "" || want == entry.Version {
		return entry.Package, entry.Version, nil
	}
	for _, v := range entry.Versions {
		if v.Version != want {
			continue
		}
		return RegistryPackage{
			URL:       v.URL,
			SHA256:    v.SHA256,
			Bytes:     v.Bytes,
			Signature: v.Signature,
		}, v.Version, nil
	}
	return RegistryPackage{}, "", fmt.Errorf("version %s not published for %s", want, entry.ID)
}

// InstallFromRegistry downloads the package for id from the registry index,
// verifies its sha256 and signature, and installs it like an uploaded zip.
// It refuses to install a version older than the installed one unless
// wantVersion names an explicit published version, which is how rollbacks
// get past the downgrade guard.
func InstallFromRegistry(ctx context.Context, dataDir, id, wantVersion string) (Manifest, error) {
	if !IsValidExtensionID(id) {
		return Manifest{}, fmt.Errorf("invalid extension id")
	}
	index, indexURL, verified, err := FetchRegistry(ctx, dataDir)
	if err != nil {
		return Manifest{}, err
	}
	entry, ok := FindRegistryEntry(index, id)
	if !ok {
		return Manifest{}, fmt.Errorf("extension not found in registry")
	}
	if entry.Delisted != nil {
		reason := strings.TrimSpace(entry.Delisted.Reason)
		if reason == "" {
			reason = "no reason given"
		}
		return Manifest{}, fmt.Errorf("extension was delisted: %s", reason)
	}
	if min := strings.TrimSpace(entry.MinAppVersion); min != "" {
		// Sha-stamped dev builds are not comparable to semver floors.
		if compat.IsSemverVersion(compat.Version) && CompareVersions(compat.Version, min) < 0 {
			return Manifest{}, fmt.Errorf(
				"extension needs Melovian %s or newer, this is %s", min, compat.Version)
		}
	}
	for _, dep := range entry.Requires {
		if !IsValidExtensionID(dep) {
			continue
		}
		if _, err := findByID(dataDir, dep); err != nil {
			return Manifest{}, fmt.Errorf("extension requires %s, install it first", dep)
		}
	}
	pkg, _, err := selectPackage(entry, wantVersion)
	if err != nil {
		return Manifest{}, err
	}
	// Explicit version requests are rollbacks: the user confirmed the
	// downgrade, so only the implicit latest path enforces ordering.
	if wantVersion == "" {
		if cur, err := findByID(dataDir, id); err == nil {
			if CompareVersions(entry.Version, cur.Manifest.Version) < 0 {
				return Manifest{}, fmt.Errorf(
					"registry version %s is older than installed %s",
					entry.Version, cur.Manifest.Version)
			}
		}
	}
	pkgURL := strings.TrimSpace(pkg.URL)
	if pkgURL == "" {
		return Manifest{}, fmt.Errorf("registry entry has no package url")
	}
	if pkg.Bytes > maxPackageBytes {
		return Manifest{}, fmt.Errorf("package exceeds %d bytes", maxPackageBytes)
	}
	data, err := fetchRemote(ctx, pkgURL, maxPackageBytes)
	if err != nil {
		return Manifest{}, err
	}
	if pkg.Bytes > 0 && int64(len(data)) != pkg.Bytes {
		return Manifest{}, fmt.Errorf("package size mismatch")
	}
	want := strings.ToLower(strings.TrimSpace(pkg.SHA256))
	if want != "" {
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != want {
			return Manifest{}, fmt.Errorf("package checksum mismatch")
		}
	}
	if err := verifyPackageSignature(indexURL, dataDir, verified, pkg, data); err != nil {
		return Manifest{}, err
	}
	return InstallFromZip(dataDir, data)
}
