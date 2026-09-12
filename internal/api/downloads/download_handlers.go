// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package downloads

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"melovian/internal/api/apishared"
	"melovian/internal/api/library"
	"melovian/internal/appconfig"
	"melovian/internal/httputil"
	"melovian/internal/osutil"
	"melovian/internal/store"
)

// Handler serves the download, cache-settings, and media download routes.
type Handler struct {
	cfg            appconfig.Config
	downloads      *store.DownloadStore
	preferences    *store.PreferencesStore
	listen         *store.ListenStore
	localLibraries *store.LocalLibraryStore
	localTracks    *store.LocalTrackStore
	resolver       *apishared.Resolver
	library        *library.Handler
	downloadSem    chan struct{}
	downloadLocks  sync.Map
}

type Deps struct {
	Config      appconfig.Config
	Downloads   *store.DownloadStore
	Preferences *store.PreferencesStore
	Listen      *store.ListenStore
	Libraries   *store.LocalLibraryStore
	Tracks      *store.LocalTrackStore
	Resolver    *apishared.Resolver
	Library     *library.Handler
	DownloadSem chan struct{}
}

func New(d Deps) *Handler {
	return &Handler{
		cfg:            d.Config,
		downloads:      d.Downloads,
		preferences:    d.Preferences,
		listen:         d.Listen,
		localLibraries: d.Libraries,
		localTracks:    d.Tracks,
		resolver:       d.Resolver,
		library:        d.Library,
		downloadSem:    d.DownloadSem,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	h.registerDownloadRoutes(mux)
	h.registerMediaDownloadRoutes(mux)
}

func (h *Handler) registerDownloadRoutes(mux *http.ServeMux) {
	h.registerDownloadExportRoutes(mux)
	mux.HandleFunc("GET /api/downloads", h.handleListDownloads)
	mux.HandleFunc("GET /api/downloads/dir", h.handleGetDownloadDir)
	mux.HandleFunc("POST /api/downloads/reveal", h.handleRevealDownloadDir)
	mux.HandleFunc("GET /api/music/settings/cache", h.handleGetCacheSettings)
	mux.HandleFunc("PUT /api/music/settings/cache", h.handlePutCacheSettings)
	mux.HandleFunc("DELETE /api/downloads", h.handleClearDownloads)
	mux.HandleFunc("POST /api/downloads/{trackId}", h.handleCreateDownload)
	mux.HandleFunc("DELETE /api/downloads/{trackId}", h.handleDeleteDownload)
	mux.HandleFunc("GET /api/downloads/{trackId}/stream", h.handleStreamDownload)
}

func (h *Handler) downloadDir(instanceID string) string {
	return filepath.Join(h.cfg.DataDir, "downloads", instanceID)
}

func downloadFileName(trackID string) string {
	sum := sha256.Sum256([]byte(trackID))
	return hex.EncodeToString(sum[:])
}

func (h *Handler) downloadTrackMu(instanceID, trackID string) *sync.Mutex {
	key := instanceID + "\x00" + trackID
	actual, _ := h.downloadLocks.LoadOrStore(key, &sync.Mutex{})
	return actual.(*sync.Mutex)
}

func (h *Handler) openDownloadPath(instanceID, path string) (*os.File, error) {
	jail := h.downloadDir(instanceID)
	cleaned := filepath.Clean(path)
	// Canonicalize both sides so a symlinked data dir (macOS /var ->
	// /private/var) compares against the resolved file path.
	resolved, err := osutil.ResolveInside(cleaned, jail)
	if err != nil {
		return nil, os.ErrNotExist
	}
	return os.Open(resolved) //#nosec G304 -- path re-checked against download jail
}

func (h *Handler) handleListDownloads(w http.ResponseWriter, r *http.Request) {
	instanceID := apishared.InstanceIDFromContext(r.Context())
	items, err := h.downloads.List(instanceID)
	if err != nil {
		httputil.WriteInternalError(w, r, "list downloads:", err)
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, t := range items {
		out = append(out, map[string]any{
			"trackId":     t.TrackID,
			"size":        t.Size,
			"contentType": t.ContentType,
			"trackTitle":  t.TrackTitle,
			"artistName":  t.ArtistName,
			"createdAt":   t.CreatedAt,
		})
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"downloads": out})
}

func (h *Handler) handleGetDownloadDir(w http.ResponseWriter, r *http.Request) {
	instanceID := apishared.InstanceIDFromContext(r.Context())
	dir := h.downloadDir(instanceID)
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"path": dir})
}

