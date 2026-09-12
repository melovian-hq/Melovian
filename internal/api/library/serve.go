// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package library

import (
	"context"
	"melovian/internal/api/apishared"
	"melovian/internal/httputil"
	"melovian/internal/localmusic"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// callerMayAccessLocalTrack reports whether the request user may read a local track.
// When auth is off, any present library is allowed (single-tenant desktop).
func (h *Handler) CallerMayAccessLocalTrack(ctx context.Context, trackID string) bool {
	track, err := h.localTracks.GetByID(trackID)
	if err != nil {
		return false
	}
	userID := apishared.UserIDFromContext(ctx)
	if userID != "" {
		_, err := h.localLibraries.GetForUser(userID, track.LibraryID)
		return err == nil
	}
	_, err = h.localLibraries.Get(track.LibraryID)
	return err == nil
}

func (h *Handler) ServeLocalTrackFile(w http.ResponseWriter, r *http.Request, trackID string, asDownload bool) {
	track, err := h.localTracks.GetByID(trackID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	lib, err := h.localLibraries.Get(track.LibraryID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	path, err := localmusic.ResolveTrackPath(lib.Path, track.AbsPath)
	if err != nil {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
		return
	}
	file, err := os.Open(path) //#nosec G304 -- path resolved under library jail
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = file.Close() }()
	contentType := localmusic.ContentTypeForFormat(track.Format)
	w.Header().Set("Content-Type", contentType)
	if asDownload {
		filename := apishared.DownloadExportFilename(track.Title, track.Artist, contentType)
		if ext := filepath.Ext(path); ext != "" && len(ext) <= 8 {
			base := strings.TrimSuffix(filename, filepath.Ext(filename))
			filename = base + strings.ToLower(ext)
		}
		apishared.SetAttachmentFilename(w, filename)
	}
	http.ServeContent(w, r, filepath.Base(path), track.UpdatedAt, file)
}
