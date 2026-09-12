// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package instances

import (
	"errors"
	"net/http"

	"melovian/internal/api/apishared"
	"melovian/internal/httputil"
	"melovian/internal/store"
)

func (h *Handler) registerSourceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/sources/status", h.handleSourceStatus)
	mux.HandleFunc("PUT /api/sources/view-mode", h.handleSetSourceViewMode)
	mux.HandleFunc("PUT /api/sources/multi-local-library", h.handleSetMultiLocalLibrary)
}

type multiLocalLibraryRequest struct {
	Enabled bool `json:"enabled"`
}

type sourceViewModeRequest struct {
	Mode string `json:"mode"`
}

func (h *Handler) handleSourceStatus(w http.ResponseWriter, r *http.Request) {
	userID := apishared.UserIDFromContext(r.Context())
	mode := apishared.SourceViewModeForUser(h.preferences, userID)

	subsonicID, _ := h.instances.GetActiveIDForUser(userID)
	localID, _ := h.localLibraries.GetActiveIDForUser(userID)
	multiLocal, _ := h.preferences.GetMultiLocalLibrary(userID)

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"mode":              mode,
		"activeInstanceId":  subsonicID,
		"activeLocalId":     localID,
		"multiLocalLibrary": multiLocal,
		"unifiedAvailable":  subsonicID != "" && localID != "" && h.localLibraryEnabled(),
	})
}

func (h *Handler) handleSetMultiLocalLibrary(w http.ResponseWriter, r *http.Request) {
	if !h.localLibraryEnabled() {
		httputil.WriteError(w, http.StatusBadRequest, "local_disabled", "local libraries are disabled")
		return
	}
	var req multiLocalLibraryRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	userID := apishared.UserIDFromContext(r.Context())
	if err := h.preferences.SetMultiLocalLibrary(userID, req.Enabled); err != nil {
		httputil.WriteInternalError(w, r, "set multi local library", err)
		return
	}
	h.handleSourceStatus(w, r)
}

func (h *Handler) handleSetSourceViewMode(w http.ResponseWriter, r *http.Request) {
	var req sourceViewModeRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	userID := apishared.UserIDFromContext(r.Context())
	if err := h.preferences.SetSourceViewMode(userID, req.Mode); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_mode", err.Error())
		return
	}

	if req.Mode == store.SourceViewUnified {
		if err := h.ensureUnifiedSources(userID); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "unified_unavailable", err.Error())
			return
		}
	}

	h.handleSourceStatus(w, r)
}

func (h *Handler) ensureUnifiedSources(userID string) error {
	if !h.localLibraryEnabled() {
		return errors.New("local libraries are disabled")
	}
	instanceID, _ := h.instances.GetActiveIDForUser(userID)
	if instanceID == "" {
		items, err := h.instances.ListForUser(userID)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return errors.New("no subsonic instance configured")
		}
		if err := h.instances.SetActiveForUser(userID, items[0].ID); err != nil {
			return err
		}
		_ = h.resolver.ReloadActive()
	}
	localID, _ := h.localLibraries.GetActiveIDForUser(userID)
	if localID == "" {
		items, err := h.localLibraries.ListForUser(userID)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return errors.New("no local library configured")
		}
		if err := h.localLibraries.SetActiveForUser(userID, items[0].ID); err != nil {
			return err
		}
	}
	return nil
}
