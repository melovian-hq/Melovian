// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

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

func (s *Server) registerVideoRoutes() {
	s.mux.HandleFunc("GET /api/local-music/videos", s.handleLocalVideosList)
	s.mux.HandleFunc("GET /api/local-music/videos/{id}", s.handleLocalVideoGet)
	s.mux.HandleFunc("GET /api/video/settings", s.handleGetVideoSettings)
	s.mux.HandleFunc("PUT /api/video/settings", s.handlePutVideoSettings)
	s.mux.HandleFunc("GET /api/video/search", s.handleVideoSearch)
	s.mux.HandleFunc("GET /api/video/resolve", s.handleVideoResolve)
	s.mux.HandleFunc("GET /api/video/links/{trackId}", s.handleGetTrackVideoLink)
	s.mux.HandleFunc("PUT /api/video/links/{trackId}", s.handlePutTrackVideoLink)
	s.mux.HandleFunc("DELETE /api/video/links/{trackId}", s.handleDeleteTrackVideoLink)
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

func (s *Server) handleLocalVideosList(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireVideoEnabled(w, r); !ok {
		return
	}
	userID := UserIDFromContext(r.Context())
	libraryIDs, err := s.libraryIDsForUser(userID)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	tracks, err := s.localTracks.ListVideosInLibraries(libraryIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	videos := make([]localVideoView, 0, len(tracks))
	for _, track := range tracks {
		videos = append(videos, localVideoViewFrom(track))
	}
	httputil.WriteJSON(w, http.StatusOK, localVideosResponse{Videos: videos})
}

func (s *Server) handleLocalVideoGet(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireVideoEnabled(w, r); !ok {
		return
	}
	userID := UserIDFromContext(r.Context())
	trackID := r.PathValue("id")
	track, err := s.localTracks.GetByID(trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if store.NormalizeMediaKind(track.MediaKind) != store.MediaKindVideo {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if _, err := s.localLibraries.GetForUser(userID, track.LibraryID); err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	if track.Status != store.TrackStatusPresent || track.DuplicateOf != "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, localVideoViewFrom(track))
}

func (s *Server) handleVideoSearch(w http.ResponseWriter, r *http.Request) {
	settings, ok := s.requireVideoEnabled(w, r)
	if !ok {
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		http.Error(w, "q is required", http.StatusBadRequest)
		return
	}
	provider := strings.TrimSpace(r.URL.Query().Get("provider"))
	if provider == "" {
		provider = settings.SearchProvider
	} else {
		provider = normalizeVideoSearchProvider(provider)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	var results []video.SearchHit
	var err error
	switch provider {
	case VideoSearchProviderYouTube:
		if settings.YouTubeAPIKey == "" {
			http.Error(w, "configure a YouTube Data API key in Settings → Video", http.StatusBadRequest)
			return
		}
		results, err = s.videoClient.SearchYouTube(ctx, settings.YouTubeAPIKey, query)
	default:
		provider = VideoSearchProviderInvidious
		if settings.InvidiousBaseURL == "" {
			http.Error(w, "configure an Invidious instance in Settings → Video", http.StatusBadRequest)
			return
		}
		results, err = s.videoClient.SearchInvidious(ctx, settings.InvidiousBaseURL, query)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, videoSearchResponse{Results: results, Provider: provider})
}

func (s *Server) handleVideoResolve(w http.ResponseWriter, r *http.Request) {
	settings, ok := s.requireVideoEnabled(w, r)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if videoID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
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
	lookupCtx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	switch normalizedSource {
	case store.VideoSourceYouTube:
		embedURL, err = video.YouTubeEmbedURL(videoID)
		if title == "" && settings.YouTubeAPIKey != "" {
			if info, lookupErr := s.videoClient.GetYouTubeVideo(lookupCtx, settings.YouTubeAPIKey, videoID); lookupErr == nil {
				title = info.Title
				author = info.Author
			}
		}
	default:
		if settings.InvidiousBaseURL == "" {
			http.Error(w, "configure an Invidious instance in Settings → Video", http.StatusBadRequest)
			return
		}
		embedURL, err = video.EmbedURL(settings.InvidiousBaseURL, videoID)
		if title == "" {
			if info, lookupErr := s.videoClient.GetInvidiousVideo(lookupCtx, settings.InvidiousBaseURL, videoID); lookupErr == nil {
				title = info.Title
				author = info.Author
			}
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

func (s *Server) handleGetTrackVideoLink(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireVideoEnabled(w, r); !ok {
		return
	}
	userID := ResolveProgressUserID(r.Context())
	trackID := r.PathValue("trackId")
	link, err := s.videoLinks.Get(userID, trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

func (s *Server) handlePutTrackVideoLink(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireVideoEnabled(w, r); !ok {
		return
	}
	userID := ResolveProgressUserID(r.Context())
	trackID := r.PathValue("trackId")
	var req upsertTrackVideoLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	link, err := s.videoLinks.Upsert(store.UpsertTrackVideoLinkInput{
		UserID:  userID,
		TrackID: trackID,
		Source:  req.Source,
		VideoID: req.VideoID,
		Title:   req.Title,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

func (s *Server) handleDeleteTrackVideoLink(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireVideoEnabled(w, r); !ok {
		return
	}
	userID := ResolveProgressUserID(r.Context())
	trackID := r.PathValue("trackId")
	if err := s.videoLinks.Delete(userID, trackID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
