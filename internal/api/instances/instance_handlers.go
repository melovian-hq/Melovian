// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package instances

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"melovian/internal/api/apishared"
	"melovian/internal/appconfig"
	"melovian/internal/httputil"
	"melovian/internal/melog"
	"melovian/internal/store"
	"melovian/internal/subsonic"
)

// Handler serves the subsonic instance and source-selection routes.
type Handler struct {
	instances      *store.InstanceStore
	localLibraries *store.LocalLibraryStore
	preferences    *store.PreferencesStore
	resolver       *apishared.Resolver
	cfg            appconfig.Config
}

func New(instances *store.InstanceStore, localLibraries *store.LocalLibraryStore, preferences *store.PreferencesStore, resolver *apishared.Resolver, cfg appconfig.Config) *Handler {
	return &Handler{
		instances:      instances,
		localLibraries: localLibraries,
		preferences:    preferences,
		resolver:       resolver,
		cfg:            cfg,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	h.registerInstanceRoutes(mux)
	h.registerSourceRoutes(mux)
}

func (h *Handler) localLibraryEnabled() bool {
	return h.cfg.LocalLibraryEffective().Enabled
}

func (h *Handler) registerInstanceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/instances", h.handleListInstances)
	mux.HandleFunc("POST /api/instances", h.handleCreateInstance)
	mux.HandleFunc("POST /api/instances/test", h.handleTestInstance)
	mux.HandleFunc("GET /api/instances/active", h.handleGetActiveInstance)
	mux.HandleFunc("POST /api/instances/{id}/activate", h.handleActivateInstance)
	mux.HandleFunc("GET /api/instances/{id}/ping", h.handlePingInstance)
	mux.HandleFunc("PUT /api/instances/{id}", h.handleUpdateInstance)
	mux.HandleFunc("DELETE /api/instances/{id}", h.handleDeleteInstance)
}

