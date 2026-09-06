// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonicserver

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"melovian/internal/transcode"
)

func (s *Server) library() LibraryProvider {
	if lp, ok := s.provider.(LibraryProvider); ok {
		return lp
	}
	return nil
}

func (s *Server) handleGetPlaylists(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	lp := s.library()
	if lp == nil {
		writeError(w, r, 70, "playlists unavailable")
		return
	}
	items, err := lp.ListPlaylists(ctx, userID)
	if err != nil {
		writeError(w, r, 0, err.Error())
		return
	}
	playlists := make([]map[string]any, len(items))
	for i, item := range items {
		playlists[i] = playlistView(item)
	}
	writeOK(w, r, map[string]any{
		"playlists": map[string]any{"playlist": playlists},
	})
}

func (s *Server) handleGetPlaylist(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	lp := s.library()
	if lp == nil {
		writeError(w, r, 70, "playlists unavailable")
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	summary, songs, err := lp.GetPlaylist(ctx, userID, id)
	if err != nil {
		writeError(w, r, 70, "playlist not found")
		return
	}
	writeOK(w, r, map[string]any{
		"playlist": map[string]any{
			"id":        summary.ID,
			"name":      summary.Name,
			"songCount": summary.SongCount,
			"duration":  summary.Duration,
			"public":    summary.Public,
			"created":   formatSubsonicTime(summary.Created),
			"changed":   formatSubsonicTime(summary.Changed),
			"song":      songViews(songs),
		},
	})
}

func (s *Server) handleGetStarred2(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	lp := s.library()
	if lp == nil {
		writeError(w, r, 70, "starred unavailable")
		return
	}
	ids, err := lp.ListStarredIDs(ctx, userID)
	if err != nil {
		writeError(w, r, 0, err.Error())
		return
	}
	catalog, err := s.provider.Catalog(ctx, userID)
	if err != nil {
		writeError(w, r, 0, err.Error())
		return
	}
	songs := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		if song, ok := catalog.Song(id); ok {
			view := songView(song)
			view["starred"] = formatSubsonicTime(time.Now())
			songs = append(songs, view)
		}
	}
	writeOK(w, r, map[string]any{
		"starred2": map[string]any{"song": songs},
	})
}

func (s *Server) handleStar(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	lp := s.library()
	if lp == nil {
		writeError(w, r, 70, "star unavailable")
		return
	}
	for _, id := range queryIDs(r) {
		track, _, err := s.provider.Track(ctx, userID, id)
		if err != nil {
			continue
		}
		_ = lp.Star(ctx, userID, id, track)
	}
	writeOK(w, r, map[string]any{})
}

func (s *Server) handleUnstar(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	lp := s.library()
	if lp == nil {
		writeError(w, r, 70, "unstar unavailable")
		return
	}
	for _, id := range queryIDs(r) {
		_ = lp.Unstar(ctx, userID, id)
	}
	writeOK(w, r, map[string]any{})
}

func (s *Server) handleScrobble(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	lp := s.library()
	if lp == nil {
		writeError(w, r, 70, "scrobble unavailable")
		return
	}
	ids := queryIDs(r)
	times := r.URL.Query()["time"]
	submission := strings.EqualFold(r.URL.Query().Get("submission"), "true")
	for i, id := range ids {
		track, _, err := s.provider.Track(ctx, userID, id)
		if err != nil {
			continue
		}
		playedAt := time.Now()
		if i < len(times) {
			if ms, err := strconv.ParseInt(times[i], 10, 64); err == nil && ms > 0 {
				playedAt = time.UnixMilli(ms)
			}
		}
		if submission {
			_ = lp.Scrobble(ctx, userID, track, playedAt)
		}
	}
	writeOK(w, r, map[string]any{})
}

func (s *Server) handleGetGenres(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	lp := s.library()
	if lp == nil {
		writeError(w, r, 70, "genres unavailable")
		return
	}
	genres, err := lp.ListGenres(ctx, userID)
	if err != nil {
		writeError(w, r, 0, err.Error())
		return
	}
	out := make([]map[string]any, len(genres))
	for i, genre := range genres {
		out[i] = map[string]any{
			"name":      genre.Name,
			"songCount": genre.SongCount,
		}
	}
	writeOK(w, r, map[string]any{
		"genres": map[string]any{"genre": out},
	})
}

func queryIDs(r *http.Request) []string {
	if values := r.URL.Query()["id"]; len(values) > 0 {
		return values
	}
	if single := strings.TrimSpace(r.URL.Query().Get("id")); single != "" {
		return []string{single}
	}
	return nil
}

func playlistView(item PlaylistSummary) map[string]any {
	return map[string]any{
		"id":        item.ID,
		"name":      item.Name,
		"songCount": item.SongCount,
		"duration":  item.Duration,
		"public":    item.Public,
		"created":   formatSubsonicTime(item.Created),
		"changed":   formatSubsonicTime(item.Changed),
	}
}

func formatSubsonicTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format("2006-01-02T15:04:05.000Z")
}

func streamWithTranscode(
	w http.ResponseWriter,
	r *http.Request,
	ctx context.Context,
	inputPath string,
	format string,
) bool {
	maxBitRate := strings.TrimSpace(r.URL.Query().Get("maxBitRate"))
	if !transcode.ShouldTranscode(maxBitRate, format) {
		return false
	}
	rate, _ := strconv.Atoi(maxBitRate)
	if err := transcode.ServeMP3(ctx, inputPath, rate, w, r); err != nil {
		return false
	}
	return true
}
