// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// Package update implements Melovian's update pipeline: version discovery via
// the GitHub releases Atom feed, asset resolution, signed checksum
// verification, optional bsdiff delta application, and atomic binary swap.
//
// Trust model: the checksums file published with each release is signed with
// a pinned Ed25519 key. The public half is compiled into the binary via
// -X melovian/internal/update.ReleasePublicKey=<base64>. When a key is
// compiled in, every downloaded byte must trace back to a signed checksum
// line. Unsigned or unverifiable releases fail closed. When no key is
// compiled in (local dev builds), checksum verification still runs but the
// missing signature is reported as a warning instead of a hard failure.
package update

import (
	"fmt"
	"net/http"
	"time"
)

// Build-time knobs. Override at link time:
//
//	-X melovian/internal/update.RepoSlug=org/repo
//	-X melovian/internal/update.ReleasePublicKey=<base64 ed25519 public key>
var (
	// RepoSlug is the GitHub owner/repo releases are published under.
	RepoSlug = "melovian-hq/Melovian"
	// FeedURL is the Atom feed polled for update checks.
	FeedURL = "https://github.com/" + RepoSlug + "/releases.atom"
	// APIBase is the GitHub API root used to enumerate release assets.
	APIBase = "https://api.github.com"
	// ReleasePublicKey is the base64-encoded Ed25519 public key that signs
	// the release checksums file. Empty means unsigned-release mode.
	ReleasePublicKey = ""
)

// Channel selects which releases the checker will offer.
type Channel string

const (
	// ChannelStable ignores prereleases and nightly tags.
	ChannelStable Channel = "stable"
	// ChannelPrerelease accepts prerelease tags.
	ChannelPrerelease Channel = "prerelease"
)

// ReleaseInfo describes one release discovered from the update feed.
type ReleaseInfo struct {
	Version     string    `json:"version"`
	Tag         string    `json:"tag"`
	NotesURL    string    `json:"notesUrl"`
	PublishedAt time.Time `json:"publishedAt"`
	Prerelease  bool      `json:"prerelease"`
}

// CheckResult is the outcome of a feed check.
type CheckResult struct {
	Current   string       `json:"current"`
	Latest    *ReleaseInfo `json:"latest,omitempty"`
	UpToDate  bool         `json:"upToDate"`
	CheckedAt time.Time    `json:"checkedAt"`
}

// ProgressStage identifies which phase a Progress event belongs to.
type ProgressStage string

const (
	StageCheck    ProgressStage = "check"
	StageDownload ProgressStage = "download"
	StageVerify   ProgressStage = "verify"
	StagePatch    ProgressStage = "patch"
	StageInstall  ProgressStage = "install"
	StageRestart  ProgressStage = "restart"
	StageDone     ProgressStage = "done"
)

// Progress reports update progress to a caller-supplied callback.
type Progress struct {
	Stage   ProgressStage `json:"stage"`
	Message string        `json:"message"`
	Written int64         `json:"written,omitempty"`
	Total   int64         `json:"total,omitempty"`
}

// ProgressFunc receives Progress events. It must be safe for concurrent use
// only if the caller shares it across goroutines. The updater calls it
// sequentially.
type ProgressFunc func(Progress)

// Options control a Check or Apply run.
type Options struct {
	// Channel selects stable vs prerelease releases. Default stable.
	Channel Channel
	// TargetVersion pins a specific tag (with or without leading v).
	// Empty means "latest on the selected channel".
	TargetVersion string
	// Target is the binary path to replace. Empty means the running
	// executable.
	Target string
	// ArchSuffix overrides the release-asset arch suffix (for example
	// "armv7"). Default derives from runtime.GOARCH/GOARM.
	ArchSuffix string
	// NoRestart skips the post-swap restart command.
	NoRestart bool
	// RestartCommand is run after a successful swap (for example
	// "systemctl restart melovian"). Split on whitespace.
	RestartCommand string
	// OnProgress receives stage and download progress events.
	OnProgress ProgressFunc
	// HTTPClient may be set for tests. Nil uses a default client.
	HTTPClient *http.Client
}

// ErrManualOnly marks releases that cannot be self-applied on this platform
// (for example a Windows installer asset). The caller should link the user
// to the release page instead.
type ErrManualOnly struct {
	ReleaseURL string
}

func (e *ErrManualOnly) Error() string {
	return fmt.Sprintf("update must be installed manually: %s", e.ReleaseURL)
}

func report(fn ProgressFunc, p Progress) {
	if fn != nil {
		fn(p)
	}
}
