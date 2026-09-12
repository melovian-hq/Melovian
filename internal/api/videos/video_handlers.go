// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package videos

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"

	"melovian/internal/api/apishared"
	"melovian/internal/api/library"
	"melovian/internal/consts"
	"melovian/internal/httputil"
	"melovian/internal/store"
	"melovian/internal/video"
)

type localVideoView struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Artist    string `json:"artist,omitempty"`
	Album     string `json:"album,omitempty"`
	Duration  int    `json:"duration,omitempty"`
	Format    string `json:"format"`
	RelPath   string `json:"relPath"`
	MediaKind string `json:"mediaKind"`
}

type localVideosResponse struct {
	Videos []localVideoView `json:"videos"`
}

type trackVideoLinkView struct {
	TrackID string `json:"trackId"`
	Source  string `json:"source"`
	VideoID string `json:"videoId"`
	Title   string `json:"title,omitempty"`
	PlayID  string `json:"playId"`
}

type upsertTrackVideoLinkRequest struct {
	Source  string `json:"source"`
	VideoID string `json:"videoId"`
	Title   string `json:"title"`
}

type videoSearchResponse struct {
	Results  []video.SearchHit `json:"results"`
	Provider string            `json:"provider"`
}

type videoResolveResponse struct {
	Source   string `json:"source"`
	VideoID  string `json:"videoId"`
	Title    string `json:"title,omitempty"`
	Author   string `json:"author,omitempty"`
	EmbedURL string `json:"embedUrl"`
	PlayID   string `json:"playId"`
}

// Handler serves the video link, search, resolve, and settings routes.
type Handler struct {
	preferences        *store.PreferencesStore
	videoLinks         *store.TrackVideoLinkStore
	videoClient        *video.Client
	videoSettingsCache sync.Map
	localLibraries     *store.LocalLibraryStore
	localTracks        *store.LocalTrackStore
	library            *library.Handler
}

