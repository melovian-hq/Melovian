// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"melovian/internal/appconfig"
	"melovian/internal/httputil"
	"melovian/internal/metaloader"
	"melovian/internal/store"
)

func (s *Server) registerLocalLibraryRoutes() {
	s.mux.HandleFunc("GET /api/local-libraries", s.handleListLocalLibraries)
	s.mux.HandleFunc("POST /api/local-libraries", s.handleCreateLocalLibrary)
	s.mux.HandleFunc("GET /api/local-libraries/active", s.handleGetActiveLocalLibrary)
	s.mux.HandleFunc("POST /api/local-libraries/{id}/activate", s.handleActivateLocalLibrary)
	s.mux.HandleFunc("POST /api/local-libraries/{id}/scan", s.handleScanLocalLibrary)
	s.mux.HandleFunc("GET /api/local-libraries/{id}", s.handleGetLocalLibrary)
	s.mux.HandleFunc("PUT /api/local-libraries/{id}", s.handleUpdateLocalLibrary)
	s.mux.HandleFunc("DELETE /api/local-libraries/{id}", s.handleDeleteLocalLibrary)
}

type localLibraryRequest struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (s *Server) localLibraryPublicView(lib store.LocalLibrary) map[string]any {
	view := s.localLibraries.PublicView(lib)
	if lib.ScanStatus == "scanning" {
		if progress, ok := s.libraryScanner.Progress(lib.ID); ok {
			view["scanProgress"] = progress
		}
	}
	return view
}

func (s *Server) startLibraryScan(libraryID string) bool {
	return s.libraryScanner.StartScan(libraryID, func(result metaloader.ScanResult, scanErr error) {
		s.invalidateLocalLibraryData(libraryID)
		if scanErr != nil {
			s.emitScanError(libraryID, scanErr.Error())
			return
		}
		if len(result.Errors) > 0 && result.Total == 0 {
			s.emitScanError(libraryID, result.Errors[0])
			return
		}
		s.emitScanComplete(libraryID)
	})
}

func (s *Server) watchLocalLibrary(lib store.LocalLibrary) {
	if s.libraryWatcher == nil {
		return
	}
	if err := s.libraryWatcher.WatchLibrary(lib.ID, lib.Path); err != nil {
		slog.Warn("local library watch failed",
			"library_id", lib.ID,
			"path", lib.Path,
			"err", err,
		)
	}
}

func (s *Server) handleLibraryWatchUpdate(libraryID string) {
	s.invalidateLocalLibraryData(libraryID)
	lib, err := s.localLibraries.Get(libraryID)
	if err != nil {
		return
	}
	s.events.Broadcast(Event{
		Type: EventLibraryUpdated,
		Payload: LibraryUpdatedPayload{
			LibraryID:      lib.ID,
			Name:           lib.Name,
			TrackCount:     lib.TrackCount,
			MissingCount:   lib.MissingCount,
			DuplicateCount: lib.DuplicateCount,
		},
	})
}

func (s *Server) localLibraryConfig() appconfig.LocalLibraryConfig {
	return s.cfg.LocalLibraryEffective()
}

func (s *Server) localLibraryEnabled() bool {
	return s.localLibraryConfig().Enabled
}

func (s *Server) resolveLibraryPath(inputPath string) (string, error) {
	cfg := s.localLibraryConfig()
	if !cfg.AllowCustomPath {
		if cfg.DefaultPath == "" {
			return "", errors.New("custom library paths are disabled")
		}
		return filepath.Clean(cfg.DefaultPath), nil
	}

	path := strings.TrimSpace(inputPath)
	if path == "" && cfg.DefaultPath != "" {
		path = cfg.DefaultPath
	}
	if path == "" {
		return "", errors.New("path is required")
	}
	if !filepath.IsAbs(path) {
		return "", errors.New("path must be absolute")
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("path is not a directory")
	}
	return path, nil
}

func (s *Server) handleListLocalLibraries(w http.ResponseWriter, r *http.Request) {
	if !s.localLibraryEnabled() {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{"libraries": []any{}})
		return
	}

	userID := UserIDFromContext(r.Context())
	items, err := s.localLibraries.ListForUser(userID)
	if err != nil {
		http.Error(w, "failed to list libraries", http.StatusInternalServerError)
		return
	}

	payload := make([]map[string]any, 0, len(items))
	for _, item := range items {
		payload = append(payload, s.localLibraryPublicView(item))
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"libraries": payload})
}

