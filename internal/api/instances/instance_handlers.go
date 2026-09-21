// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package instances

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"melovian/internal/api/apishared"
	"melovian/internal/appconfig"
	"melovian/internal/httputil"
	"melovian/internal/melog"
	"melovian/internal/navidrome"
	"melovian/internal/sources"
	"melovian/internal/store"
	"melovian/internal/subsonic"
)

// Handler serves the subsonic instance and source-selection routes.
type Handler struct {
	instances      *store.InstanceStore
	localLibraries *store.LocalLibraryStore
	preferences    *store.PreferencesStore
	resolver       *apishared.Resolver
	sources        *sources.Registry
	sourceEvents   *sources.EventBridge
	cfg            appconfig.Config
}

// SetSourceEvents attaches the bridge that mirrors upstream source event
// streams onto the websocket hub. Optional; nil-safe.
func (h *Handler) SetSourceEvents(b *sources.EventBridge) {
	h.sourceEvents = b
}

func (h *Handler) syncSourceEvents(userID string) {
	if h.sourceEvents != nil {
		h.sourceEvents.SyncUser(userID)
	}
}

func New(instances *store.InstanceStore, localLibraries *store.LocalLibraryStore, preferences *store.PreferencesStore, resolver *apishared.Resolver, reg *sources.Registry, cfg appconfig.Config) *Handler {
	return &Handler{
		instances:      instances,
		localLibraries: localLibraries,
		preferences:    preferences,
		resolver:       resolver,
		sources:        reg,
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
	mux.HandleFunc("GET /api/instances/detect", h.handleDetectInstances)
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
	// SourceID selects the backing source extension. Empty means
	// auto-detect from the server ping, falling back to subsonic.
	SourceID string `json:"sourceId"`
}

// resolveSourceID picks the source extension for an instance. An explicit
// request value wins; otherwise the detected server identity decides.
func (h *Handler) resolveSourceID(requested, serverName, version string) (string, error) {
	if requested != "" {
		if h.sources != nil {
			if _, ok := h.sources.Get(requested); !ok {
				return "", fmt.Errorf("unknown source %q; valid values: %s", requested, h.sourceIDs())
			}
		}
		return requested, nil
	}
	if navidrome.IsNavidromeServer(serverName, version) {
		return sources.NavidromeSourceID, nil
	}
	return store.DefaultSourceID, nil
}

func (h *Handler) sourceIDs() string {
	ids := make([]string, 0)
	if h.sources == nil {
		return store.DefaultSourceID
	}
	for _, s := range h.sources.List() {
		ids = append(ids, s.ID())
	}
	return strings.Join(ids, ", ")
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
	serverName, version, err := client.Ping()
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "connection test failed: "+err.Error())
		return
	}
	if req.ServerName == "" {
		req.ServerName = serverName
	}
	sourceID, err := h.resolveSourceID(req.SourceID, serverName, version)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "unknown_source", err.Error())
		return
	}
	if h.sources != nil && !h.sources.Enabled(sourceID) {
		httputil.WriteError(w, http.StatusConflict, "source_disabled", "the "+sourceID+" extension is disabled")
		return
	}

	inst, err := h.instances.CreateForUser(apishared.UserIDFromContext(r.Context()), store.CreateInstanceInput{
		Name:       req.Name,
		ServerURL:  req.ServerURL,
		Username:   req.Username,
		Password:   req.Password,
		ServerName: req.ServerName,
		SourceID:   sourceID,
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

	sourceID, sourceErr := h.resolveSourceID(req.SourceID, serverName, version)
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"connected":   true,
		"serverName":  serverName,
		"version":     version,
		"sourceId":    sourceID,
		"sourceKnown": sourceErr == nil,
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
	if h.sources != nil {
		if inst, err := h.instances.GetForUser(userID, id); err == nil {
			if _, availErr := h.sources.Available(inst); availErr != nil {
				httputil.WriteError(w, http.StatusConflict, "source_unavailable", "the "+inst.SourceOrDefault()+" source is unavailable (extension disabled or unhealthy)")
				return
			}
		}
	}
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
	h.syncSourceEvents(userID)

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
	var serverName, version string
	if h.sources != nil {
		serverName, version, err = h.sources.PingInstance(r.Context(), inst)
	} else {
		serverName, version, err = client.Ping()
	}
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
		"sourceId":   inst.SourceOrDefault(),
	}
	if stats, statsErr := client.LibraryStats(); statsErr == nil {
		payload["songCount"] = stats.SongCount
	}
	httputil.WriteJSON(w, http.StatusOK, payload)
}

func (h *Handler) handleUpdateInstance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := apishared.UserIDFromContext(r.Context())
	existing, err := h.instances.GetForUser(userID, id)
	if err != nil {
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
	if serverURL != "" {
		// Repointing a stored instance must always re-verify the target.
		// The effective credentials come from the request or the stored
		// record, so a URL-only update cannot bypass the ping.
		if username == "" {
			username = existing.Username
		}
		if password == "" {
			password = existing.Password
		}
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

	if req.SourceID != "" && h.sources != nil {
		if _, ok := h.sources.Get(req.SourceID); !ok {
			httputil.WriteError(w, http.StatusBadRequest, "unknown_source", "unknown source "+req.SourceID)
			return
		}
	}
	inst, err := h.instances.Update(id, store.UpdateInstanceInput{
		Name:       req.Name,
		ServerURL:  req.ServerURL,
		Username:   req.Username,
		Password:   req.Password,
		ServerName: req.ServerName,
		SourceID:   req.SourceID,
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
		h.syncSourceEvents(userID)
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
	h.syncSourceEvents(userID)

	w.WriteHeader(http.StatusNoContent)
}
