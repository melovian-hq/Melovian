// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package music

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"melovian/internal/api/apishared"
	"melovian/internal/httputil"
	"melovian/internal/lastfm"
	"melovian/internal/rocksky"
	"melovian/internal/store"
)

type MusicService struct {
	listen      *store.ListenStore
	preferences *store.PreferencesStore
	dataDir     string
	httpClient  *http.Client
	rocksky     *rocksky.Client
	lastfm      *lastfm.Client
}

func NewMusicService(listen *store.ListenStore, preferences *store.PreferencesStore, dataDir string, httpClient *http.Client) *MusicService {
	return &MusicService{
		listen:      listen,
		preferences: preferences,
		dataDir:     dataDir,
		httpClient:  httpClient,
		rocksky:     rocksky.NewClient(httpClient),
		lastfm:      lastfm.NewClient(httpClient),
	}
}

func (m *MusicService) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/music/history", m.handleHistory)
	mux.HandleFunc("GET /api/music/listen-events", m.handleListenEvents)
	mux.HandleFunc("DELETE /api/music/listen-events", m.handleClearListenEvents)
	mux.HandleFunc("GET /api/music/listen-events/years", m.handleListenEventYears)
	mux.HandleFunc("GET /api/music/resume", m.handleResume)
	mux.HandleFunc("GET /api/music/stats", m.handleStats)
	mux.HandleFunc("GET /api/music/batch", m.handleBatch)
	mux.HandleFunc("GET /api/music/items/{trackId}", m.handleGet)
	mux.HandleFunc("PUT /api/music/items/{trackId}", m.handleUpsert)
	mux.HandleFunc("POST /api/music/items/{trackId}/played", m.handlePlayed)
	mux.HandleFunc("DELETE /api/music/items/{trackId}", m.handleDelete)

	mux.HandleFunc("GET /api/music/playlists", m.handleListPlaylists)
	mux.HandleFunc("POST /api/music/playlists", m.handleCreatePlaylist)
	mux.HandleFunc("GET /api/music/playlists/{playlistId}", m.handleGetPlaylist)
	mux.HandleFunc("PATCH /api/music/playlists/{playlistId}", m.handleRenamePlaylist)
	mux.HandleFunc("DELETE /api/music/playlists/{playlistId}", m.handleDeletePlaylist)
	mux.HandleFunc("PUT /api/music/playlists/{playlistId}/tracks", m.handleSetTracks)
	mux.HandleFunc("POST /api/music/playlists/{playlistId}/tracks", m.handleAddTrack)
	mux.HandleFunc("DELETE /api/music/playlists/{playlistId}/tracks/{trackId}", m.handleRemoveTrack)

	mux.HandleFunc("GET /api/music/favorites", m.handleListFavorites)
	mux.HandleFunc("POST /api/music/favorites/{trackId}", m.handleAddFavorite)
	mux.HandleFunc("DELETE /api/music/favorites/{trackId}", m.handleRemoveFavorite)

	mux.HandleFunc("GET /api/music/settings/eq", m.handleGetEq)
	mux.HandleFunc("PUT /api/music/settings/eq", m.handlePutEq)
	mux.HandleFunc("GET /api/music/settings/connection", m.handleGetConnectionSettings)
	mux.HandleFunc("PUT /api/music/settings/connection", m.handlePutConnectionSettings)
	mux.HandleFunc("GET /api/music/settings/rocksky", m.handleGetRockskySettings)
	mux.HandleFunc("PUT /api/music/settings/rocksky", m.handlePutRockskySettings)
	mux.HandleFunc("POST /api/music/settings/rocksky/test", m.handleTestRockskyToken)
	mux.HandleFunc("POST /api/music/rocksky/now-playing", m.handleRockskyNowPlaying)
	mux.HandleFunc("POST /api/music/rocksky/scrobble", m.handleRockskyScrobble)
	mux.HandleFunc("GET /api/music/settings/listenbrainz", m.handleGetListenBrainzSettings)
	mux.HandleFunc("PUT /api/music/settings/listenbrainz", m.handlePutListenBrainzSettings)
	mux.HandleFunc("POST /api/music/settings/listenbrainz/test", m.handleTestListenBrainzToken)
	mux.HandleFunc("POST /api/music/listenbrainz/now-playing", m.handleListenBrainzNowPlaying)
	mux.HandleFunc("POST /api/music/listenbrainz/scrobble", m.handleListenBrainzScrobble)
	mux.HandleFunc("GET /api/music/settings/lastfm", m.handleGetLastFMSettings)
	mux.HandleFunc("PUT /api/music/settings/lastfm", m.handlePutLastFMSettings)
	mux.HandleFunc("POST /api/music/settings/lastfm/test", m.handleTestLastFMToken)
	mux.HandleFunc("POST /api/music/lastfm/now-playing", m.handleLastFMNowPlaying)
	mux.HandleFunc("POST /api/music/lastfm/scrobble", m.handleLastFMScrobble)
}

