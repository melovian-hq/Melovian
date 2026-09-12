// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package downloads

import (
	"archive/zip"
	"fmt"
	"io"
	"melovian/internal/api/apishared"
	"melovian/internal/httputil"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"melovian/internal/brand"
)

func (h *Handler) registerDownloadExportRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/downloads/{trackId}/export", h.handleExportDownload)
	mux.HandleFunc("GET /api/downloads/export.zip", h.handleExportAllDownloads)
}

func (h *Handler) handleExportDownload(w http.ResponseWriter, r *http.Request) {
	trackID := r.PathValue("trackId")
	instanceID := apishared.InstanceIDFromContext(r.Context())
	entry, err := h.downloads.Get(instanceID, trackID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_downloaded", "not downloaded")
		return
	}

	file, err := h.openDownloadPath(instanceID, entry.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "open_cache_file", "open cache file")
		return
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "stat cache file")
		return
	}

	filename := apishared.DownloadExportFilename(entry.TrackTitle, entry.ArtistName, entry.ContentType)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if entry.ContentType != "" {
		w.Header().Set("Content-Type", entry.ContentType)
	}
	apishared.SetAttachmentFilename(w, filename)
	http.ServeContent(w, r, filename, info.ModTime(), file)
}

func (h *Handler) handleExportAllDownloads(w http.ResponseWriter, r *http.Request) {
	instanceID := apishared.InstanceIDFromContext(r.Context())
	items, err := h.downloads.List(instanceID)
	if err != nil {
		httputil.WriteInternalError(w, r, "list downloads:", err)
		return
	}
	if len(items) == 0 {
		httputil.WriteError(w, http.StatusNotFound, "no_downloads", "no downloads")
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	apishared.SetAttachmentFilename(w, brand.Slug+"-downloads.zip")

	zipWriter := zip.NewWriter(w)
	usedNames := make(map[string]int, len(items))

	for _, item := range items {
		file, err := h.openDownloadPath(instanceID, item.Path)
		if err != nil {
			continue
		}

		filename := apishared.DownloadExportFilename(item.TrackTitle, item.ArtistName, item.ContentType)
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
