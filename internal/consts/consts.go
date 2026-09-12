// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// Package consts holds shared timeouts, limits, and protocol constants that
// were previously hardcoded at call sites.
package consts

import "time"

// SubsonicVersion is the Subsonic API version Melovian speaks, both as a
// client (internal/subsonic) and as a server (internal/subsonicserver,
// internal/democatalog).
const SubsonicVersion = "1.16.1"

// HTTP request timeouts for outbound and proxied calls.
const (
	// UpstreamTimeout covers short outbound API calls (OIDC, scrobblers,
	// video lookups) and the HTTP server ReadHeaderTimeout.
	UpstreamTimeout = 15 * time.Second
	// CoverFetchTimeout covers fetching remote cover art for local music.
	CoverFetchTimeout = 12 * time.Second
	// MetadataFetchTimeout covers local metadata and smart playlist queries.
	MetadataFetchTimeout = 20 * time.Second
	// SmartPlaylistPreviewTimeout covers smart playlist preview evaluation.
	SmartPlaylistPreviewTimeout = 8 * time.Second
)

// Rate limits for sensitive endpoints (requests per window).
const (
	AuthRateLimit       = 10
	AuthRateWindow      = 5 * time.Minute
	ShareRateLimit      = 10
	ShareRateWindow     = 5 * time.Minute
	ClientLogRateLimit  = 60
	ClientLogRateWindow = time.Minute
)