func (s *Server) handleCreateLocalLibrary(w http.ResponseWriter, r *http.Request) {
	if !s.localLibraryEnabled() {
		http.Error(w, "local libraries are disabled", http.StatusForbidden)
		return
	}

	var req localLibraryRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	path, err := s.resolveLibraryPath(req.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID := UserIDFromContext(r.Context())
	if _, err := s.localLibraries.FindByPathForUser(userID, path); err == nil {
		http.Error(w, "a library with this path already exists", http.StatusConflict)
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "failed to validate library path", http.StatusInternalServerError)
		return
	}

	lib, err := s.localLibraries.CreateForUser(userID, store.CreateLocalLibraryInput{
		Name: req.Name,
		Path: path,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.watchLocalLibrary(lib)
	_ = s.localLibraries.SetScanState(lib.ID, "scanning", "")
	if !s.startLibraryScan(lib.ID) {
		_ = s.localLibraries.SetScanState(lib.ID, "error", "failed to start scan")
	}

	lib, _ = s.localLibraries.Get(lib.ID)
	httputil.WriteJSON(w, http.StatusCreated, s.localLibraryPublicView(lib))
}

func (s *Server) handleGetLocalLibrary(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := UserIDFromContext(r.Context())
	lib, err := s.localLibraries.GetForUser(userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to load library", http.StatusInternalServerError)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, s.localLibraryPublicView(lib))
}

func (s *Server) handleGetActiveLocalLibrary(w http.ResponseWriter, r *http.Request) {
	if !s.localLibraryEnabled() {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{})
		return
	}

	userID := UserIDFromContext(r.Context())
	lib, err := s.localLibraries.GetActiveForUser(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteJSON(w, http.StatusOK, map[string]any{})
			return
		}
		http.Error(w, "failed to load active library", http.StatusInternalServerError)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, s.localLibraryPublicView(lib))
}

func (s *Server) handleActivateLocalLibrary(w http.ResponseWriter, r *http.Request) {
	if !s.localLibraryEnabled() {
		http.Error(w, "local libraries are disabled", http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	userID := UserIDFromContext(r.Context())
	if err := s.localLibraries.SetActiveForUser(userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !s.shouldKeepOtherSourceOnActivate(userID) {
		if err := s.instances.ClearActiveForUser(userID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	_ = s.reloadActiveSubsonic()

	lib, err := s.localLibraries.GetForUser(userID, id)
	if err != nil {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{"activated": true})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, s.localLibraryPublicView(lib))
}

func (s *Server) handleScanLocalLibrary(w http.ResponseWriter, r *http.Request) {
	if !s.localLibraryEnabled() {
		http.Error(w, "local libraries are disabled", http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	userID := UserIDFromContext(r.Context())
	if _, err := s.localLibraries.GetForUser(userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !s.startLibraryScan(id) {
		lib, getErr := s.localLibraries.Get(id)
		if getErr != nil {
			http.Error(w, "failed to load library", http.StatusInternalServerError)
			return
		}
		httputil.WriteJSON(w, http.StatusOK, s.localLibraryPublicView(lib))
		return
	}

	lib, _ := s.localLibraries.Get(id)
	httputil.WriteJSON(w, http.StatusAccepted, s.localLibraryPublicView(lib))
}

func (s *Server) handleUpdateLocalLibrary(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := UserIDFromContext(r.Context())
	existing, err := s.localLibraries.GetForUser(userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var req localLibraryRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	path := existing.Path
	if strings.TrimSpace(req.Path) != "" {
		resolved, err := s.resolveLibraryPath(req.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if resolved != existing.Path {
			if other, findErr := s.localLibraries.FindByPathForUser(userID, resolved); findErr == nil && other.ID != id {
				http.Error(w, "a library with this path already exists", http.StatusConflict)
				return
			} else if findErr != nil && !errors.Is(findErr, sql.ErrNoRows) {
				http.Error(w, "failed to validate library path", http.StatusInternalServerError)
				return
			}
		}
		path = resolved
	} else if !s.localLibraryConfig().AllowCustomPath {
		resolved, err := s.resolveLibraryPath("")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		path = resolved
	}

	lib, err := s.localLibraries.Update(id, store.UpdateLocalLibraryInput{
		Name: req.Name,
		Path: path,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if lib.Path != existing.Path {
		s.watchLocalLibrary(lib)
	}
	httputil.WriteJSON(w, http.StatusOK, s.localLibraryPublicView(lib))
}

func (s *Server) handleDeleteLocalLibrary(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := UserIDFromContext(r.Context())
	if _, err := s.localLibraries.GetForUser(userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.localLibraries.Delete(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.libraryWatcher != nil {
		s.libraryWatcher.UnwatchLibrary(id)
	}
	s.invalidateLocalLibraryData(id)

	activeID, _ := s.localLibraries.GetActiveIDForUser(userID)
	if activeID == id {
		_ = s.localLibraries.ClearActiveForUser(userID)
	}

	w.WriteHeader(http.StatusNoContent)
}
