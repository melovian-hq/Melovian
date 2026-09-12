// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package library

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"melovian/internal/api/apishared"
	"melovian/internal/api/realtime"
	"melovian/internal/appconfig"
	"melovian/internal/httputil"
	"melovian/internal/metaloader"
	"melovian/internal/osutil"
	"melovian/internal/store"
)

func (h *Handler) registerLocalLibraryRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/local-libraries", h.handleListLocalLibraries)
	mux.HandleFunc("POST /api/local-libraries", h.handleCreateLocalLibrary)
	mux.HandleFunc("GET /api/local-libraries/active", h.handleGetActiveLocalLibrary)
	mux.HandleFunc("POST /api/local-libraries/{id}/activate", h.handleActivateLocalLibrary)
	mux.HandleFunc("POST /api/local-libraries/{id}/scan", h.handleScanLocalLibrary)
	mux.HandleFunc("GET /api/local-libraries/{id}", h.handleGetLocalLibrary)
	mux.HandleFunc("PUT /api/local-libraries/{id}", h.handleUpdateLocalLibrary)
	mux.HandleFunc("DELETE /api/local-libraries/{id}", h.handleDeleteLocalLibrary)
}

type localLibraryRequest struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (h *Handler) localLibraryPublicView(lib store.LocalLibrary) map[string]any {
	view := h.localLibraries.PublicView(lib)
	if lib.ScanStatus == "scanning" {
		if progress, ok := h.libraryScanner.Progress(lib.ID); ok {
			view["scanProgress"] = progress
		}
	}
	return view
}

func (h *Handler) startLibraryScan(libraryID string) bool {
	return h.libraryScanner.StartScan(libraryID, func(result metaloader.ScanResult, scanErr error) {
		h.invalidateLocalLibraryData(libraryID)
		if scanErr != nil {
			h.EmitScanError(libraryID, scanErr.Error())
			return
		}
		if len(result.Errors) > 0 && result.Total == 0 {
			h.EmitScanError(libraryID, result.Errors[0])
			return
		}
		h.EmitScanComplete(libraryID)
	})
}

func (h *Handler) watchLocalLibrary(lib store.LocalLibrary) {
	if h.libraryWatcher == nil {
		return
	}
	if err := h.libraryWatcher.WatchLibrary(lib.ID, lib.Path); err != nil {
		slog.Warn("local library watch failed",
			"library_id", lib.ID,
			"path", lib.Path,
			"err", err,
		)
	}
}

func (h *Handler) handleLibraryWatchUpdate(libraryID string) {
	h.invalidateLocalLibraryData(libraryID)
	lib, err := h.localLibraries.Get(libraryID)
	if err != nil {
		return
	}
	h.events.Broadcast(realtime.Event{
		Type: realtime.EventLibraryUpdated,
		Payload: realtime.LibraryUpdatedPayload{
			LibraryID:      lib.ID,
			Name:           lib.Name,
			TrackCount:     lib.TrackCount,
			MissingCount:   lib.MissingCount,
			DuplicateCount: lib.DuplicateCount,
		},
	})
}

func (h *Handler) localLibraryConfig() appconfig.LocalLibraryConfig {
	return h.cfg.LocalLibraryEffective()
}

func (h *Handler) Enabled() bool {
	return h.localLibraryConfig().Enabled
}

func (h *Handler) ResolveLibraryPath(inputPath string) (string, error) {
	cfg := h.localLibraryConfig()
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
	// Resolve before validation so a symlink cannot hide the real location.
	resolved, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	// On a multi-user server, a user supplied path must stay under the same
	// roots the folder browser offers. Otherwise any account could point a
	// library at arbitrary directories and stream whatever the process can
	// read. Single user installs are trusted with any path.
	if h.cfg.ServerMode && h.cfg.AuthEnabled() {
		if _, err := osutil.ResolveInside(resolved, h.browseStartCandidates()...); err != nil {
			return "", errors.New("path must be under an allowed root (home, /media, /mnt, /run/media, or MELOVIAN_LOCAL_LIBRARY_ROOTS)")
		}
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("path is not a directory")
	}
	return resolved, nil
}

func (h *Handler) handleListLocalLibraries(w http.ResponseWriter, r *http.Request) {
	if !h.Enabled() {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{"libraries": []any{}})
		return
	}

	userID := apishared.UserIDFromContext(r.Context())
	items, err := h.localLibraries.ListForUser(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list libraries")
		return
	}

	payload := make([]map[string]any, 0, len(items))
	for _, item := range items {
		payload = append(payload, h.localLibraryPublicView(item))
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"libraries": payload})
}

