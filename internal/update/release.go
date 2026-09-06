// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

// Asset is one downloadable file attached to a release.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

// ghRelease is the slice of the GitHub release API response we use. The
// Atom feed discovers the tag. This call enumerates its assets.
type ghRelease struct {
	TagName string  `json:"tag_name"`
	HTMLURL string  `json:"html_url"`
	Body    string  `json:"body"`
	Assets  []Asset `json:"assets"`
}

// fetchRelease returns release metadata for a tag.
func fetchRelease(ctx context.Context, client *http.Client, tag string) (*ghRelease, error) {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	url := fmt.Sprintf("%s/repos/%s/releases/tags/%s", APIBase, RepoSlug, tag)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent())
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch release %s: %w", tag, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch release %s: %s", tag, resp.Status)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release %s: %w", tag, err)
	}
	return &rel, nil
}

// ChecksumAssetName is the release checksums manifest.
func ChecksumAssetName(tag string) string {
	return fmt.Sprintf("melovian-%s-checksums.txt", tag)
}

// SigAssetName is the detached signature over the checksums manifest.
func SigAssetName(tag string) string {
	return ChecksumAssetName(tag) + ".sig"
}

// PlatformSuffix maps GOOS/GOARCH to the release asset suffix. Server
// archives are named melovian-<tag>-server-<os>-<arch>.<ext> with arm
// becoming armv6 or armv7. A caller may override the suffix via
// Options.ArchSuffix.
func PlatformSuffix(archSuffix string) string {
	if archSuffix != "" {
		return archSuffix
	}
	arch := runtime.GOARCH
	if arch == "arm" {
		// GOARM is a build tag, not a runtime value. Melovian ships armv6 and
		// armv7. Detect via runtime or default to v7 (the far more common
		// target on modern boards).
		arch = "armv7"
	}
	return runtime.GOOS + "-" + arch
}

// ServerAssetNames returns candidate archive names for the running
// platform, most-preferred first.
func ServerAssetNames(tag, suffix string) []string {
	base := fmt.Sprintf("melovian-%s-server-%s", tag, suffix)
	if runtime.GOOS == "windows" {
		return []string{base + ".zip", base + ".tar.gz"}
	}
	return []string{base + ".tar.gz", base + ".zip"}
}

// BinaryHashAssetName is the checksums entry covering the raw binary inside
// the archive, used to verify delta-patched results.
func BinaryHashAssetName(tag, suffix string) string {
	return fmt.Sprintf("melovian-%s-bin-%s", tag, suffix)
}

// DeltaAssetName is the bsdiff patch that turns the oldTag binary into the
// newTag binary for this platform.
func DeltaAssetName(newTag, oldTag, suffix string) string {
	return fmt.Sprintf("melovian-%s-patch-%s-from-%s.bspatch", newTag, suffix, oldTag)
}

// FindAsset returns the first named asset present on the release.
func FindAsset(rel *ghRelease, names ...string) *Asset {
	for _, want := range names {
		for _, a := range rel.Assets {
			if a.Name == want {
				c := a
				return &c
			}
		}
	}
	return nil
}

// releaseDownloadURL builds the direct download URL when the API asset list
// is unavailable (private repo, API rate limit).
func releaseDownloadURL(tag, name string) string {
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", RepoSlug, tag, name)
}

// assetURL resolves the download URL for a named asset, preferring the API
// listing but falling back to the deterministic download path.
func assetURL(rel *ghRelease, tag, name string) string {
	if rel != nil {
		for _, a := range rel.Assets {
			if a.Name == name && a.URL != "" {
				return a.URL
			}
		}
	}
	return releaseDownloadURL(tag, name)
}
