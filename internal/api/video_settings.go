// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

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

func (s *Server) loadVideoSettings(userID string) (VideoSettings, error) {
	if cached, ok := s.videoSettingsCache.Load(userID); ok {
		if settings, ok := cached.(VideoSettings); ok {
			return settings, nil
		}
	}
	raw, err := s.preferences.Get(userID, store.PrefKeyVideoSettings)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			settings := defaultVideoSettings()
			s.videoSettingsCache.Store(userID, settings)
			return settings, nil
		}
		return VideoSettings{}, err
	}
	settings, err := mergeVideoSettings(json.RawMessage(raw))
	if err != nil {
		return VideoSettings{}, err
	}
	s.videoSettingsCache.Store(userID, settings)
	return settings, nil
}

func (s *Server) requireVideoEnabled(w http.ResponseWriter, r *http.Request) (VideoSettings, bool) {
	userID := ResolveProgressUserID(r.Context())
	settings, err := s.loadVideoSettings(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return VideoSettings{}, false
	}
	if !settings.Enabled {
		http.Error(w, "videos are disabled. enable them in Settings → Video", http.StatusForbidden)
		return VideoSettings{}, false
	}
	return settings, true
}

func (s *Server) handleGetVideoSettings(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	settings, err := s.loadVideoSettings(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, settings)
}

func (s *Server) handlePutVideoSettings(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	var req VideoSettings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
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
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		settings.InvidiousBaseURL = normalized
	}
	encoded, err := json.Marshal(settings)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.preferences.Set(userID, store.PrefKeyVideoSettings, string(encoded)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.videoSettingsCache.Store(userID, settings)
	httputil.WriteJSON(w, http.StatusOK, settings)
}
