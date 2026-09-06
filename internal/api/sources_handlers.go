// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"errors"
	"net/http"

	"melovian/internal/httputil"
	"melovian/internal/store"
)

func (s *Server) registerSourceRoutes() {
	s.mux.HandleFunc("GET /api/sources/status", s.handleSourceStatus)
	s.mux.HandleFunc("PUT /api/sources/view-mode", s.handleSetSourceViewMode)
	s.mux.HandleFunc("PUT /api/sources/multi-local-library", s.handleSetMultiLocalLibrary)
}

type multiLocalLibraryRequest struct {
	Enabled bool `json:"enabled"`
}

type sourceViewModeRequest struct {
	Mode string `json:"mode"`
}

func (s *Server) sourceViewModeForUser(userID string) string {
	mode, err := s.preferences.GetSourceViewMode(userID)
	if err != nil {
		return store.SourceViewSubsonic
	}
	return mode
}

func (s *Server) handleSourceStatus(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	mode := s.sourceViewModeForUser(userID)

	subsonicID, _ := s.instances.GetActiveIDForUser(userID)
	localID, _ := s.localLibraries.GetActiveIDForUser(userID)
	multiLocal, _ := s.preferences.GetMultiLocalLibrary(userID)

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"mode":              mode,
		"activeInstanceId":  subsonicID,
		"activeLocalId":     localID,
		"multiLocalLibrary": multiLocal,
		"unifiedAvailable":  subsonicID != "" && localID != "" && s.localLibraryEnabled(),
	})
}

func (s *Server) handleSetMultiLocalLibrary(w http.ResponseWriter, r *http.Request) {
	if !s.localLibraryEnabled() {
		httputil.WriteError(w, http.StatusBadRequest, "local_disabled", "local libraries are disabled")
		return
	}
	var req multiLocalLibraryRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	userID := UserIDFromContext(r.Context())
	if err := s.preferences.SetMultiLocalLibrary(userID, req.Enabled); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "save_failed", err.Error())
		return
	}
	s.handleSourceStatus(w, r)
}

func (s *Server) handleSetSourceViewMode(w http.ResponseWriter, r *http.Request) {
	var req sourceViewModeRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	userID := UserIDFromContext(r.Context())
	if err := s.preferences.SetSourceViewMode(userID, req.Mode); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_mode", err.Error())
		return
	}

	if req.Mode == store.SourceViewUnified {
		if err := s.ensureUnifiedSources(userID); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "unified_unavailable", err.Error())
			return
		}
	}

	s.handleSourceStatus(w, r)
}

func (s *Server) ensureUnifiedSources(userID string) error {
	if !s.localLibraryEnabled() {
		return errors.New("local libraries are disabled")
	}
	instanceID, _ := s.instances.GetActiveIDForUser(userID)
	if instanceID == "" {
		items, err := s.instances.ListForUser(userID)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return errors.New("no subsonic instance configured")
		}
		if err := s.instances.SetActiveForUser(userID, items[0].ID); err != nil {
			return err
		}
		_ = s.reloadActiveSubsonic()
	}
	localID, _ := s.localLibraries.GetActiveIDForUser(userID)
	if localID == "" {
		items, err := s.localLibraries.ListForUser(userID)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return errors.New("no local library configured")
		}
		if err := s.localLibraries.SetActiveForUser(userID, items[0].ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) shouldKeepOtherSourceOnActivate(userID string) bool {
	return store.IsUnifiedSourceView(s.sourceViewModeForUser(userID))
}
