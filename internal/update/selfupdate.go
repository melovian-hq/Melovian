// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package update

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"melovian/internal/brand"
)

// ApplyResult summarises a completed update.
type ApplyResult struct {
	Version         string `json:"version"`
	Path            string `json:"path"`
	Method          string `json:"method"` // "delta" or "full"
	BytesDownloaded int64  `json:"bytesDownloaded"`
	Restarted       bool   `json:"restarted"`
	ReleaseURL      string `json:"releaseUrl"`
}

// Apply runs the full self-update pipeline: resolve the release, download
// (delta when available), verify against the signed checksums, swap the
// binary in place, then optionally run a restart command.
func Apply(ctx context.Context, currentVersion string, opts Options) (*ApplyResult, error) {
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Minute}
	}

	target := opts.Target
	if target == "" {
		if t, err := selfPath(); err == nil {
			target = t
		} else {
			return nil, fmt.Errorf("resolve executable path: %w", err)
		}
	}

	latest, upToDate, err := resolveTarget(ctx, currentVersion, opts)
	if err != nil {
		return nil, err
	}
	if upToDate {
		report(opts.OnProgress, Progress{Stage: StageDone, Message: "Already up to date"})
		return &ApplyResult{Version: currentVersion, Path: target, Method: "none"}, nil
	}

	rel, sums, err := resolveRelease(ctx, client, latest.Tag)
	if err != nil {
		return nil, err
	}

	staging, err := os.MkdirTemp(filepath.Dir(target), ".melovian-update-*")
	if err != nil {
		return nil, fmt.Errorf("create staging dir next to target: %w", err)
	}
	defer os.RemoveAll(staging)

	staged, method, downloaded, err := fetchNewBinary(ctx, client, rel, latest.Tag, currentVersion, PlatformSuffix(opts.ArchSuffix), target, staging, sums, opts)
	if err != nil {
		return nil, err
	}

	report(opts.OnProgress, Progress{Stage: StageInstall, Message: "Installing update"})
	if err := replaceBinary(target, staged); err != nil {
		return nil, err
	}

	result := &ApplyResult{
		Version:         latest.Version,
		Path:            target,
		Method:          method,
		BytesDownloaded: downloaded,
		ReleaseURL:      releaseURL(rel, latest),
	}
	if !opts.NoRestart && strings.TrimSpace(opts.RestartCommand) != "" {
		report(opts.OnProgress, Progress{Stage: StageRestart, Message: "Restarting " + brand.Slug})
		if err := runRestart(ctx, opts.RestartCommand); err != nil {
			return result, fmt.Errorf("update installed but restart failed: %w", err)
		}
		result.Restarted = true
	}
	report(opts.OnProgress, Progress{Stage: StageDone, Message: "Updated to " + latest.Version})
	return result, nil
}

// resolveRelease fetches the release asset list and its verified checksums
// for a tag. The asset list may be nil when the API is unreachable; the
// deterministic download URLs still work.
func resolveRelease(ctx context.Context, client *http.Client, tag string) (*ghRelease, map[string]string, error) {
	rel, relErr := fetchRelease(ctx, client, tag)
	if relErr != nil {
		// The Atom feed already proved the tag exists. A missing asset list
		// only blocks the delta/preferred-name lookup, not the download.
		rel = nil
	}
	sums, err := fetchVerifiedChecksums(ctx, client, rel, tag)
	if err != nil {
		return nil, nil, err
	}
	return rel, sums, nil
}

// resolveTarget picks the release to install: the pinned TargetVersion when
// given, else the newest release on the channel.
func resolveTarget(ctx context.Context, currentVersion string, opts Options) (*ReleaseInfo, bool, error) {
	res, err := Check(ctx, currentVersion, opts)
	if err != nil {
		return nil, false, err
	}
	if res.UpToDate && opts.TargetVersion == "" {
		return nil, true, nil
	}
	if res.Latest == nil {
		return nil, false, fmt.Errorf("no release found for %q", opts.TargetVersion)
	}
	return res.Latest, false, nil
}

// fetchVerifiedChecksums downloads the checksums manifest and its detached
// signature, verifies the signature when a key is pinned, and returns the
// parsed name -> sha256 map. A pinned key with a missing or bad signature
// fails closed.
func fetchVerifiedChecksums(ctx context.Context, client *http.Client, rel *ghRelease, tag string) (map[string]string, error) {
	sumsData, err := DownloadBytes(ctx, client, assetURL(rel, tag, ChecksumAssetName(tag)))
	if err != nil {
		return nil, fmt.Errorf("download checksums: %w", err)
	}
	if !Pinned() {
		report(nil, Progress{Stage: StageVerify, Message: "no signing key compiled in; checksum verification only"})
		return ParseChecksums(sumsData), nil
	}
	sig, err := DownloadBytes(ctx, client, assetURL(rel, tag, SigAssetName(tag)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsignedRelease, err)
	}
	if err := VerifySignature(sumsData, sig); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsignedRelease, err)
	}
	return ParseChecksums(sumsData), nil
}

