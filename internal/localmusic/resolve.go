// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package localmusic

import (
	"fmt"
	"path/filepath"
	"strings"

	"melovian/internal/osutil"
	"melovian/internal/store"
)

func LoadCatalog(tracks *store.LocalTrackStore, libraryID string) (Catalog, error) {
	items, err := tracks.ListForCatalog(libraryID)
	if err != nil {
		return Catalog{}, err
	}
	return BuildCatalog(items), nil
}

func ResolveTrackPath(libraryRoot, absPath string) (string, error) {
	root := filepath.Clean(libraryRoot)
	path := filepath.Clean(absPath)
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("track path is not absolute")
	}
	if err := osutil.PathEscapesRoot(root, path); err != nil {
		return "", fmt.Errorf("track path escapes library root")
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		resolvedRoot = root
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path, nil
	}
	if err := osutil.PathEscapesRoot(resolvedRoot, resolvedPath); err != nil {
		return "", fmt.Errorf("track path escapes library root")
	}
	return resolvedPath, nil
}

func ContentTypeForFormat(format string) string {
	switch strings.ToLower(format) {
	case "mp3", "mp2", "mp1":
		return "audio/mpeg"
	case "flac":
		return "audio/flac"
	case "ogg", "oga", "mogg":
		return "audio/ogg"
	case "opus":
		return "audio/opus"
	case "m4a", "m4b", "aac":
		return "audio/mp4"
	case "mka":
		return "audio/x-matroska"
	case "wav":
		return "audio/wav"
	case "wma":
		return "audio/x-ms-wma"
	case "aiff", "aif", "aifc":
		return "audio/aiff"
	case "ape":
		return "audio/x-ape"
	case "mpc":
		return "audio/x-musepack"
	case "wv":
		return "audio/x-wavpack"
	case "dsf":
		return "audio/x-dsf"
	case "dsd":
		return "audio/x-dsd"
	case "tta":
		return "audio/x-tta"
	case "mp4", "m4v":
		return "video/mp4"
	case "webm":
		return "video/webm"
	default:
		return "application/octet-stream"
	}
}
