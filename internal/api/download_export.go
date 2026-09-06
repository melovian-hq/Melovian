// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"melovian/internal/brand"
)

func (s *Server) registerDownloadExportRoutes() {
	s.mux.HandleFunc("GET /api/downloads/{trackId}/export", s.handleExportDownload)
	s.mux.HandleFunc("GET /api/downloads/export.zip", s.handleExportAllDownloads)
}

func downloadExportBasename(title, artist string) string {
	title = strings.TrimSpace(title)
	artist = strings.TrimSpace(artist)
	name := title
	if artist != "" && title != "" {
		name = artist + " - " + title
	} else if artist != "" {
		name = artist
	}
	return sanitizeDownloadFilename(name)
}

func sanitizeDownloadFilename(name string) string {
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

func contentTypeExtension(contentType string) string {
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

func downloadExportFilename(title, artist, contentType string) string {
	base := downloadExportBasename(title, artist)
	return base + contentTypeExtension(contentType)
}

func setAttachmentFilename(w http.ResponseWriter, filename string) {
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
}

func (s *Server) handleExportDownload(w http.ResponseWriter, r *http.Request) {
	trackID := r.PathValue("trackId")
	instanceID := InstanceIDFromContext(r.Context())
	entry, err := s.downloads.Get(instanceID, trackID)
	if err != nil {
		http.Error(w, "not downloaded", http.StatusNotFound)
		return
	}

	file, err := s.openDownloadPath(instanceID, entry.Path)
	if err != nil {
		http.Error(w, "open cache file", http.StatusNotFound)
		return
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		http.Error(w, "stat cache file", http.StatusInternalServerError)
		return
	}

	filename := downloadExportFilename(entry.TrackTitle, entry.ArtistName, entry.ContentType)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if entry.ContentType != "" {
		w.Header().Set("Content-Type", entry.ContentType)
	}
	setAttachmentFilename(w, filename)
	http.ServeContent(w, r, filename, info.ModTime(), file)
}

func (s *Server) handleExportAllDownloads(w http.ResponseWriter, r *http.Request) {
	instanceID := InstanceIDFromContext(r.Context())
	items, err := s.downloads.List(instanceID)
	if err != nil {
		http.Error(w, "list downloads: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if len(items) == 0 {
		http.Error(w, "no downloads", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	setAttachmentFilename(w, brand.Slug+"-downloads.zip")

	zipWriter := zip.NewWriter(w)
	usedNames := make(map[string]int, len(items))

	for _, item := range items {
		file, err := s.openDownloadPath(instanceID, item.Path)
		if err != nil {
			continue
		}

		filename := downloadExportFilename(item.TrackTitle, item.ArtistName, item.ContentType)
		if count := usedNames[filename]; count > 0 {
			ext := filepath.Ext(filename)
			base := strings.TrimSuffix(filename, ext)
			filename = fmt.Sprintf("%s (%d)%s", base, count+1, ext)
		}
		usedNames[filename]++

		header := &zip.FileHeader{
			Name:     filename,
			Method:   zip.Deflate,
			Modified: time.Unix(item.CreatedAt, 0),
		}
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			_ = file.Close()
			continue
		}
		_, _ = io.Copy(writer, file)
		_ = file.Close()
	}

	if err := zipWriter.Close(); err != nil {
		return
	}
}
