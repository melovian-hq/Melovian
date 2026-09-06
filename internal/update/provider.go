// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package update

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

// AtomProvider is a Wails updater.Provider backed by the project release
// Atom feed. Check discovers the newest tag from the feed, enumerates its
// assets through the release API, and verifies the signed checksums before
// handing the artifact digest to the Wails Updater for download-time
// verification.
type AtomProvider struct {
	// HTTPClient may be set for tests.
	HTTPClient *http.Client
	// Channel selects stable or prerelease releases.
	Channel Channel
}

// NewAtomProvider builds the default provider for the compiled-in repo.
func NewAtomProvider() *AtomProvider {
	return &AtomProvider{Channel: ChannelStable}
}

// Name implements updater.Provider.
func (p *AtomProvider) Name() string { return "atom" }

func (p *AtomProvider) client() *http.Client {
	if p.HTTPClient != nil {
		return p.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Minute}
}

// Check implements updater.Provider. It returns (nil, nil) when up to date
// and *ErrManualOnly when the release has no in-place-swappable asset for
// this platform (installers, AppImage, container images).
func (p *AtomProvider) Check(ctx context.Context, req updater.CheckRequest) (*updater.Release, error) {
	feed, err := FetchFeed(ctx, p.client(), FeedURL)
	if err != nil {
		return nil, err
	}
	latest := LatestFromFeed(feed, p.Channel)
	if latest == nil || !IsNewer(req.CurrentVersion, latest.Version) {
		return nil, nil
	}
	rel, relErr := FetchRelease(ctx, p.client(), latest.Tag)
	if relErr != nil {
		return nil, fmt.Errorf("enumerate assets for %s: %w", latest.Tag, relErr)
	}
	asset := selectDesktopAsset(rel, req)
	if asset == nil {
		return nil, &ErrManualOnly{ReleaseURL: releaseURL(rel, latest)}
	}

	// Verify the signed checksums manifest now, then hand the artifact's
	// digest to the Updater so the downloaded bytes are checked against a
	// value that traced through the pinned signature.
	sums, err := fetchVerifiedChecksums(ctx, p.client(), rel, latest.Tag)
	if err != nil {
		return nil, err
	}
	sumHex, ok := sums[asset.Name]
	if !ok {
		return nil, fmt.Errorf("no checksum entry for %s", asset.Name)
	}
	digest, err := hex.DecodeString(sumHex)
	if err != nil {
		return nil, fmt.Errorf("bad checksum for %s: %w", asset.Name, err)
	}

	return &updater.Release{
		Version:     latest.Version,
		Channel:     string(p.Channel),
		Name:        rel.TagName,
		Notes:       rel.Body,
		PublishedAt: latest.PublishedAt,
		Artifact: updater.Artifact{
			Filename: asset.Name,
			Size:     asset.Size,
			Platform: req.Platform,
			Arch:     req.Arch,
		},
		Verification: &updater.Verification{
			DigestAlgo: "sha256",
			Digest:     digest,
		},
		Metadata: map[string]any{
			"releaseUrl": releaseURL(rel, latest),
		},
	}, nil
}

// Download implements updater.Provider.
func (p *AtomProvider) Download(ctx context.Context, r *updater.Release, dst io.Writer, onProgress func(written, total int64)) error {
	url := releaseDownloadURL("v"+strings.TrimPrefix(r.Version, "v"), r.Artifact.Filename)
	pw := &progressWriter{fn: onProgress}
	_, err := Download(ctx, p.client(), url, io.MultiWriter(dst, pw), nil)
	return err
}

type progressWriter struct {
	fn      func(written, total int64)
	written int64
}

func (w *progressWriter) Write(b []byte) (int, error) {
	n := len(b)
	w.written += int64(n)
	if w.fn != nil {
		w.fn(w.written, -1)
	}
	return n, nil
}

func releaseURL(rel *ghRelease, latest *ReleaseInfo) string {
	if rel != nil && rel.HTMLURL != "" {
		return rel.HTMLURL
	}
	return latest.NotesURL
}

// selectDesktopAsset picks an in-place-swappable asset for the running
// platform. Raw-binary archives and .app bundles in .zip can be swapped by
// the Wails helper. Installers (msi, exe, dmg), AppImages, and container
// images are manual downloads.
func selectDesktopAsset(rel *ghRelease, req updater.CheckRequest) *Asset {
	goos := req.Platform
	if goos == "" {
		goos = runtime.GOOS
	}
	arch := req.Arch
	if arch == "" {
		arch = runtime.GOARCH
	}
	osNames := map[string][]string{
		"darwin":  {"darwin", "macos", "mac"},
		"windows": {"windows", "win"},
		"linux":   {"linux"},
	}[goos]
	if len(osNames) == 0 {
		osNames = []string{goos}
	}
	bestScore := -1
	var best *Asset
	for i := range rel.Assets {
		a := &rel.Assets[i]
		name := strings.ToLower(a.Name)
		osMatch := false
		for _, n := range osNames {
			if strings.Contains(name, n) {
				osMatch = true
				break
			}
		}
		if !osMatch {
			continue
		}
		archMatch := strings.Contains(name, arch) || (goos == "darwin" && strings.Contains(name, "universal"))
		if !archMatch {
			continue
		}
		score := desktopAssetScore(name, goos)
		if score > bestScore {
			bestScore = score
			best = a
		}
	}
	if bestScore <= 0 {
		return nil
	}
	return best
}

// desktopAssetScore ranks an asset by swap-ability. <=0 means manual only.
func desktopAssetScore(name, goos string) int {
	switch {
	case strings.HasSuffix(name, ".msi"), strings.HasSuffix(name, ".dmg"),
		strings.HasSuffix(name, ".appimage"), strings.HasSuffix(name, ".apk"),
		strings.HasSuffix(name, ".ipa"), strings.Contains(name, "checksums"),
		strings.HasSuffix(name, ".sig"), strings.HasSuffix(name, ".bspatch"),
		strings.HasSuffix(name, "-setup.exe"), strings.HasSuffix(name, "-installer.exe"):
		return 0
	case goos == "windows" && strings.HasSuffix(name, ".zip"):
		return 2 // portable zip containing the exe
	case goos == "darwin" && strings.HasSuffix(name, ".zip"):
		return 3 // .app bundle zip, handled by the helper extraction
	case strings.HasSuffix(name, ".tar.gz"), strings.HasSuffix(name, ".zip"):
		return 1
	}
	return 0
}
