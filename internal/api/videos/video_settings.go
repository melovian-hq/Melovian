// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package videos

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"melovian/internal/api/apishared"
	"melovian/internal/httputil"
	"melovian/internal/store"
	"melovian/internal/video"
)

const (
	VideoSearchProviderInvidious = "invidious"
	VideoSearchProviderYouTube   = "youtube"
)

type VideoSettings struct {
	Enabled          bool   `json:"enabled"`
	SearchProvider   string `json:"searchProvider"`
	InvidiousBaseURL string `json:"invidiousBaseUrl"`
	YouTubeAPIKey    string `json:"youtubeApiKey"`
}

func defaultVideoSettings() VideoSettings {
	return VideoSettings{
		Enabled:        false,
		SearchProvider: VideoSearchProviderInvidious,
	}
}

func normalizeVideoSearchProvider(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case VideoSearchProviderYouTube:
		return VideoSearchProviderYouTube
	default:
		return VideoSearchProviderInvidious
	}
}

func mergeVideoSettings(raw json.RawMessage) (VideoSettings, error) {
	settings := defaultVideoSettings()
	if len(raw) == 0 {
		return settings, nil
	}
	if err := json.Unmarshal(raw, &settings); err != nil {
		return VideoSettings{}, err
	}
	settings.SearchProvider = normalizeVideoSearchProvider(settings.SearchProvider)
	settings.InvidiousBaseURL = strings.TrimSpace(settings.InvidiousBaseURL)
	settings.YouTubeAPIKey = strings.TrimSpace(settings.YouTubeAPIKey)
	if settings.InvidiousBaseURL != "" {
		normalized, err := video.NormalizeInstanceURL(settings.InvidiousBaseURL)
		if err != nil {
			return VideoSettings{}, err
		}
		settings.InvidiousBaseURL = normalized
	}
	return settings, nil
}

func (h *Handler) loadVideoSettings(userID string) (VideoSettings, error) {
	if cached, ok := h.videoSettingsCache.Load(userID); ok {
		if settings, ok := cached.(VideoSettings); ok {
			return settings, nil
		}
	}
	raw, err := h.preferences.Get(userID, store.PrefKeyVideoSettings)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			settings := defaultVideoSettings()
			h.videoSettingsCache.Store(userID, settings)
			return settings, nil
		}
		return VideoSettings{}, err
	}
	settings, err := mergeVideoSettings(json.RawMessage(raw))
	if err != nil {
		return VideoSettings{}, err
	}
	h.videoSettingsCache.Store(userID, settings)
	return settings, nil
}

func (h *Handler) requireVideoEnabled(w http.ResponseWriter, r *http.Request) (VideoSettings, bool) {
	userID := apishared.ResolveProgressUserID(r.Context())
	settings, err := h.loadVideoSettings(userID)
	if err != nil {
		httputil.WriteInternalError(w, r, "requireVideoEnabled", err)
		return VideoSettings{}, false
	}
	if !settings.Enabled {
		httputil.WriteError(w, http.StatusForbidden, "videos_are_disabled_enable_them_in_setti", "videos are disabled. enable them in Settings → Video")
		return VideoSettings{}, false
	}
	return settings, true
}

func (h *Handler) handleGetVideoSettings(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	settings, err := h.loadVideoSettings(userID)
	if err != nil {
		httputil.WriteInternalError(w, r, "handleGetVideoSettings", err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, settings)
}

func (h *Handler) handlePutVideoSettings(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	var req VideoSettings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	settings := VideoSettings{
		Enabled:          req.Enabled,
		SearchProvider:   normalizeVideoSearchProvider(req.SearchProvider),
		InvidiousBaseURL: strings.TrimSpace(req.InvidiousBaseURL),
		YouTubeAPIKey:    strings.TrimSpace(req.YouTubeAPIKey),
	}
	if settings.InvidiousBaseURL != "" {
		normalized, err := video.NormalizeInstanceURL(settings.InvidiousBaseURL)
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		settings.InvidiousBaseURL = normalized
	}
	encoded, err := json.Marshal(settings)
	if err != nil {
		httputil.WriteInternalError(w, r, "handlePutVideoSettings", err)
		return
	}
	if err := h.preferences.Set(userID, store.PrefKeyVideoSettings, string(encoded)); err != nil {
		httputil.WriteInternalError(w, r, "handlePutVideoSettings", err)
		return
	}
	h.videoSettingsCache.Store(userID, settings)
	httputil.WriteJSON(w, http.StatusOK, settings)
}
