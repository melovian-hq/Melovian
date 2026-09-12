// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metaloader

import (
	"path/filepath"
	"strings"

	"melovian/internal/store"
)

var audioExtensions = map[string]struct{}{
	".mp3":  {},
	".mp2":  {},
	".mp1":  {},
	".flac": {},
	".ogg":  {},
	".oga":  {},
	".opus": {},
	".m4a":  {},
	".m4b":  {},
	".mka":  {},
	".aac":  {},
	".wav":  {},
	".wma":  {},
	".aiff": {},
	".aif":  {},
	".aifc": {},
	".ape":  {},
	".mpc":  {},
	".wv":   {},
	".dsf":  {},
	".dsd":  {},
	".tta":  {},
	".mogg": {},
}

var videoExtensions = map[string]struct{}{
	".mp4":  {},
	".m4v":  {},
	".webm": {},
}

func IsAudioFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	_, ok := audioExtensions[ext]
	return ok
}

func IsVideoFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	_, ok := videoExtensions[ext]
	return ok
}

func IsMediaFile(name string) bool {
	return IsAudioFile(name) || IsVideoFile(name)
}

func MediaKindFromPath(name string) string {
	if IsVideoFile(name) {
		return store.MediaKindVideo
	}
	return store.MediaKindAudio
}

func FormatFromPath(name string) string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
	if ext == "" {
		return "unknown"
	}
	return ext
}

// ContentTypeExtension maps an audio or video MIME type to the file
// extension used for downloads and exports. Extensions come from the
// canonical audioExtensions and videoExtensions lists above, with two
// export-only additions: .mkv for Matroska video the scanner does not
// index, and the .audio fallback for unrecognized audio types.
func ContentTypeExtension(contentType string) string {
	lower := strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case strings.Contains(lower, "video/mp4"), strings.HasPrefix(lower, "video/"):
		if strings.Contains(lower, "webm") {
			return ".webm"
		}
		if strings.Contains(lower, "mkv") || strings.Contains(lower, "x-matroska") {
			return ".mkv"
		}
		return ".mp4"
	case strings.Contains(lower, "mpeg"), strings.Contains(lower, "mp3"):
		return ".mp3"
	case strings.Contains(lower, "flac"):
		return ".flac"
	case strings.Contains(lower, "ogg"):
		return ".ogg"
	case strings.Contains(lower, "opus"):
		return ".opus"
	case strings.Contains(lower, "wav"):
		return ".wav"
	case strings.Contains(lower, "m4a"):
		return ".m4a"
	case strings.Contains(lower, "mp4"), strings.Contains(lower, "aac"):
		// audio/mp4 and bare aac without video/
		if strings.Contains(lower, "aac") && !strings.Contains(lower, "mp4") {
			return ".aac"
		}
		return ".m4a"
	default:
		return ".audio"
	}
}