func New(preferences *store.PreferencesStore, videoLinks *store.TrackVideoLinkStore, videoClient *video.Client, libraries *store.LocalLibraryStore, tracks *store.LocalTrackStore, lib *library.Handler) *Handler {
	return &Handler{
		preferences:    preferences,
		videoLinks:     videoLinks,
		videoClient:    videoClient,
		localLibraries: libraries,
		localTracks:    tracks,
		library:        lib,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/local-music/videos", h.handleLocalVideosList)
	mux.HandleFunc("GET /api/local-music/videos/{id}", h.handleLocalVideoGet)
	mux.HandleFunc("GET /api/video/settings", h.handleGetVideoSettings)
	mux.HandleFunc("PUT /api/video/settings", h.handlePutVideoSettings)
	mux.HandleFunc("GET /api/video/search", h.handleVideoSearch)
	mux.HandleFunc("GET /api/video/resolve", h.handleVideoResolve)
	mux.HandleFunc("GET /api/video/links/{trackId}", h.handleGetTrackVideoLink)
	mux.HandleFunc("PUT /api/video/links/{trackId}", h.handlePutTrackVideoLink)
	mux.HandleFunc("DELETE /api/video/links/{trackId}", h.handleDeleteTrackVideoLink)
}

func localVideoViewFrom(track store.LocalTrack) localVideoView {
	duration := 0
	if track.DurationMs > 0 {
		duration = track.DurationMs / 1000
	}
	return localVideoView{
		ID:        track.ID,
		Title:     track.Title,
		Artist:    track.Artist,
		Album:     track.Album,
		Duration:  duration,
		Format:    track.Format,
		RelPath:   track.RelPath,
		MediaKind: store.NormalizeMediaKind(track.MediaKind),
	}
}

func playIDForLink(source, videoID string) string {
	switch source {
	case store.VideoSourceLocal:
		return videoID
	case store.VideoSourceInvidious:
		return "ext:invidious:" + videoID
	case store.VideoSourceYouTube:
		return "ext:youtube:" + videoID
	default:
		return videoID
	}
}

func (h *Handler) handleLocalVideosList(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireVideoEnabled(w, r); !ok {
		return
	}
	userID := apishared.UserIDFromContext(r.Context())
	libraryIDs, err := h.library.LibraryIDsForUser(userID)
	if err != nil {
		h.library.WriteLocalMusicError(w, r, err)
		return
	}
	tracks, err := h.localTracks.ListVideosInLibraries(libraryIDs)
	if err != nil {
		httputil.WriteInternalError(w, r, "handleLocalVideosList", err)
		return
	}
	videos := make([]localVideoView, 0, len(tracks))
	for _, track := range tracks {
		videos = append(videos, localVideoViewFrom(track))
	}
	httputil.WriteJSON(w, http.StatusOK, localVideosResponse{Videos: videos})
}

func (h *Handler) handleLocalVideoGet(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireVideoEnabled(w, r); !ok {
		return
	}
	userID := apishared.UserIDFromContext(r.Context())
	trackID := r.PathValue("id")
	track, err := h.localTracks.GetByID(trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleLocalVideoGet", err)
		return
	}
	if store.NormalizeMediaKind(track.MediaKind) != store.MediaKindVideo {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if _, err := h.localLibraries.GetForUser(userID, track.LibraryID); err != nil {
		h.library.WriteLocalMusicError(w, r, err)
		return
	}
	if track.Status != store.TrackStatusPresent || track.DuplicateOf != "" {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, localVideoViewFrom(track))
}

func (h *Handler) handleVideoSearch(w http.ResponseWriter, r *http.Request) {
	settings, ok := h.requireVideoEnabled(w, r)
	if !ok {
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		httputil.WriteError(w, http.StatusBadRequest, "q_is_required", "q is required")
		return
	}
	provider := strings.TrimSpace(r.URL.Query().Get("provider"))
	if provider == "" {
		provider = settings.SearchProvider
	} else {
		provider = normalizeVideoSearchProvider(provider)
	}
	ctx, cancel := context.WithTimeout(r.Context(), consts.UpstreamTimeout)
	defer cancel()

	var results []video.SearchHit
	var err error
	switch provider {
	case VideoSearchProviderYouTube:
		if settings.YouTubeAPIKey == "" {
			httputil.WriteError(w, http.StatusBadRequest, "configure_a_youtube_data_api_key_in_sett", "configure a YouTube Data API key in Settings → Video")
			return
		}
		results, err = h.videoClient.SearchYouTube(ctx, settings.YouTubeAPIKey, query)
	default:
		provider = VideoSearchProviderInvidious
		if settings.InvidiousBaseURL == "" {
			httputil.WriteError(w, http.StatusBadRequest, "configure_an_invidious_instance_in_setti", "configure an Invidious instance in Settings → Video")
			return
		}
		results, err = h.videoClient.SearchInvidious(ctx, settings.InvidiousBaseURL, query)
	}
	if err != nil {
		httputil.WriteInternalError(w, r, "handleVideoSearch", err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, videoSearchResponse{Results: results, Provider: provider})
}

func (h *Handler) handleVideoResolve(w http.ResponseWriter, r *http.Request) {
	settings, ok := h.requireVideoEnabled(w, r)
	if !ok {
		return
	}
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	videoID := strings.TrimSpace(r.URL.Query().Get("id"))
	title := strings.TrimSpace(r.URL.Query().Get("title"))
	if source == "" {
		if settings.SearchProvider == VideoSearchProviderYouTube {
			source = store.VideoSourceYouTube
		} else {
			source = store.VideoSourceInvidious
		}
	}
	normalizedSource, err := store.NormalizeVideoSource(source)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if videoID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "id_is_required", "id is required")
		return
	}
	if normalizedSource == store.VideoSourceLocal {
		httputil.WriteJSON(w, http.StatusOK, videoResolveResponse{
			Source:  normalizedSource,
			VideoID: videoID,
			Title:   title,
			PlayID:  playIDForLink(normalizedSource, videoID),
		})
		return
	}
	var embedURL string
	author := ""
	lookupCtx, cancel := context.WithTimeout(r.Context(), consts.UpstreamTimeout)
	defer cancel()
	switch normalizedSource {
	case store.VideoSourceYouTube:
		embedURL, err = video.YouTubeEmbedURL(videoID)
		if title == "" && settings.YouTubeAPIKey != "" {
			if info, lookupErr := h.videoClient.GetYouTubeVideo(lookupCtx, settings.YouTubeAPIKey, videoID); lookupErr == nil {
				title = info.Title
				author = info.Author
			}
		}
	default:
		if settings.InvidiousBaseURL == "" {
			httputil.WriteError(w, http.StatusBadRequest, "configure_an_invidious_instance_in_setti", "configure an Invidious instance in Settings → Video")
			return
		}
		embedURL, err = video.EmbedURL(settings.InvidiousBaseURL, videoID)
		if title == "" {
			if info, lookupErr := h.videoClient.GetInvidiousVideo(lookupCtx, settings.InvidiousBaseURL, videoID); lookupErr == nil {
				title = info.Title
				author = info.Author
			}
		}
	}
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, videoResolveResponse{
		Source:   normalizedSource,
		VideoID:  videoID,
		Title:    title,
		Author:   author,
		EmbedURL: embedURL,
		PlayID:   playIDForLink(normalizedSource, videoID),
	})
}

func (h *Handler) handleGetTrackVideoLink(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireVideoEnabled(w, r); !ok {
		return
	}
	userID := apishared.ResolveProgressUserID(r.Context())
	trackID := r.PathValue("trackId")
	link, err := h.videoLinks.Get(userID, trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleGetTrackVideoLink", err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, trackVideoLinkView{
		TrackID: link.TrackID,
		Source:  link.Source,
		VideoID: link.VideoID,
		Title:   link.Title,
		PlayID:  playIDForLink(link.Source, link.VideoID),
	})
}

func (h *Handler) handlePutTrackVideoLink(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireVideoEnabled(w, r); !ok {
		return
	}
	userID := apishared.ResolveProgressUserID(r.Context())
	trackID := r.PathValue("trackId")
	var req upsertTrackVideoLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	link, err := h.videoLinks.Upsert(store.UpsertTrackVideoLinkInput{
		UserID:  userID,
		TrackID: trackID,
		Source:  req.Source,
		VideoID: req.VideoID,
		Title:   req.Title,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, trackVideoLinkView{
		TrackID: link.TrackID,
		Source:  link.Source,
		VideoID: link.VideoID,
		Title:   link.Title,
		PlayID:  playIDForLink(link.Source, link.VideoID),
	})
}

func (h *Handler) handleDeleteTrackVideoLink(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireVideoEnabled(w, r); !ok {
		return
	}
	userID := apishared.ResolveProgressUserID(r.Context())
	trackID := r.PathValue("trackId")
	if err := h.videoLinks.Delete(userID, trackID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		httputil.WriteInternalError(w, r, "handleDeleteTrackVideoLink", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
