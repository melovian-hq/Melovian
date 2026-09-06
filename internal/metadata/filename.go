// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// FilenameSuggestion is metadata inferred from a file path.
type FilenameSuggestion struct {
	Source      string `json:"source"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	AlbumArtist string `json:"albumArtist"`
	TrackNum    int    `json:"trackNum"`
	DiscNum     int    `json:"discNum"`
}

var (
	trackPrefixPattern = regexp.MustCompile(`^(\d{1,3})[\s._-]+(.+)$`)
	discTrackPattern   = regexp.MustCompile(`^(\d{1,2})-(\d{1,3})[\s._-]+(.+)$`)
)

// ParseFilenameSuggestion extracts likely tags from a relative audio path.
func ParseFilenameSuggestion(relPath string) FilenameSuggestion {
	relPath = filepath.ToSlash(strings.TrimSpace(relPath))
	if relPath == "" {
		return FilenameSuggestion{Source: "filename"}
	}

	parts := strings.Split(relPath, "/")
	fileName := parts[len(parts)-1]
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	base = strings.TrimSpace(base)
	if base == "" {
		return FilenameSuggestion{Source: "filename"}
	}

	suggestion := FilenameSuggestion{Source: "filename"}

	if m := discTrackPattern.FindStringSubmatch(base); len(m) == 4 {
		suggestion.DiscNum, _ = strconv.Atoi(m[1])
		suggestion.TrackNum, _ = strconv.Atoi(m[2])
		base = strings.TrimSpace(m[3])
	} else if m := trackPrefixPattern.FindStringSubmatch(base); len(m) == 3 {
		suggestion.TrackNum, _ = strconv.Atoi(m[1])
		base = strings.TrimSpace(m[2])
	}

	if artist, title, ok := splitArtistTitle(base); ok {
		suggestion.Artist = artist
		suggestion.Title = title
	} else {
		suggestion.Title = base
	}

	switch len(parts) {
	case 2:
		if suggestion.Artist == "" {
			suggestion.Artist = cleanPathSegment(parts[0])
		}
	case 3:
		if suggestion.Artist == "" {
			suggestion.Artist = cleanPathSegment(parts[0])
		}
		suggestion.Album = cleanPathSegment(parts[1])
		if suggestion.AlbumArtist == "" {
			suggestion.AlbumArtist = suggestion.Artist
		}
	case 4:
		if suggestion.Artist == "" {
			suggestion.Artist = cleanPathSegment(parts[0])
		}
		suggestion.Album = cleanPathSegment(parts[1])
		if suggestion.AlbumArtist == "" {
			suggestion.AlbumArtist = suggestion.Artist
		}
	}

	return suggestion
}

func splitArtistTitle(value string) (artist, title string, ok bool) {
	separators := []string{" - ", " – ", " — ", " _ "}
	for _, sep := range separators {
		if idx := strings.Index(value, sep); idx > 0 {
			left := strings.TrimSpace(value[:idx])
			right := strings.TrimSpace(value[idx+len(sep):])
			if left != "" && right != "" {
				return left, right, true
			}
		}
	}
	return "", "", false
}

func cleanPathSegment(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "_", " ")
	return strings.TrimSpace(value)
}

// SuggestionToMatch converts a filename suggestion into a lookup-style match.
func SuggestionToMatch(suggestion FilenameSuggestion) LookupMatch {
	return LookupMatch{
		ID:          "filename",
		Source:      suggestion.Source,
		Title:       suggestion.Title,
		Artist:      suggestion.Artist,
		Album:       suggestion.Album,
		AlbumArtist: FirstNonEmpty(suggestion.AlbumArtist, suggestion.Artist),
		TrackNum:    suggestion.TrackNum,
		Year:        0,
		Genre:       "",
	}
}

// FirstNonEmpty returns the first non-empty trimmed string.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
