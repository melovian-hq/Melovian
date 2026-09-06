// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonicserver

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"melovian/internal/jukebox"
	"melovian/internal/localmusic"
)

func (s *Server) handleCreatePlaylist(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	lp := s.extended()
	if lp == nil {
		writeError(w, r, 70, "playlists unavailable")
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		writeError(w, r, 10, "name is required")
		return
	}
	songIDs := queryIDs(r)
	pl, err := lp.CreatePlaylist(ctx, userID, name, songIDs)
	if err != nil {
		writeError(w, r, 0, err.Error())
		return
	}
	writeOK(w, r, map[string]any{"playlist": playlistView(pl)})
}

func (s *Server) handleUpdatePlaylist(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	lp := s.extended()
	if lp == nil {
		writeError(w, r, 70, "playlists unavailable")
		return
	}
	playlistID := strings.TrimSpace(r.URL.Query().Get("playlistId"))
	if playlistID == "" {
		writeError(w, r, 10, "playlistId is required")
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	songIDs := queryIDs(r)
	pl, err := lp.UpdatePlaylist(ctx, userID, playlistID, name, songIDs)
	if err != nil {
		writeError(w, r, 70, err.Error())
		return
	}
	writeOK(w, r, map[string]any{"playlist": playlistView(pl)})
}

func (s *Server) handleDeletePlaylist(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	lp := s.extended()
	if lp == nil {
		writeError(w, r, 70, "playlists unavailable")
		return
	}
	playlistID := strings.TrimSpace(r.URL.Query().Get("id"))
	if err := lp.DeletePlaylist(ctx, userID, playlistID); err != nil {
		writeError(w, r, 70, err.Error())
		return
	}
	writeOK(w, r, map[string]any{})
}

func (s *Server) handleGetShares(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	lp := s.extended()
	if lp == nil {
		writeError(w, r, 70, "shares unavailable")
		return
	}
	items, err := lp.ListShares(ctx, userID)
	if err != nil {
		writeError(w, r, 0, err.Error())
		return
	}
	shares := make([]map[string]any, len(items))
	for i, item := range items {
		shares[i] = shareView(item)
	}
	writeOK(w, r, map[string]any{"shares": map[string]any{"share": shares}})
}

func (s *Server) handleCreateShare(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	lp := s.extended()
	if lp == nil {
		writeError(w, r, 70, "shares unavailable")
		return
	}
	id := strings.TrimSpace(firstQuery(r, "id", "songId", "albumId", "playlistId"))
	if id == "" {
		writeError(w, r, 10, "id is required")
		return
	}
	description := strings.TrimSpace(r.URL.Query().Get("description"))
	var expires *time.Time
	if raw := strings.TrimSpace(r.URL.Query().Get("expires")); raw != "" {
		if ms, err := strconv.ParseInt(raw, 10, 64); err == nil {
			t := time.UnixMilli(ms)
			expires = &t
		}
	}
	item, err := lp.CreateShare(ctx, userID, id, description, expires)
	if err != nil {
		writeError(w, r, 0, err.Error())
		return
	}
	writeOK(w, r, map[string]any{"share": shareView(item)})
}

func (s *Server) handleDeleteShare(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	lp := s.extended()
	if lp == nil {
		writeError(w, r, 70, "shares unavailable")
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if err := lp.DeleteShare(ctx, userID, id); err != nil {
		writeError(w, r, 70, err.Error())
		return
	}
	writeOK(w, r, map[string]any{})
}

func (s *Server) handleGetJukeboxStatus(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	lp := s.extended()
	if lp == nil {
		writeError(w, r, 70, "jukebox unavailable")
		return
	}
	status := lp.JukeboxStatus(ctx)
	writeOK(w, r, map[string]any{"jukeboxStatus": jukeboxView(status)})
}

func (s *Server) handleJukeboxControl(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	lp := s.extended()
	if lp == nil {
		writeError(w, r, 70, "jukebox unavailable")
		return
	}
	action := strings.TrimSpace(r.URL.Query().Get("action"))
	position, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	gain, _ := strconv.ParseFloat(r.URL.Query().Get("gain"), 64)
	var track *jukebox.Track
	if id := strings.TrimSpace(firstQuery(r, "id", "songId")); id != "" {
		songTrack, _, err := s.provider.Track(ctx, userID, id)
		if err == nil {
			catalog, catErr := s.provider.Catalog(ctx, userID)
			if catErr == nil {
				if song, ok := catalog.Song(songTrack.ID); ok {
					track = &jukebox.Track{
						ID:       song.ID,
						Title:    song.Title,
						Artist:   song.Artist,
						Album:    song.Album,
						Duration: song.Duration,
						CoverArt: song.CoverArt,
					}
				}
			}
		}
	}
	status := lp.JukeboxControl(ctx, action, track, position, gain, nil)
	writeOK(w, r, map[string]any{"jukeboxStatus": jukeboxView(status)})
}

func firstQuery(r *http.Request, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(r.URL.Query().Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func shareView(item ShareSummary) map[string]any {
	view := map[string]any{
		"id":          item.ID,
		"url":         item.URL,
		"description": item.Description,
		"created":     formatSubsonicTime(item.Created),
		"visitCount":  item.VisitCount,
	}
	if item.Expires != nil {
		view["expires"] = formatSubsonicTime(*item.Expires)
	}
	if strings.HasPrefix(item.ResourceID, "alb_") {
		view["entry"] = map[string]any{"id": item.ResourceID, "isDir": true}
	} else {
		view["entry"] = map[string]any{"id": item.ResourceID, "isDir": false}
	}
	return view
}

func jukeboxView(status jukebox.Status) map[string]any {
	view := map[string]any{
		"playing":     status.Playing,
		"gain":        status.Gain,
		"position":    status.PositionSec,
		"changeCount": 1,
	}
	if status.Current != nil {
		view["currentIndex"] = 0
		view["entry"] = songView(localmusic.Song{
			ID:       status.Current.ID,
			Title:    status.Current.Title,
			Artist:   status.Current.Artist,
			Album:    status.Current.Album,
			Duration: status.Current.Duration,
			CoverArt: status.Current.CoverArt,
		})
	}
	return view
}
