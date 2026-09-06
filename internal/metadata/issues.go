// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"slices"
	"strings"

	"melovian/internal/store"
)

const (
	IssueUnknownArtist = "unknown-artist"
	IssueUnknownAlbum  = "unknown-album"
	IssueMissingTitle  = "missing-title"
	IssueAny           = "any"
)

var unknownArtistKeys = map[string]struct{}{
	"":               {},
	"unknown":        {},
	"unknown artist": {},
	"unknownartist":  {},
}

var unknownAlbumKeys = map[string]struct{}{
	"":              {},
	"unknown":       {},
	"unknown album": {},
	"unknownalbum":  {},
}

func normalizeKey(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func IsUnknownArtist(name string) bool {
	_, ok := unknownArtistKeys[normalizeKey(name)]
	return ok
}

func IsUnknownAlbum(name string) bool {
	_, ok := unknownAlbumKeys[normalizeKey(name)]
	return ok
}

func TrackIssues(track store.LocalTrack) []string {
	var issues []string
	if IsUnknownArtist(track.Artist) {
		issues = append(issues, IssueUnknownArtist)
	}
	if IsUnknownAlbum(track.Album) {
		issues = append(issues, IssueUnknownAlbum)
	}
	if strings.TrimSpace(track.Title) == "" {
		issues = append(issues, IssueMissingTitle)
	}
	return issues
}

func TrackHasIssue(track store.LocalTrack, issue string) bool {
	if issue == "" || issue == IssueAny {
		return len(TrackIssues(track)) > 0
	}
	return slices.Contains(TrackIssues(track), issue)
}
