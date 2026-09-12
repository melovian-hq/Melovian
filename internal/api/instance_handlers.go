// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"melovian/internal/httputil"
	"melovian/internal/melog"
	"melovian/internal/store"
	"melovian/internal/subsonic"
)

func (s *Server) registerInstanceRoutes() {
	s.mux.HandleFunc("GET /api/instances", s.handleListInstances)
	s.mux.HandleFunc("POST /api/instances", s.handleCreateInstance)
	s.mux.HandleFunc("POST /api/instances/test", s.handleTestInstance)
	s.mux.HandleFunc("GET /api/instances/active", s.handleGetActiveInstance)
	s.mux.HandleFunc("POST /api/instances/{id}/activate", s.handleActivateInstance)
	s.mux.HandleFunc("GET /api/instances/{id}/ping", s.handlePingInstance)
	s.mux.HandleFunc("PUT /api/instances/{id}", s.handleUpdateInstance)
	s.mux.HandleFunc("DELETE /api/instances/{id}", s.handleDeleteInstance)
}

type instanceRequest struct {
	Name       string `json:"name"`
	ServerURL  string `json:"serverUrl"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	ServerName string `json:"serverName"`
}

func (s *Server) handleListInstances(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	items, err := s.instances.ListForUser(userID)
	if err != nil {
		http.Error(w, "failed to list instances", http.StatusInternalServerError)
		return
	}

	payload := make([]map[string]any, 0, len(items))
	for _, item := range items {
		payload = append(payload, s.instances.PublicView(item))
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"instances": payload})
}

func (s *Server) handleCreateInstance(w http.ResponseWriter, r *http.Request) {
	var req instanceRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client := subsonic.NewClient(req.ServerURL, req.Username, req.Password)
	serverName, _, err := client.Ping()
	if err != nil {
		http.Error(w, "connection test failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.ServerName == "" {
		req.ServerName = serverName
	}

	inst, err := s.instances.CreateForUser(UserIDFromContext(r.Context()), store.CreateInstanceInput{
		Name:       req.Name,
		ServerURL:  req.ServerURL,
		Username:   req.Username,
		Password:   req.Password,
		ServerName: req.ServerName,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	activeID, _ := s.instances.GetActiveIDForUser(UserIDFromContext(r.Context()))
	if activeID == "" {
		_ = s.instances.SetActiveForUser(UserIDFromContext(r.Context()), inst.ID)
		_ = s.reloadActiveSubsonic()
	}

	httputil.WriteJSON(w, http.StatusCreated, s.instances.PublicView(inst))
}

func (s *Server) handleTestInstance(w http.ResponseWriter, r *http.Request) {
	var req instanceRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteJSON(w, http.StatusBadRequest, map[string]any{
			"connected": false,
			"error":     err.Error(),
		})
		return
	}

	client := subsonic.NewClient(req.ServerURL, req.Username, req.Password)
	serverName, version, err := client.Ping()
	if err != nil {
		slog.Warn("instance connection test failed", "url", melog.Sanitize(req.ServerURL), "err", err)
		httputil.WriteJSON(w, http.StatusBadRequest, map[string]any{
			"connected": false,
			"error":     err.Error(),
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"connected":  true,
		"serverName": serverName,
		"version":    version,
	})
}

func (s *Server) handleGetActiveInstance(w http.ResponseWriter, r *http.Request) {
	inst, err := s.instances.GetActiveForUser(UserIDFromContext(r.Context()))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteJSON(w, http.StatusOK, map[string]any{})
			return
		}
		http.Error(w, "failed to load active instance", http.StatusInternalServerError)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, s.instances.PublicView(inst))
}

func (s *Server) handleActivateInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := UserIDFromContext(r.Context())
	if err := s.instances.SetActiveForUser(userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !s.shouldKeepOtherSourceOnActivate(userID) {
		if err := s.localLibraries.ClearActiveForUser(userID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := s.reloadActiveSubsonic(); err != nil {
		http.Error(w, "failed to reload subsonic client", http.StatusInternalServerError)
		return
	}
	_ = s.instances.TouchLastUsed(id)

	inst, err := s.instances.GetForUser(userID, id)
	if err != nil {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{"activated": true})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, s.instances.PublicView(inst))
}

func (s *Server) handlePingInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := UserIDFromContext(r.Context())
	inst, err := s.instances.GetForUser(userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := subsonic.NewClient(inst.ServerURL, inst.Username, inst.Password)
	start := time.Now()
	serverName, version, err := client.Ping()
	latencyMs := time.Since(start).Milliseconds()
	if err != nil {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{
			"online":    false,
			"latencyMs": latencyMs,
			"error":     err.Error(),
		})
		return
	}

	payload := map[string]any{
		"online":     true,
		"latencyMs":  latencyMs,
		"serverName": serverName,
		"version":    version,
	}
	if stats, statsErr := client.LibraryStats(); statsErr == nil {
		payload["songCount"] = stats.SongCount
	}
	httputil.WriteJSON(w, http.StatusOK, payload)
}

func (s *Server) handleUpdateInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := UserIDFromContext(r.Context())
	if _, err := s.instances.GetForUser(userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var req instanceRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	serverURL := strings.TrimSpace(req.ServerURL)
	username := strings.TrimSpace(req.Username)
	password := req.Password
	if serverURL != "" && username != "" && password != "" {
		client := subsonic.NewClient(serverURL, username, password)
		serverName, _, err := client.Ping()
		if err != nil {
			http.Error(w, "connection test failed: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.ServerName == "" {
			req.ServerName = serverName
		}
	}

	inst, err := s.instances.Update(id, store.UpdateInstanceInput{
		Name:       req.Name,
		ServerURL:  req.ServerURL,
		Username:   req.Username,
		Password:   req.Password,
		ServerName: req.ServerName,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	delete(s.clientCache, id)
	s.mu.Unlock()

	activeID, _ := s.instances.GetActiveIDForUser(userID)
	if activeID == id {
		_ = s.reloadActiveSubsonic()
	}

	httputil.WriteJSON(w, http.StatusOK, s.instances.PublicView(inst))
}

func (s *Server) handleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := UserIDFromContext(r.Context())
	if _, err := s.instances.GetForUser(userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	wasActive := false
	if activeID, err := s.instances.GetActiveIDForUser(userID); err == nil && activeID == id {
		wasActive = true
	}

	if err := s.instances.Delete(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	delete(s.clientCache, id)
	s.mu.Unlock()

	if wasActive {
		_ = s.instances.ClearActiveForUser(userID)
		_ = s.reloadActiveSubsonic()
	}

	w.WriteHeader(http.StatusNoContent)
}
