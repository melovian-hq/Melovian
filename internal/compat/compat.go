// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// Package compat holds shared client/server version and capability contracts.
package compat

import (
	"cmp"
	"runtime/debug"
	"slices"
	"strings"
)

// defaultVersion is the link-time default before -X or VCS stamping.
const defaultVersion = "0.1.0"

// Version is the Melovian release version. Override at link time with
// -X melovian/internal/compat.Version=v0.1.0
var Version = defaultVersion

// BuildDate is set at link time (-X melovian/internal/compat.BuildDate=...)
// or filled from VCS commit time when available.
var BuildDate = ""

func init() {
	if Version != defaultVersion && BuildDate != "" {
		return
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	rev := ""
	modified := false
	vcsTime := ""
	for _, setting := range bi.Settings {
		switch setting.Key {
		case "vcs.revision":
			rev = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		case "vcs.time":
			vcsTime = setting.Value
		}
	}
	if BuildDate == "" && vcsTime != "" {
		BuildDate = vcsTime
	}
	if Version != defaultVersion {
		return
	}
	if rev == "" {
		return
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	if modified {
		rev += "-dirty"
	}
	Version = rev
}

// APIVersion increments only when the HTTP wire contract breaks.
const APIVersion = 1

// MinClientVersion is the oldest client this server accepts.
const MinClientVersion = "0.1.0"

// MinServerVersion is the oldest server this client accepts.
const MinServerVersion = "0.1.0"

const (
	HeaderClientVersion = "X-Melovian-Client-Version"
	HeaderAPIVersion    = "X-Melovian-API-Version"
	HeaderCapabilities  = "X-Melovian-Capabilities"
	HeaderServerVersion = "X-Melovian-Server-Version"
)

// Capability names. Add a constant when a new feature needs UI gating.
const (
	CapAuth           = "auth"
	CapInstances      = "instances"
	CapPlaylists      = "playlists"
	CapFavorites      = "favorites"
	CapHistory        = "history"
	CapMixes          = "mixes"
	CapPersonalRadio  = "personal_radio"
	CapLocalLibrary   = "local_library"
	CapDownloads      = "downloads"
	CapParty          = "party"
	CapExtensions     = "extensions"
	CapWS             = "ws"
	CapLyrics         = "lyrics"
	CapEQ             = "eq"
	CapDevices        = "devices"
	CapShared         = "shared"
	CapVideos         = "videos"
	CapMetadataEditor = "metadata_editor"
	CapSettings       = "settings"
	CapBrowse         = "browse"
	CapPlayback       = "playback"
)

// Capabilities is the full set this build supports.
var Capabilities = []string{
	CapBrowse,
	CapPlayback,
	CapAuth,
	CapInstances,
	CapPlaylists,
	CapFavorites,
	CapHistory,
	CapMixes,
	CapPersonalRadio,
	CapLocalLibrary,
	CapDownloads,
	CapParty,
	CapExtensions,
	CapWS,
	CapLyrics,
	CapEQ,
	CapDevices,
	CapShared,
	CapVideos,
	CapMetadataEditor,
	CapSettings,
}

// LegacyCapabilities is assumed when talking to a server that does not
// advertise capabilities (pre-compat builds).
var LegacyCapabilities = []string{
	CapBrowse,
	CapPlayback,
	CapAuth,
	CapInstances,
	CapPlaylists,
	CapFavorites,
	CapHistory,
	CapSettings,
}

// ConfigFields returns version/capability fields for /api/config and /health.
func ConfigFields() map[string]any {
	fields := map[string]any{
		"version":          Version,
		"apiVersion":       APIVersion,
		"minClientVersion": MinClientVersion,
		"minServerVersion": MinServerVersion,
		"capabilities":     Capabilities,
	}
	if strings.TrimSpace(BuildDate) != "" {
		fields["buildDate"] = BuildDate
	}
	return fields
}

// ParseCapabilities splits a comma-separated capability header.
func ParseCapabilities(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// JoinCapabilities joins capabilities for response headers.
func JoinCapabilities(caps []string) string {
	return strings.Join(caps, ",")
}

// HasCapability reports whether cap is in the list.
func HasCapability(caps []string, cap string) bool {
	return slices.Contains(caps, cap)
}

// Intersect returns capabilities present in both lists.
func Intersect(a, b []string) []string {
	set := make(map[string]struct{}, len(b))
	for _, c := range b {
		set[c] = struct{}{}
	}
	out := make([]string, 0, len(a))
	for _, c := range a {
		if _, ok := set[c]; ok {
			out = append(out, c)
		}
	}
	return out
}

// CompareSemver compares two dotted versions (major.minor.patch...).
// Returns -1 if a < b, 0 if equal, 1 if a > b. Non-numeric suffixes are ignored
// after the numeric core (v prefix stripped).
func CompareSemver(a, b string) int {
	return cmp.Compare(normalizeSemver(a), normalizeSemver(b))
}

func normalizeSemver(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	for i, p := range parts {
		p = strings.TrimLeft(p, "0")
		if p == "" {
			p = "0"
		}
		parts[i] = padNumeric(p)
	}
	return strings.Join(parts, ".")
}

func padNumeric(p string) string {
	const width = 8
	if len(p) >= width {
		return p
	}
	return strings.Repeat("0", width-len(p)) + p
}

// ClientTooOld reports whether the client version is below MinClientVersion.
func ClientTooOld(clientVersion string) bool {
	if clientVersion == "" {
		return false
	}
	return CompareSemver(clientVersion, MinClientVersion) < 0
}

// ServerTooOld reports whether the server version is below MinServerVersion.
func ServerTooOld(serverVersion string) bool {
	if serverVersion == "" {
		return false
	}
	return CompareSemver(serverVersion, MinServerVersion) < 0
}
