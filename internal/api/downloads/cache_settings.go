// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package downloads

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"melovian/internal/api/apishared"
	"melovian/internal/httputil"
	"melovian/internal/store"
)

const defaultCacheLimitBytes = 5 * 1024 * 1024 * 1024

type CacheSettings struct {
	Enabled    bool   `json:"enabled"`
	LimitBytes int64  `json:"limitBytes"`
	Strategy   string `json:"strategy"`
}

func defaultCacheSettings() CacheSettings {
	return CacheSettings{
		Enabled:    true,
		LimitBytes: defaultCacheLimitBytes,
		Strategy:   "playback",
	}
}

func mergeCacheSettings(raw json.RawMessage) (CacheSettings, error) {
	settings := defaultCacheSettings()
	if len(raw) == 0 {
		return settings, nil
	}
	if err := json.Unmarshal(raw, &settings); err != nil {
		return CacheSettings{}, err
	}
	if settings.LimitBytes <= 0 {
		settings.LimitBytes = defaultCacheLimitBytes
	}
	switch settings.Strategy {
	case "playback", "most_listened", "playlists", "all":
	default:
		settings.Strategy = "playback"
	}
	return settings, nil
}

func (h *Handler) loadCacheSettings(userID string) (CacheSettings, error) {
	raw, err := h.preferences.Get(userID, store.PrefKeyCacheSettings)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return defaultCacheSettings(), nil
		}
		return CacheSettings{}, err
	}
	return mergeCacheSettings(json.RawMessage(raw))
}

func (h *Handler) handleGetCacheSettings(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	settings, err := h.loadCacheSettings(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load cache settings")
		return
	}
	totalSize, count, err := h.downloads.Stats(apishared.InstanceIDFromContext(r.Context()))
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load cache stats")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"enabled":    settings.Enabled,
		"limitBytes": settings.LimitBytes,
		"strategy":   settings.Strategy,
		"usedBytes":  totalSize,
		"trackCount": count,
	})
}

func (h *Handler) handlePutCacheSettings(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	body, err := httputil.ReadJSONBytes(r)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	if !json.Valid(body) {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	settings, err := mergeCacheSettings(body)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_cache_settings", "invalid cache settings")
		return
	}
	encoded, err := json.Marshal(settings)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "encode cache settings")
		return
	}
	if err := h.preferences.Set(userID, store.PrefKeyCacheSettings, string(encoded)); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	totalSize, count, err := h.downloads.Stats(apishared.InstanceIDFromContext(r.Context()))
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load cache stats")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"enabled":    settings.Enabled,
		"limitBytes": settings.LimitBytes,
		"strategy":   settings.Strategy,
		"usedBytes":  totalSize,
		"trackCount": count,
	})
}

func (h *Handler) handleClearDownloads(w http.ResponseWriter, r *http.Request) {
	instanceID := apishared.InstanceIDFromContext(r.Context())
	paths, err := h.downloads.ClearInstance(instanceID)
	if err != nil {
		httputil.WriteInternalError(w, r, "clear cache:", err)
		return
	}
	for _, path := range paths {
		_ = os.Remove(path)
	}
	w.WriteHeader(http.StatusNoContent)
}
