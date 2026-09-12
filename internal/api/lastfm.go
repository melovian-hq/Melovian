// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"melovian/internal/extensions"
	"melovian/internal/httputil"
	"melovian/internal/lastfm"
	"melovian/internal/store"
)

type lastFMSettings struct {
	Enabled    bool   `json:"enabled"`
	HasToken   bool   `json:"hasToken"`
	APIKey     string `json:"apiKey"`
	APISecret  string `json:"apiSecret"`
	SessionKey string `json:"sessionKey"`
	Endpoint   string `json:"endpoint"`
}

type lastFMPayload struct {
	APIKey     string `json:"apiKey"`
	APISecret  string `json:"apiSecret"`
	SessionKey string `json:"sessionKey"`
	Endpoint   string `json:"endpoint"`
}

func (m *MusicService) lastFMEnabled() bool {
	return extensions.IsEnabled(m.dataDir, "lastfm")
}

func (m *MusicService) loadLastFMSettings(userID string) (lastFMPayload, error) {
	raw, err := m.preferences.Get(userID, store.PrefKeyLastFMSettings)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return lastFMPayload{}, nil
		}
		return lastFMPayload{}, err
	}
	var payload lastFMPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return lastFMPayload{}, err
	}
	payload.APIKey = strings.TrimSpace(payload.APIKey)
	payload.APISecret = strings.TrimSpace(payload.APISecret)
	payload.SessionKey = strings.TrimSpace(payload.SessionKey)
	payload.Endpoint = strings.TrimSpace(payload.Endpoint)
	return payload, nil
}

func (m *MusicService) saveLastFMSettings(userID string, payload lastFMPayload) error {
	if payload.APIKey == "" && payload.APISecret == "" && payload.SessionKey == "" && payload.Endpoint == "" {
		return m.preferences.Delete(userID, store.PrefKeyLastFMSettings)
	}
	data, err := json.Marshal(payload) //#nosec G117 -- settings persisted to the preferences store, not a response leak
	if err != nil {
		return err
	}
	return m.preferences.Set(userID, store.PrefKeyLastFMSettings, string(data))
}

func defaultLastFMEndpoint(endpoint string) string {
	if strings.TrimSpace(endpoint) == "" {
		return "https://ws.audioscrobbler.com/2.0/"
	}
	ep := strings.TrimSpace(endpoint)
	if !strings.HasSuffix(ep, "/") {
		ep += "/"
	}
	return ep
}

func maskValue(value string) string {
	if len(value) <= 8 {
		return ""
	}
	return "..." + value[len(value)-4:]
}

func (m *MusicService) handleGetLastFMSettings(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	payload, err := m.loadLastFMSettings(userID)
	if err != nil {
		http.Error(w, "failed to load last.fm settings", http.StatusInternalServerError)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, lastFMSettings{
		Enabled:    m.lastFMEnabled(),
		HasToken:   payload.APIKey != "" && payload.APISecret != "" && payload.SessionKey != "",
		APIKey:     payload.APIKey,
		APISecret:  maskValue(payload.APISecret),
		SessionKey: maskValue(payload.SessionKey),
		Endpoint:   defaultLastFMEndpoint(payload.Endpoint),
	})
}

func (m *MusicService) handlePutLastFMSettings(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	var payload lastFMPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	payload.APIKey = strings.TrimSpace(payload.APIKey)
	payload.APISecret = strings.TrimSpace(payload.APISecret)
	payload.SessionKey = strings.TrimSpace(payload.SessionKey)
	payload.Endpoint = strings.TrimSpace(payload.Endpoint)
	if err := m.saveLastFMSettings(userID, payload); err != nil {
		http.Error(w, "failed to save last.fm settings", http.StatusInternalServerError)
		return
	}
	m.handleGetLastFMSettings(w, r)
}

func (m *MusicService) handleTestLastFMToken(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	var body lastFMPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	saved, err := m.loadLastFMSettings(userID)
	if err != nil {
		http.Error(w, "failed to load last.fm settings", http.StatusInternalServerError)
		return
	}
	apiKey := firstNonEmpty(strings.TrimSpace(body.APIKey), saved.APIKey)
	apiSecret := firstNonEmpty(strings.TrimSpace(body.APISecret), saved.APISecret)
	sessionKey := firstNonEmpty(strings.TrimSpace(body.SessionKey), saved.SessionKey)
	endpoint := defaultLastFMEndpoint(firstNonEmpty(strings.TrimSpace(body.Endpoint), saved.Endpoint))
	if apiKey == "" || apiSecret == "" || sessionKey == "" {
		http.Error(w, "last.fm credentials are not configured", http.StatusBadRequest)
		return
	}
	client := lastfm.NewClient(m.httpClient)
	client.BaseURL = endpoint
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	userName, err := client.ValidateToken(ctx, apiKey, apiSecret, sessionKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("last.fm token test failed: %s", err.Error()), http.StatusBadRequest)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"userName": userName,
	})
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

func (m *MusicService) doLastFMScrobble(w http.ResponseWriter, r *http.Request, listenType string) {
	if !m.lastFMEnabled() {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	userID := ResolveProgressUserID(r.Context())
	payload, err := m.loadLastFMSettings(userID)
	if err != nil {
		http.Error(w, "failed to load last.fm settings", http.StatusInternalServerError)
		return
	}
	if payload.APIKey == "" || payload.APISecret == "" || payload.SessionKey == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var req scrobblerTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	track := lastfm.Track{
		ID:              req.ID,
		Title:           req.Title,
		Artist:          req.Artist,
		Album:           req.Album,
		DurationSeconds: req.Duration,
	}
	client := lastfm.NewClient(m.httpClient)
	client.BaseURL = defaultLastFMEndpoint(payload.Endpoint)
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if listenType == "playing_now" {
		err = client.UpdateNowPlaying(ctx, payload.APIKey, payload.APISecret, payload.SessionKey, track)
	} else {
		err = client.Scrobble(ctx, payload.APIKey, payload.APISecret, payload.SessionKey, track, time.Now())
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *MusicService) handleLastFMNowPlaying(w http.ResponseWriter, r *http.Request) {
	m.doLastFMScrobble(w, r, "playing_now")
}

func (m *MusicService) handleLastFMScrobble(w http.ResponseWriter, r *http.Request) {
	m.doLastFMScrobble(w, r, "single")
}
