// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"

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

func (s *Server) loadCacheSettings(userID string) (CacheSettings, error) {
	raw, err := s.preferences.Get(userID, store.PrefKeyCacheSettings)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return defaultCacheSettings(), nil
		}
		return CacheSettings{}, err
	}
	return mergeCacheSettings(json.RawMessage(raw))
}

func (s *Server) handleGetCacheSettings(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	settings, err := s.loadCacheSettings(userID)
	if err != nil {
		http.Error(w, "failed to load cache settings", http.StatusInternalServerError)
		return
	}
	totalSize, count, err := s.downloads.Stats(InstanceIDFromContext(r.Context()))
	if err != nil {
		http.Error(w, "failed to load cache stats", http.StatusInternalServerError)
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

func (s *Server) handlePutCacheSettings(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	body, err := httputil.ReadJSONBytes(r)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if !json.Valid(body) {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	settings, err := mergeCacheSettings(body)
	if err != nil {
		http.Error(w, "invalid cache settings", http.StatusBadRequest)
		return
	}
	encoded, err := json.Marshal(settings)
	if err != nil {
		http.Error(w, "encode cache settings", http.StatusInternalServerError)
		return
	}
	if err := s.preferences.Set(userID, store.PrefKeyCacheSettings, string(encoded)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	totalSize, count, err := s.downloads.Stats(InstanceIDFromContext(r.Context()))
	if err != nil {
		http.Error(w, "failed to load cache stats", http.StatusInternalServerError)
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

func (s *Server) handleClearDownloads(w http.ResponseWriter, r *http.Request) {
	instanceID := InstanceIDFromContext(r.Context())
	paths, err := s.downloads.ClearInstance(instanceID)
	if err != nil {
		http.Error(w, "clear cache: "+err.Error(), http.StatusInternalServerError)
		return
	}
	for _, path := range paths {
		_ = os.Remove(path)
	}
	w.WriteHeader(http.StatusNoContent)
}