type listenUpsertRequest struct {
	PositionMs    int64  `json:"positionMs"`
	Played        bool   `json:"played"`
	IncrementPlay bool   `json:"incrementPlay"`
	DeltaMs       int64  `json:"deltaMs"`
	TrackTitle    string `json:"trackTitle"`
	ArtistName    string `json:"artistName"`
	AlbumID       string `json:"albumId"`
	AlbumTitle    string `json:"albumTitle"`
	DurationMs    int    `json:"durationMs"`
	CoverArtID    string `json:"coverArtId"`
}

type listenItemsResponse struct {
	Items []store.ListenProgressJSON `json:"items"`
}

type statEntryResponse struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

type musicStatsResponse struct {
	TotalPlays       int                 `json:"totalPlays"`
	UniqueTracks     int                 `json:"uniqueTracks"`
	TotalListeningMs int64               `json:"totalListeningMs"`
	TopArtists       []statEntryResponse `json:"topArtists"`
	TopTracks        []statEntryResponse `json:"topTracks"`
	TopAlbums        []statEntryResponse `json:"topAlbums"`
}

func (m *MusicService) handleResume(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	limit := httputil.QueryIntClamped(r, "limit", 18, 1, 500)
	items, err := m.listen.ResumeTracks(userID, limit)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load resume tracks")
		return
	}

	payload := make([]store.ListenProgressJSON, len(items))
	for i, item := range items {
		payload[i] = item.JSON()
	}
	httputil.WriteJSON(w, http.StatusOK, listenItemsResponse{Items: payload})
}

func (m *MusicService) handleHistory(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	limit := httputil.QueryIntClamped(r, "limit", 50, 1, 500)
	items, err := m.listen.History(userID, limit)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load history")
		return
	}

	payload := make([]store.ListenProgressJSON, len(items))
	for i, item := range items {
		payload[i] = item.JSON()
	}
	httputil.WriteJSON(w, http.StatusOK, listenItemsResponse{Items: payload})
}

type listenEventsResponse struct {
	Items   []store.ListenEventJSON `json:"items"`
	HasMore bool                    `json:"hasMore"`
}