// fetchNewBinary returns the path of the verified new binary inside staging.
// It prefers a bsdiff delta against the current binary and falls back to a
// full archive download.
func fetchNewBinary(ctx context.Context, client *http.Client, rel *ghRelease, tag, currentVersion, suffix, target, staging string, sums map[string]string, opts Options) (string, string, int64, error) {
	if path, n, err := tryDelta(ctx, client, rel, tag, currentVersion, suffix, target, staging, sums, opts); err == nil {
		return path, "delta", n, nil
	}

	asset := FindAsset(rel, ServerAssetNames(tag, suffix)...)
	name := ""
	if asset != nil {
		name = asset.Name
	} else {
		name = ServerAssetNames(tag, suffix)[0]
	}
	report(opts.OnProgress, Progress{Stage: StageDownload, Message: "Downloading " + name})
	archivePath := filepath.Join(staging, name)
	f, err := os.Create(archivePath)
	if err != nil {
		return "", "", 0, err
	}
	n, derr := Download(ctx, client, assetURL(rel, tag, name), f, opts.OnProgress)
	cerr := f.Close()
	if derr != nil {
		return "", "", n, derr
	}
	if cerr != nil {
		return "", "", n, cerr
	}

	report(opts.OnProgress, Progress{Stage: StageVerify, Message: "Verifying checksum"})
	vf, err := os.Open(archivePath)
	if err != nil {
		return "", "", n, err
	}
	verr := VerifyChecksum(vf, sums, name)
	vf.Close()
	if verr != nil {
		return "", "", n, verr
	}

	binPath, err := ExtractBinary(archivePath, staging, brand.Slug)
	if err != nil {
		return "", "", n, err
	}
	return binPath, "full", n, nil
}

// tryDelta applies a published binary patch when the release ships one for
// our exact current version. Any failure falls back to the full download.
func tryDelta(ctx context.Context, client *http.Client, rel *ghRelease, tag, currentVersion, suffix, target, staging string, sums map[string]string, opts Options) (string, int64, error) {
	binHashName := BinaryHashAssetName(tag, suffix)
	newHash, ok := sums[binHashName]
	if !ok {
		return "", 0, fmt.Errorf("no raw-binary checksum entry")
	}
	currentTag := "v" + strings.TrimPrefix(currentVersion, "v")
	patchName := DeltaAssetName(tag, currentTag, suffix)
	if _, ok := sums[patchName]; !ok {
		// Fall back to any patch for this platform when our exact current
		// tag is not named (for example dev builds reporting a commit sha).
		patchName = ""
		for n := range sums {
			if strings.HasSuffix(n, ".bspatch") && strings.Contains(n, "-"+suffix+"-from-") {
				patchName = n
				break
			}
		}
	}
	if patchName == "" {
		return "", 0, fmt.Errorf("no delta patch published for %s", suffix)
	}

	oldData, err := os.ReadFile(target)
	if err != nil {
		return "", 0, err
	}
	report(opts.OnProgress, Progress{Stage: StageDownload, Message: "Downloading delta patch"})
	var patch bytes.Buffer
	n, err := Download(ctx, client, assetURL(rel, tag, patchName), &patch, opts.OnProgress)
	if err != nil {
		return "", n, err
	}
	report(opts.OnProgress, Progress{Stage: StageVerify, Message: "Verifying patch"})
	if err := VerifyChecksum(bytes.NewReader(patch.Bytes()), sums, patchName); err != nil {
		return "", n, err
	}
	report(opts.OnProgress, Progress{Stage: StagePatch, Message: "Applying delta patch"})
	newData, err := ApplyPatch(oldData, patch.Bytes())
	if err != nil {
		return "", n, err
	}
	if err := VerifyDigest(bytes.NewReader(newData), newHash); err != nil {
		return "", n, err
	}
	out := filepath.Join(staging, binaryBaseName())
	if err := os.WriteFile(out, newData, 0o755); err != nil {
		return "", n, err
	}
	return out, n, nil
}

func binaryBaseName() string {
	if runtime.GOOS == "windows" {
		return brand.Slug + "-server.exe"
	}
	return brand.Slug + "-server"
}

func runRestart(ctx context.Context, command string) error {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return nil
	}
	cmd := exec.CommandContext(ctx, fields[0], fields[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