type instanceRequest struct {
	Name       string `json:"name"`
	ServerURL  string `json:"serverUrl"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	ServerName string `json:"serverName"`
}

// validateInstanceServerURL rejects non HTTP(S) upstreams and URLs that embed
// basic auth credentials, which would otherwise be stored and logged.
func validateInstanceServerURL(raw string) error {
	u, err := httputil.ParseHTTPURL(raw)
	switch {
	case errors.Is(err, httputil.ErrURLScheme):
		return errors.New("server url must use http or https")
	case err != nil:
		return errors.New("server url is not valid")
	}
	if u.User != nil {
		return errors.New("credentials in the server url are not allowed, use the username and password fields")
	}
	return nil
}

func (h *Handler) handleListInstances(w http.ResponseWriter, r *http.Request) {
	userID := apishared.UserIDFromContext(r.Context())
	items, err := h.instances.ListForUser(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list instances")
		return
	}

	payload := make([]map[string]any, 0, len(items))
	for _, item := range items {
		payload = append(payload, h.instances.PublicView(item))
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"instances": payload})
}

func (h *Handler) handleCreateInstance(w http.ResponseWriter, r *http.Request) {
	var req instanceRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := validateInstanceServerURL(req.ServerURL); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	client := subsonic.NewClient(req.ServerURL, req.Username, req.Password)
	serverName, _, err := client.Ping()
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "connection test failed: "+err.Error())
		return
	}
	if req.ServerName == "" {
		req.ServerName = serverName
	}

	inst, err := h.instances.CreateForUser(apishared.UserIDFromContext(r.Context()), store.CreateInstanceInput{
		Name:       req.Name,
		ServerURL:  req.ServerURL,
		Username:   req.Username,
		Password:   req.Password,
		ServerName: req.ServerName,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	activeID, _ := h.instances.GetActiveIDForUser(apishared.UserIDFromContext(r.Context()))
	if activeID == "" {
		_ = h.instances.SetActiveForUser(apishared.UserIDFromContext(r.Context()), inst.ID)
		_ = h.resolver.ReloadActive()
	}

	httputil.WriteJSON(w, http.StatusCreated, h.instances.PublicView(inst))
}

func (h *Handler) handleTestInstance(w http.ResponseWriter, r *http.Request) {
	var req instanceRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteJSON(w, http.StatusBadRequest, map[string]any{
			"connected": false,
			"error":     err.Error(),
		})
		return
	}
	if err := validateInstanceServerURL(req.ServerURL); err != nil {
		httputil.WriteJSON(w, http.StatusBadRequest, map[string]any{
			"connected": false,
			"error":     err.Error(),
		})
		return
	}

	client := subsonic.NewClient(req.ServerURL, req.Username, req.Password)
	serverName, version, err := client.Ping()
	if err != nil {
		slog.Warn("instance connection test failed", "url", melog.Sanitize(httputil.RedactURLUserinfo(req.ServerURL)), "err", err)
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

func (h *Handler) handleGetActiveInstance(w http.ResponseWriter, r *http.Request) {
	inst, err := h.instances.GetActiveForUser(apishared.UserIDFromContext(r.Context()))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteJSON(w, http.StatusOK, map[string]any{})
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load active instance")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, h.instances.PublicView(inst))
}

func (h *Handler) handleActivateInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := apishared.UserIDFromContext(r.Context())
	if err := h.instances.SetActiveForUser(userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleActivateInstance", err)
		return
	}
	if !apishared.ShouldKeepOtherSourceOnActivate(h.preferences, userID) {
		if err := h.localLibraries.ClearActiveForUser(userID); err != nil {
			httputil.WriteInternalError(w, r, "handleActivateInstance", err)
			return
		}
	}
	if err := h.resolver.ReloadActive(); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to reload subsonic client")
		return
	}
	_ = h.instances.TouchLastUsed(id)

	inst, err := h.instances.GetForUser(userID, id)
	if err != nil {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{"activated": true})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, h.instances.PublicView(inst))
}

func (h *Handler) handlePingInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := apishared.UserIDFromContext(r.Context())
	inst, err := h.instances.GetForUser(userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handlePingInstance", err)
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

func (h *Handler) handleUpdateInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := apishared.UserIDFromContext(r.Context())
	if _, err := h.instances.GetForUser(userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleUpdateInstance", err)
		return
	}

	var req instanceRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	serverURL := strings.TrimSpace(req.ServerURL)
	if serverURL != "" {
		if err := validateInstanceServerURL(serverURL); err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
	}
	username := strings.TrimSpace(req.Username)
	password := req.Password
	if serverURL != "" && username != "" && password != "" {
		client := subsonic.NewClient(serverURL, username, password)
		serverName, _, err := client.Ping()
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "bad_request", "connection test failed: "+err.Error())
			return
		}
		if req.ServerName == "" {
			req.ServerName = serverName
		}
	}

	inst, err := h.instances.Update(id, store.UpdateInstanceInput{
		Name:       req.Name,
		ServerURL:  req.ServerURL,
		Username:   req.Username,
		Password:   req.Password,
		ServerName: req.ServerName,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	h.resolver.Invalidate(id)

	activeID, _ := h.instances.GetActiveIDForUser(userID)
	if activeID == id {
		_ = h.resolver.ReloadActive()
	}

	httputil.WriteJSON(w, http.StatusOK, h.instances.PublicView(inst))
}

func (h *Handler) handleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := apishared.UserIDFromContext(r.Context())
	if _, err := h.instances.GetForUser(userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleDeleteInstance", err)
		return
	}

	wasActive := false
	if activeID, err := h.instances.GetActiveIDForUser(userID); err == nil && activeID == id {
		wasActive = true
	}

	if err := h.instances.Delete(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleDeleteInstance", err)
		return
	}

	h.resolver.Invalidate(id)

	if wasActive {
		_ = h.instances.ClearActiveForUser(userID)
		_ = h.resolver.ReloadActive()
	}

	w.WriteHeader(http.StatusNoContent)
}