func (m *MusicService) handleListenEvents(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	limit := httputil.QueryIntClamped(r, "limit", 100, 1, 500)
	offset := httputil.QueryIntClamped(r, "offset", 0, 0, 1_000_000)
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	period := strings.TrimSpace(r.URL.Query().Get("period"))

	since, until := listenPeriodBounds(period)
	items, err := m.listen.ListListenEvents(userID, store.ListenEventsQuery{
		Limit:  limit + 1,
		Offset: offset,
		Since:  since,
		Until:  until,
		Search: search,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load listen events")
		return
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	payload := make([]store.ListenEventJSON, len(items))
	for i, item := range items {
		payload[i] = item.JSON()
	}
	httputil.WriteJSON(w, http.StatusOK, listenEventsResponse{Items: payload, HasMore: hasMore})
}

func (m *MusicService) handleListenEventYears(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	years, err := m.listen.ListenEventYears(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load listen years")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"years": years})
}

func (m *MusicService) handleClearListenEvents(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	if err := m.listen.ClearHistory(userID); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to clear listen history")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func listenPeriodBounds(period string) (since int64, until int64) {
	if period == "" || period == "all" {
		return 0, 0
	}
	if after, ok := strings.CutPrefix(period, "year:"); ok {
		yearStr := after
		year, err := strconv.Atoi(yearStr)
		if err != nil || year < 1970 {
			return 0, 0
		}
		start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
		return start.Unix(), end.Unix()
	}

	now := time.Now()
	var duration time.Duration
	switch period {
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	case "90d":
		duration = 90 * 24 * time.Hour
	default:
		return 0, 0
	}
	return now.Add(-duration).Unix(), 0
}

func (m *MusicService) handleStats(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	limit := httputil.QueryInt(r, "limit", 8)
	stats, err := m.listen.Stats(userID, limit)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load stats")
		return
	}

	topArtists := make([]statEntryResponse, len(stats.TopArtists))
	for i, entry := range stats.TopArtists {
		topArtists[i] = statEntryResponse{Key: entry.Key, Label: entry.Label, Count: entry.Count}
	}
	topTracks := make([]statEntryResponse, len(stats.TopTracks))
	for i, entry := range stats.TopTracks {
		topTracks[i] = statEntryResponse{Key: entry.Key, Label: entry.Label, Count: entry.Count}
	}
	topAlbums := make([]statEntryResponse, len(stats.TopAlbums))
	for i, entry := range stats.TopAlbums {
		topAlbums[i] = statEntryResponse{Key: entry.Key, Label: entry.Label, Count: entry.Count}
	}

	httputil.WriteJSON(w, http.StatusOK, musicStatsResponse{
		TotalPlays:       stats.TotalPlays,
		UniqueTracks:     stats.UniqueTracks,
		TotalListeningMs: stats.TotalListeningMs,
		TopArtists:       topArtists,
		TopTracks:        topTracks,
		TopAlbums:        topAlbums,
	})
}

func (m *MusicService) handleBatch(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	ids := strings.Split(r.URL.Query().Get("ids"), ",")
	filtered := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			filtered = append(filtered, id)
		}
		if len(filtered) >= 200 {
			break
		}
	}

	items, err := m.listen.Batch(userID, filtered)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load progress")
		return
	}

	payload := make(map[string]store.ListenProgressJSON, len(items))
	for id, item := range items {
		payload[id] = item.JSON()
	}
	httputil.WriteJSON(w, http.StatusOK, payload)
}

func (m *MusicService) handleGet(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	trackID := r.PathValue("trackId")
	item, err := m.listen.Get(userID, trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteJSON(w, http.StatusOK, map[string]any{})
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load progress")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, item.JSON())
}

func (m *MusicService) handleUpsert(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	trackID := r.PathValue("trackId")

	var req listenUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}

	if err := m.listen.Upsert(userID, store.ListenUpsertInput{
		TrackID:       trackID,
		PositionMs:    req.PositionMs,
		Played:        req.Played,
		IncrementPlay: req.IncrementPlay,
		DeltaMs:       req.DeltaMs,
		TrackTitle:    req.TrackTitle,
		ArtistName:    req.ArtistName,
		AlbumID:       req.AlbumID,
		AlbumTitle:    req.AlbumTitle,
		DurationMs:    req.DurationMs,
		CoverArtID:    req.CoverArtID,
	}); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	item, err := m.listen.Get(userID, trackID)
	if err != nil {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{"saved": true})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, item.JSON())
}

func (m *MusicService) handlePlayed(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	trackID := r.PathValue("trackId")
	if err := m.listen.MarkPlayed(userID, trackID); err != nil {
		httputil.WriteInternalError(w, r, "handlePlayed", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *MusicService) handleDelete(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	trackID := r.PathValue("trackId")
	if err := m.listen.Delete(userID, trackID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		httputil.WriteInternalError(w, r, "handleDelete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *MusicService) handleListPlaylists(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	playlists, err := m.listen.ListPlaylists(userID)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load playlists")
		return
	}

	payload := make([]map[string]any, 0, len(playlists))
	for _, pl := range playlists {
		entry := pl.ToJSON()
		entry["tracks"] = nil
		payload = append(payload, entry)
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"playlists": payload})
}

type createPlaylistRequest struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	RulesJSON string `json:"rulesJson"`
}

func (m *MusicService) handleCreatePlaylist(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	var req createPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}

	pl, err := m.listen.CreatePlaylistOpts(userID, store.CreatePlaylistOpts{
		Name:      req.Name,
		Kind:      req.Kind,
		RulesJSON: req.RulesJSON,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, pl.ToJSON())
}