func (h *Handler) handleRevealDownloadDir(w http.ResponseWriter, r *http.Request) {
	instanceID := apishared.InstanceIDFromContext(r.Context())
	dir := h.downloadDir(instanceID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		httputil.WriteInternalError(w, r, "create cache dir:", err)
		return
	}
	if err := osutil.RevealPath(dir); err != nil {
		httputil.WriteInternalError(w, r, "open folder:", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleCreateDownload(w http.ResponseWriter, r *http.Request) {
	trackID := r.PathValue("trackId")
	if trackID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "missing_track_id", "missing track id")
		return
	}
	if strings.HasPrefix(trackID, "trk_") {
		httputil.WriteError(w, http.StatusBadRequest, "local_library_tracks_are_already_stored_", "local library tracks are already stored on disk")
		return
	}
	instanceID := apishared.InstanceIDFromContext(r.Context())
	userID := apishared.ResolveProgressUserID(r.Context())

	mu := h.downloadTrackMu(instanceID, trackID)
	mu.Lock()
	defer mu.Unlock()

	settings, err := h.loadCacheSettings(userID)
	if err != nil {
		httputil.WriteInternalError(w, r, "load cache settings:", err)
		return
	}
	if !settings.Enabled {
		httputil.WriteError(w, http.StatusForbidden, "track_caching_is_disabled", "track caching is disabled")
		return
	}

	if existing, err := h.downloads.Get(instanceID, trackID); err == nil {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{
			"trackId":     trackID,
			"size":        existing.Size,
			"contentType": existing.ContentType,
			"cached":      true,
		})
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		httputil.WriteInternalError(w, r, "lookup cache:", err)
		return
	}

	client := h.resolver.ForContext(r.Context())
	if !client.Enabled() {
		httputil.WriteError(w, http.StatusServiceUnavailable, "service_unavailable", "no subsonic instance configured")
		return
	}

	if err := h.EvictDownloads(instanceID, settings.LimitBytes, apishared.MaxDownloadBytes); err != nil {
		httputil.WriteInternalError(w, r, "evict cache:", err)
		return
	}

	select {
	case h.downloadSem <- struct{}{}:
		defer func() { <-h.downloadSem }()
	case <-r.Context().Done():
		return
	}

	body, contentType, err := client.Stream(trackID)
	if err != nil {
		httputil.WriteInternalError(w, r, "download failed:", err)
		return
	}
	defer func() { _ = body.Close() }()

	dir := h.downloadDir(instanceID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		httputil.WriteInternalError(w, r, "create cache dir:", err)
		return
	}

	path := filepath.Join(dir, downloadFileName(trackID))
	tmp := path + ".part"
	file, err := os.Create(tmp) //#nosec G304,G703 -- path derived from hashed track id within app data dir
	if err != nil {
		httputil.WriteInternalError(w, r, "create cache file:", err)
		return
	}

	size, err := io.Copy(file, io.LimitReader(body, apishared.MaxDownloadBytes))
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		_ = os.Remove(tmp) //#nosec G703 -- tmp is under download jail
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "write cache file")
		return
	}
	if err := os.Rename(tmp, path); err != nil { //#nosec G703 -- path is under download jail
		_ = os.Remove(tmp) //#nosec G703 -- tmp is under download jail
		httputil.WriteInternalError(w, r, "finalize cache file:", err)
		return
	}

	entry := store.DownloadedTrack{
		InstanceID:  instanceID,
		TrackID:     trackID,
		Path:        path,
		ContentType: contentType,
		Size:        size,
		TrackTitle:  r.URL.Query().Get("title"),
		ArtistName:  r.URL.Query().Get("artist"),
		CreatedAt:   time.Now().Unix(),
	}
	if err := h.downloads.Upsert(entry); err != nil {
		_ = os.Remove(path) //#nosec G703 -- path is under download jail
		httputil.WriteInternalError(w, r, "record download:", err)
		return
	}

	if err := h.EvictDownloads(instanceID, settings.LimitBytes, 0); err != nil {
		httputil.WriteInternalError(w, r, "trim cache:", err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"trackId":     trackID,
		"size":        size,
		"contentType": contentType,
	})
}

func (h *Handler) handleDeleteDownload(w http.ResponseWriter, r *http.Request) {
	trackID := r.PathValue("trackId")
	instanceID := apishared.InstanceIDFromContext(r.Context())
	path, err := h.downloads.Delete(instanceID, trackID)
	if err != nil {
		httputil.WriteInternalError(w, r, "delete download:", err)
		return
	}
	if path != "" {
		_ = os.Remove(path) //#nosec G703 -- path comes from download store under data dir
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleStreamDownload(w http.ResponseWriter, r *http.Request) {
	trackID := r.PathValue("trackId")
	instanceID := apishared.InstanceIDFromContext(r.Context())
	entry, err := h.downloads.Get(instanceID, trackID)
	if errors.Is(err, sql.ErrNoRows) {
		httputil.WriteError(w, http.StatusNotFound, "not_downloaded", "not downloaded")
		return
	}
	if err != nil {
		httputil.WriteInternalError(w, r, "lookup download:", err)
		return
	}

	file, err := h.openDownloadPath(instanceID, entry.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "open cache file: "+err.Error())
		return
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "stat cache file")
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	if entry.ContentType != "" {
		w.Header().Set("Content-Type", entry.ContentType)
	}
	http.ServeContent(w, r, filepath.Base(entry.Path), info.ModTime(), file)
}

func (h *Handler) EvictDownloads(instanceID string, limitBytes int64, reserveBytes int64) error {
	if limitBytes <= 0 {
		return nil
	}
	totalSize, _, err := h.downloads.Stats(instanceID)
	if err != nil {
		return err
	}
	target := max(limitBytes-reserveBytes, 0)
	if totalSize <= target {
		return nil
	}

	paths, err := h.downloads.EvictUntil(instanceID, target)
	if err != nil {
		return err
	}
	for _, path := range paths {
		_ = os.Remove(path)
	}
	return nil
}
