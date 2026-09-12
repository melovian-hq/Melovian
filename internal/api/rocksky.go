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

type rockskySettings struct {
	Enabled  bool   `json:"enabled"`
	HasToken bool   `json:"hasToken"`
	Token    string `json:"token"`
}

type rockskyTokenPayload struct {
	Token string `json:"token"`
}

func (m *MusicService) rockskyEnabled() bool {
	return extensions.IsEnabled(m.dataDir, "rocksky")
}

func (m *MusicService) loadRockskyToken(userID string) (string, error) {
	raw, err := m.preferences.Get(userID, store.PrefKeyRockskySettings)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	var payload rockskyTokenPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", err
	}
	return strings.TrimSpace(payload.Token), nil
}

func (m *MusicService) saveRockskyToken(userID, token string) error {
	if token == "" {
		return m.preferences.Delete(userID, store.PrefKeyRockskySettings)
	}
	data, err := json.Marshal(rockskyTokenPayload{Token: token})
	if err != nil {
		return err
	}
	return m.preferences.Set(userID, store.PrefKeyRockskySettings, string(data))
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return ""
	}
	return "..." + token[len(token)-4:]
}

func (m *MusicService) handleGetRockskySettings(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	token, err := m.loadRockskyToken(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load rocksky settings")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, rockskySettings{
		Enabled:  m.rockskyEnabled(),
		HasToken: token != "",
		Token:    maskToken(token),
	})
}

func (m *MusicService) handlePutRockskySettings(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	var payload rockskyTokenPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	if err := m.saveRockskyToken(userID, strings.TrimSpace(payload.Token)); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to save rocksky settings")
		return
	}
	m.handleGetRockskySettings(w, r)
}

func (m *MusicService) handleTestRockskyToken(w http.ResponseWriter, r *http.Request) {
	userID := ResolveProgressUserID(r.Context())
	var payload rockskyTokenPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	token := strings.TrimSpace(payload.Token)
	if token == "" {
		token, _ = m.loadRockskyToken(userID)
	}
	if token == "" {
		httputil.WriteError(w, http.StatusBadRequest, "rocksky_token_is_not_configured", "rocksky token is not configured")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	userName, err := m.rocksky.ValidateToken(ctx, token)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "token_test_failed", fmt.Sprintf("rocksky token test failed: %s", err.Error()))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"userName": userName,
	})
}

func (m *MusicService) doRockskyScrobble(w http.ResponseWriter, r *http.Request, listenType string) {
	if !m.rockskyEnabled() {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	userID := ResolveProgressUserID(r.Context())
	token, err := m.loadRockskyToken(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load rocksky token")
		return
	}
	if token == "" {
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
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if listenType == "playing_now" {
		err = m.rocksky.SubmitNowPlaying(ctx, token, track)
	} else {
		err = m.rocksky.SubmitScrobble(ctx, token, track, time.Now())
	}
	if err != nil {
		httputil.WriteInternalError(w, r, "doRockskyScrobble", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *MusicService) handleRockskyNowPlaying(w http.ResponseWriter, r *http.Request) {
	m.doRockskyScrobble(w, r, "playing_now")
}

func (m *MusicService) handleRockskyScrobble(w http.ResponseWriter, r *http.Request) {
	m.doRockskyScrobble(w, r, "single")
}
