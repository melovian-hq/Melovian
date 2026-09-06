// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

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

	"melovian/internal/httputil"
	"melovian/internal/osutil"
	"melovian/internal/store"
)

const maxDownloadBytes = 512 << 20 // 512 MiB safety ceiling per track

func (s *Server) registerDownloadRoutes() {
	s.registerDownloadExportRoutes()
	s.mux.HandleFunc("GET /api/downloads", s.handleListDownloads)
	s.mux.HandleFunc("GET /api/downloads/dir", s.handleGetDownloadDir)
	s.mux.HandleFunc("POST /api/downloads/reveal", s.handleRevealDownloadDir)
	s.mux.HandleFunc("GET /api/music/settings/cache", s.handleGetCacheSettings)
	s.mux.HandleFunc("PUT /api/music/settings/cache", s.handlePutCacheSettings)
	s.mux.HandleFunc("DELETE /api/downloads", s.handleClearDownloads)
	s.mux.HandleFunc("POST /api/downloads/{trackId}", s.handleCreateDownload)
	s.mux.HandleFunc("DELETE /api/downloads/{trackId}", s.handleDeleteDownload)
	s.mux.HandleFunc("GET /api/downloads/{trackId}/stream", s.handleStreamDownload)
}

func (s *Server) downloadDir(instanceID string) string {
	return filepath.Join(s.cfg.DataDir, "downloads", instanceID)
}

func downloadFileName(trackID string) string {
	sum := sha256.Sum256([]byte(trackID))
	return hex.EncodeToString(sum[:])
}

func (s *Server) downloadTrackMu(instanceID, trackID string) *sync.Mutex {
	key := instanceID + "\x00" + trackID
	actual, _ := s.downloadLocks.LoadOrStore(key, &sync.Mutex{})
	return actual.(*sync.Mutex)
}

func (s *Server) openDownloadPath(instanceID, path string) (*os.File, error) {
	jail := s.downloadDir(instanceID)
	// Canonicalize the jail so a symlinked data dir (macOS /var -> /private/var)
	// matches the resolved file path below.
	if resolved, err := filepath.EvalSymlinks(jail); err == nil {
		jail = resolved
	}
	cleaned := filepath.Clean(path)
	if err := osutil.PathEscapesRoot(jail, cleaned); err != nil {
		return nil, os.ErrNotExist
	}
	if resolved, err := filepath.EvalSymlinks(cleaned); err == nil {
		cleaned = resolved
		if err := osutil.PathEscapesRoot(jail, cleaned); err != nil {
			return nil, os.ErrNotExist
		}
	}
	return os.Open(cleaned) //#nosec G304 -- path re-checked against download jail
}

func (s *Server) handleListDownloads(w http.ResponseWriter, r *http.Request) {
	instanceID := InstanceIDFromContext(r.Context())
	items, err := s.downloads.List(instanceID)
	if err != nil {
		http.Error(w, "list downloads: "+err.Error(), http.StatusInternalServerError)
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

func (s *Server) handleGetDownloadDir(w http.ResponseWriter, r *http.Request) {
	instanceID := InstanceIDFromContext(r.Context())
	dir := s.downloadDir(instanceID)
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"path": dir})
}

