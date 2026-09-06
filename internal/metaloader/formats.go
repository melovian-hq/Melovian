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