func (h *Handler) handleCreateLocalLibrary(w http.ResponseWriter, r *http.Request) {
	if !h.Enabled() {
		httputil.WriteError(w, http.StatusForbidden, "local_libraries_are_disabled", "local libraries are disabled")
		return
	}

	var req localLibraryRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	path, err := h.ResolveLibraryPath(req.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	userID := apishared.UserIDFromContext(r.Context())
	if _, err := h.localLibraries.FindByPathForUser(userID, path); err == nil {
		httputil.WriteError(w, http.StatusConflict, "a_library_with_this_path_already_exists", "a library with this path already exists")
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to validate library path")
		return
	}

	lib, err := h.localLibraries.CreateForUser(userID, store.CreateLocalLibraryInput{
		Name: req.Name,
		Path: path,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	h.watchLocalLibrary(lib)
	_ = h.localLibraries.SetScanState(lib.ID, "scanning", "")
	if !h.startLibraryScan(lib.ID) {
		_ = h.localLibraries.SetScanState(lib.ID, "error", "failed to start scan")
	}

	lib, _ = h.localLibraries.Get(lib.ID)
	httputil.WriteJSON(w, http.StatusCreated, h.localLibraryPublicView(lib))
}

func (h *Handler) handleGetLocalLibrary(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := apishared.UserIDFromContext(r.Context())
	lib, err := h.localLibraries.GetForUser(userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load library")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, h.localLibraryPublicView(lib))
}

func (h *Handler) handleGetActiveLocalLibrary(w http.ResponseWriter, r *http.Request) {
	if !h.Enabled() {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{})
		return
	}

	userID := apishared.UserIDFromContext(r.Context())
	lib, err := h.localLibraries.GetActiveForUser(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteJSON(w, http.StatusOK, map[string]any{})
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load active library")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, h.localLibraryPublicView(lib))
}

func (h *Handler) handleActivateLocalLibrary(w http.ResponseWriter, r *http.Request) {
	if !h.Enabled() {
		httputil.WriteError(w, http.StatusForbidden, "local_libraries_are_disabled", "local libraries are disabled")
		return
	}

	id := r.PathValue("id")
	userID := apishared.UserIDFromContext(r.Context())
	if err := h.localLibraries.SetActiveForUser(userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleActivateLocalLibrary", err)
		return
	}
	if !apishared.ShouldKeepOtherSourceOnActivate(h.preferences, userID) {
		if err := h.instances.ClearActiveForUser(userID); err != nil {
			httputil.WriteInternalError(w, r, "handleActivateLocalLibrary", err)
			return
		}
	}
	_ = h.resolver.ReloadActive()

	lib, err := h.localLibraries.GetForUser(userID, id)
	if err != nil {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{"activated": true})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, h.localLibraryPublicView(lib))
}

func (h *Handler) handleScanLocalLibrary(w http.ResponseWriter, r *http.Request) {
	if !h.Enabled() {
		httputil.WriteError(w, http.StatusForbidden, "local_libraries_are_disabled", "local libraries are disabled")
		return
	}

	id := r.PathValue("id")
	userID := apishared.UserIDFromContext(r.Context())
	if _, err := h.localLibraries.GetForUser(userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleScanLocalLibrary", err)
		return
	}

	if !h.startLibraryScan(id) {
		lib, getErr := h.localLibraries.Get(id)
		if getErr != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load library")
			return
		}
		httputil.WriteJSON(w, http.StatusOK, h.localLibraryPublicView(lib))
		return
	}

	lib, _ := h.localLibraries.Get(id)
	httputil.WriteJSON(w, http.StatusAccepted, h.localLibraryPublicView(lib))
}

func (h *Handler) handleUpdateLocalLibrary(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := apishared.UserIDFromContext(r.Context())
	existing, err := h.localLibraries.GetForUser(userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleUpdateLocalLibrary", err)
		return
	}

	var req localLibraryRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	path := existing.Path
	if strings.TrimSpace(req.Path) != "" {
		resolved, err := h.ResolveLibraryPath(req.Path)
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		if resolved != existing.Path {
			if other, findErr := h.localLibraries.FindByPathForUser(userID, resolved); findErr == nil && other.ID != id {
				httputil.WriteError(w, http.StatusConflict, "a_library_with_this_path_already_exists", "a library with this path already exists")
				return
			} else if findErr != nil && !errors.Is(findErr, sql.ErrNoRows) {
				httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to validate library path")
				return
			}
		}
		path = resolved
	} else if !h.localLibraryConfig().AllowCustomPath {
		resolved, err := h.ResolveLibraryPath("")
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		path = resolved
	}

	lib, err := h.localLibraries.Update(id, store.UpdateLocalLibraryInput{
		Name: req.Name,
		Path: path,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if lib.Path != existing.Path {
		h.watchLocalLibrary(lib)
	}
	httputil.WriteJSON(w, http.StatusOK, h.localLibraryPublicView(lib))
}

func (h *Handler) handleDeleteLocalLibrary(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := apishared.UserIDFromContext(r.Context())
	if _, err := h.localLibraries.GetForUser(userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleDeleteLocalLibrary", err)
		return
	}
	if err := h.localLibraries.Delete(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleDeleteLocalLibrary", err)
		return
	}
	if h.libraryWatcher != nil {
		h.libraryWatcher.UnwatchLibrary(id)
	}
	h.invalidateLocalLibraryData(id)

	activeID, _ := h.localLibraries.GetActiveIDForUser(userID)
	if activeID == id {
		_ = h.localLibraries.ClearActiveForUser(userID)
	}

	w.WriteHeader(http.StatusNoContent)
}