func (m *MusicService) handleGetPlaylist(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	playlistID := r.PathValue("playlistId")
	pl, err := m.listen.GetPlaylist(userID, playlistID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load playlist")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, pl.ToJSON())
}

type renamePlaylistRequest struct {
	Name string `json:"name"`
}

func (m *MusicService) handleRenamePlaylist(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	playlistID := r.PathValue("playlistId")
	var req renamePlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}

	if err := m.listen.RenamePlaylist(userID, playlistID, req.Name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	pl, err := m.listen.GetPlaylist(userID, playlistID)
	if err != nil {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{"saved": true})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, pl.ToJSON())
}

func (m *MusicService) handleDeletePlaylist(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	playlistID := r.PathValue("playlistId")
	if err := m.listen.DeletePlaylist(userID, playlistID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleDeletePlaylist", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type setTracksRequest struct {
	Tracks []store.PlaylistTrack `json:"tracks"`
}

func (m *MusicService) handleSetTracks(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	playlistID := r.PathValue("playlistId")
	var req setTracksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}

	if err := m.listen.SetPlaylistTracks(userID, playlistID, req.Tracks); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	pl, _ := m.listen.GetPlaylist(userID, playlistID)
	httputil.WriteJSON(w, http.StatusOK, pl.ToJSON())
}

func (m *MusicService) handleAddTrack(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	playlistID := r.PathValue("playlistId")
	var track store.PlaylistTrack
	if err := json.NewDecoder(r.Body).Decode(&track); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}

	if err := m.listen.AddPlaylistTrack(userID, playlistID, track); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	pl, _ := m.listen.GetPlaylist(userID, playlistID)
	httputil.WriteJSON(w, http.StatusOK, pl.ToJSON())
}

func (m *MusicService) handleRemoveTrack(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	playlistID := r.PathValue("playlistId")
	trackID := r.PathValue("trackId")
	if err := m.listen.RemovePlaylistTrack(userID, playlistID, trackID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleRemoveTrack", err)
		return
	}

	pl, _ := m.listen.GetPlaylist(userID, playlistID)
	httputil.WriteJSON(w, http.StatusOK, pl.ToJSON())
}

type favoriteTrackRequest struct {
	TrackTitle string `json:"trackTitle"`
	ArtistName string `json:"artistName"`
	AlbumID    string `json:"albumId"`
	AlbumTitle string `json:"albumTitle"`
	DurationMs int    `json:"durationMs"`
	CoverArtID string `json:"coverArtId"`
}

func (m *MusicService) handleListFavorites(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	limit := httputil.QueryInt(r, "limit", 200)
	items, err := m.listen.ListFavorites(userID, limit)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load favorites")
		return
	}

	payload := make([]map[string]any, 0, len(items))
	for _, item := range items {
		payload = append(payload, item.ToJSON())
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"items": payload})
}

func (m *MusicService) handleAddFavorite(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	trackID := r.PathValue("trackId")

	var req favoriteTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}

	if err := m.listen.AddFavorite(userID, store.FavoriteTrack{
		TrackID:    trackID,
		TrackTitle: req.TrackTitle,
		ArtistName: req.ArtistName,
		AlbumID:    req.AlbumID,
		AlbumTitle: req.AlbumTitle,
		DurationMs: req.DurationMs,
		CoverArtID: req.CoverArtID,
	}); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *MusicService) handleRemoveFavorite(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	trackID := r.PathValue("trackId")
	if err := m.listen.RemoveFavorite(userID, trackID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		httputil.WriteInternalError(w, r, "handleRemoveFavorite", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *MusicService) handleGetEq(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	raw, err := m.preferences.Get(userID, store.PrefKeyMusicEq)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteJSON(w, http.StatusOK, map[string]any{})
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load eq settings")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(raw))
}

func (m *MusicService) handlePutEq(w http.ResponseWriter, r *http.Request) {
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
	if err := m.preferences.Set(userID, store.PrefKeyMusicEq, string(body)); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body) //#nosec G705 -- echoes validated JSON preference blob
}

func (m *MusicService) handleGetConnectionSettings(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	raw, err := m.preferences.Get(userID, store.PrefKeyConnectionSettings)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteJSON(w, http.StatusOK, map[string]any{})
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load connection settings")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(raw))
}

func (m *MusicService) handlePutConnectionSettings(w http.ResponseWriter, r *http.Request) {
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
	if err := m.preferences.Set(userID, store.PrefKeyConnectionSettings, string(body)); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body) //#nosec G705 -- echoes validated JSON preference blob
}
