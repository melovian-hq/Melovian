// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package apishared

import (
	"fmt"
	"net/http"
	"strings"

	"melovian/internal/metaloader"
)

// MaxDownloadBytes is the safety ceiling per downloaded track.
const MaxDownloadBytes = 512 << 20 // 512 MiB

func DownloadExportBasename(title, artist string) string {
	title = strings.TrimSpace(title)
	artist = strings.TrimSpace(artist)
	name := title
	if artist != "" && title != "" {
		name = artist + " - " + title
	} else if artist != "" {
		name = artist
	}
	return SanitizeDownloadFilename(name)
}

func SanitizeDownloadFilename(name string) string {
	replacer := strings.NewReplacer(
		"\\", "_",
		"/", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	cleaned := strings.TrimSpace(replacer.Replace(name))
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if cleaned == "" {
		return "track"
	}
	if len(cleaned) > 180 {
		cleaned = cleaned[:180]
	}
	return cleaned
}

func ContentTypeExtension(contentType string) string {
	return metaloader.ContentTypeExtension(contentType)
}

func DownloadExportFilename(title, artist, contentType string) string {
	base := DownloadExportBasename(title, artist)
	return base + ContentTypeExtension(contentType)
}

func SetAttachmentFilename(w http.ResponseWriter, filename string) {
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
}
