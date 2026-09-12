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
	"melovian/internal/rocksky"
	"melovian/internal/store"
)

type listenBrainzSettings struct {
	Enabled  bool   `json:"enabled"`
	HasToken bool   `json:"hasToken"`
	Token    string `json:"token"`
	Endpoint string `json:"endpoint"`
}

type listenBrainzPayload struct {
	Token    string `json:"token"`
	Endpoint string `json:"endpoint"`
}

func (m *MusicService) listenBrainzEnabled() bool {
	return extensions.IsEnabled(m.dataDir, "listenbrainz")
}

func (m *MusicService) loadListenBrainzSettings(userID string) (listenBrainzPayload, error) {
	raw, err := m.preferences.Get(userID, store.PrefKeyListenBrainzSettings)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return listenBrainzPayload{}, nil
		}
		return listenBrainzPayload{}, err
	}
	var payload listenBrainzPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return listenBrainzPayload{}, err
	}
	payload.Token = strings.TrimSpace(payload.Token)
	payload.Endpoint = strings.TrimSpace(payload.Endpoint)
	return payload, nil
}

func (m *MusicService) saveListenBrainzSettings(userID string, payload listenBrainzPayload) error {
	if payload.Token == "" && payload.Endpoint == "" {
		return m.preferences.Delete(userID, store.PrefKeyListenBrainzSettings)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return m.preferences.Set(userID, store.PrefKeyListenBrainzSettings, string(data))
}

func defaultListenBrainzEndpoint(endpoint string) string {
	if strings.TrimSpace(endpoint) == "" {
		return "https://api.listenbrainz.org"
	}
	return strings.TrimRight(strings.TrimSpace(endpoint), "/")
}

func (m *MusicService) handleGetListenBrainzSettings(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	payload, err := m.loadListenBrainzSettings(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load listenbrainz settings")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, listenBrainzSettings{
		Enabled:  m.listenBrainzEnabled(),
		HasToken: payload.Token != "",
		Token:    maskToken(payload.Token),
		Endpoint: defaultListenBrainzEndpoint(payload.Endpoint),
	})
}

func (m *MusicService) handlePutListenBrainzSettings(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	var payload listenBrainzPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	payload.Token = strings.TrimSpace(payload.Token)
	payload.Endpoint = strings.TrimSpace(payload.Endpoint)
	if err := m.saveListenBrainzSettings(userID, payload); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to save listenbrainz settings")
		return
	}
	m.handleGetListenBrainzSettings(w, r)
}

func (m *MusicService) handleTestListenBrainzToken(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	var body listenBrainzPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	saved, err := m.loadListenBrainzSettings(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load listenbrainz settings")
		return
	}
	token := strings.TrimSpace(body.Token)
	endpoint := defaultListenBrainzEndpoint(body.Endpoint)
	if token == "" {
		token = saved.Token
		endpoint = defaultListenBrainzEndpoint(saved.Endpoint)
	}
	if token == "" {
		httputil.WriteError(w, http.StatusBadRequest, "listenbrainz_token_is_not_configured", "listenbrainz token is not configured")
		return
	}
	client := rocksky.NewClient(m.httpClient)
	client.BaseURL = endpoint
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	userName, err := client.ValidateToken(ctx, token)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "token_test_failed", fmt.Sprintf("listenbrainz token test failed: %s", err.Error()))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"userName": userName,
	})
}

func (m *MusicService) doListenBrainzScrobble(w http.ResponseWriter, r *http.Request, listenType string) {
	if !m.listenBrainzEnabled() {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	userID := ResolveProgressUserID(r.Context())
	payload, err := m.loadListenBrainzSettings(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load listenbrainz settings")
		return
	}
	if payload.Token == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var req scrobblerTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	track := rocksky.Track{
		ID:              req.ID,
		Title:           req.Title,
		Artist:          req.Artist,
		Album:           req.Album,
		DurationSeconds: req.Duration,
	}
	client := rocksky.NewClient(m.httpClient)
	client.BaseURL = defaultListenBrainzEndpoint(payload.Endpoint)
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if listenType == "playing_now" {
		err = client.SubmitNowPlaying(ctx, payload.Token, track)
	} else {
		err = client.SubmitScrobble(ctx, payload.Token, track, time.Now())
	}
	if err != nil {
		httputil.WriteInternalError(w, r, "doListenBrainzScrobble", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *MusicService) handleListenBrainzNowPlaying(w http.ResponseWriter, r *http.Request) {
	m.doListenBrainzScrobble(w, r, "playing_now")
}

func (m *MusicService) handleListenBrainzScrobble(w http.ResponseWriter, r *http.Request) {
	m.doListenBrainzScrobble(w, r, "single")
}
