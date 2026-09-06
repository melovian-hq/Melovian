// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"melovian/internal/httputil"
	"melovian/internal/jukebox"
)

func (s *Server) registerJukeboxRoutes() {
	s.mux.HandleFunc("GET /api/jukebox/status", s.handleJukeboxStatus)
	s.mux.HandleFunc("POST /api/jukebox/control", s.handleJukeboxControl)
}

func (s *Server) handleJukeboxStatus(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, s.jukebox.Status())
}

func (s *Server) handleJukeboxControl(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action      string          `json:"action"`
		TrackID     string          `json:"trackId"`
		PositionSec int             `json:"positionSec"`
		Gain        float64         `json:"gain"`
		Queue       []jukebox.Track `json:"queue"`
	}
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	var track *jukebox.Track
	if req.TrackID != "" {
		localTrack, err := s.localTracks.GetByID(req.TrackID)
		if err == nil {
			track = &jukebox.Track{
				ID:       localTrack.ID,
				Title:    localTrack.Title,
				Artist:   localTrack.Artist,
				Album:    localTrack.Album,
				Duration: localTrack.DurationMs / 1000,
				CoverArt: localTrack.ID,
			}
		}
	}
	status := s.jukebox.Control(req.Action, track, req.PositionSec, req.Gain, req.Queue)
	httputil.WriteJSON(w, http.StatusOK, status)
}