func (s *Server) handleRevealDownloadDir(w http.ResponseWriter, r *http.Request) {
	instanceID := InstanceIDFromContext(r.Context())
	dir := s.downloadDir(instanceID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		http.Error(w, "create cache dir: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := osutil.RevealPath(dir); err != nil {
		http.Error(w, "open folder: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateDownload(w http.ResponseWriter, r *http.Request) {
	trackID := r.PathValue("trackId")
	if trackID == "" {
		http.Error(w, "missing track id", http.StatusBadRequest)
		return
	}
	if strings.HasPrefix(trackID, "trk_") {
		http.Error(w, "local library tracks are already stored on disk", http.StatusBadRequest)
		return
	}
	instanceID := InstanceIDFromContext(r.Context())
	userID := ResolveProgressUserID(r.Context())

	mu := s.downloadTrackMu(instanceID, trackID)
	mu.Lock()
	defer mu.Unlock()

	settings, err := s.loadCacheSettings(userID)
	if err != nil {
		http.Error(w, "load cache settings: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !settings.Enabled {
		http.Error(w, "track caching is disabled", http.StatusForbidden)
		return
	}

	if existing, err := s.downloads.Get(instanceID, trackID); err == nil {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{
			"trackId":     trackID,
			"size":        existing.Size,
			"contentType": existing.ContentType,
			"cached":      true,
		})
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "lookup cache: "+err.Error(), http.StatusInternalServerError)
		return
	}

	client := s.subsonicForContext(r.Context())
	if !client.Enabled() {
		http.Error(w, "no subsonic instance configured", http.StatusServiceUnavailable)
		return
	}

	if err := s.evictDownloads(instanceID, settings.LimitBytes, maxDownloadBytes); err != nil {
		http.Error(w, "evict cache: "+err.Error(), http.StatusInternalServerError)
		return
	}

	select {
	case s.downloadSem <- struct{}{}:
		defer func() { <-s.downloadSem }()
	case <-r.Context().Done():
		return
	}

	body, contentType, err := client.Stream(trackID)
	if err != nil {
		http.Error(w, "download failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer func() { _ = body.Close() }()

	dir := s.downloadDir(instanceID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		http.Error(w, "create cache dir: "+err.Error(), http.StatusInternalServerError)
		return
	}

	path := filepath.Join(dir, downloadFileName(trackID))
	tmp := path + ".part"
	file, err := os.Create(tmp) //#nosec G304,G703 -- path derived from hashed track id within app data dir
	if err != nil {
		http.Error(w, "create cache file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	size, err := io.Copy(file, io.LimitReader(body, maxDownloadBytes))
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		_ = os.Remove(tmp) //#nosec G703 -- tmp is under download jail
		http.Error(w, "write cache file", http.StatusInternalServerError)
		return
	}
	if err := os.Rename(tmp, path); err != nil { //#nosec G703 -- path is under download jail
		_ = os.Remove(tmp) //#nosec G703 -- tmp is under download jail
		http.Error(w, "finalize cache file: "+err.Error(), http.StatusInternalServerError)
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
	if err := s.downloads.Upsert(entry); err != nil {
		_ = os.Remove(path) //#nosec G703 -- path is under download jail
		http.Error(w, "record download: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.evictDownloads(instanceID, settings.LimitBytes, 0); err != nil {
		http.Error(w, "trim cache: "+err.Error(), http.StatusInternalServerError)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"trackId":     trackID,
		"size":        size,
		"contentType": contentType,
	})
}

func (s *Server) handleDeleteDownload(w http.ResponseWriter, r *http.Request) {
	trackID := r.PathValue("trackId")
	instanceID := InstanceIDFromContext(r.Context())
	path, err := s.downloads.Delete(instanceID, trackID)
	if err != nil {
		http.Error(w, "delete download: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if path != "" {
		_ = os.Remove(path) //#nosec G703 -- path comes from download store under data dir
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleStreamDownload(w http.ResponseWriter, r *http.Request) {
	trackID := r.PathValue("trackId")
	instanceID := InstanceIDFromContext(r.Context())
	entry, err := s.downloads.Get(instanceID, trackID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "not downloaded", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "lookup download: "+err.Error(), http.StatusInternalServerError)
		return
	}

	file, err := s.openDownloadPath(instanceID, entry.Path)
	if err != nil {
		http.Error(w, "open cache file: "+err.Error(), http.StatusNotFound)
		return
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		http.Error(w, "stat cache file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	if entry.ContentType != "" {
		w.Header().Set("Content-Type", entry.ContentType)
	}
	http.ServeContent(w, r, filepath.Base(entry.Path), info.ModTime(), file)
}

func (s *Server) evictDownloads(instanceID string, limitBytes int64, reserveBytes int64) error {
	if limitBytes <= 0 {
		return nil
	}
	totalSize, _, err := s.downloads.Stats(instanceID)
	if err != nil {
		return err
	}
	target := max(limitBytes-reserveBytes, 0)
	if totalSize <= target {
		return nil
	}

	paths, err := s.downloads.EvictUntil(instanceID, target)
	if err != nil {
		return err
	}
	for _, path := range paths {
		_ = os.Remove(path)
	}
	return nil
}
