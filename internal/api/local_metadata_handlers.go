// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"melovian/internal/httputil"
	"melovian/internal/localmusic"
	"melovian/internal/metadata"
	"melovian/internal/metaloader"
	"melovian/internal/store"
)

func (s *Server) registerLocalMetadataRoutes() {
	s.mux.HandleFunc("GET /api/local-music/metadata/summary", s.handleLocalMetadataSummary)
	s.mux.HandleFunc("GET /api/local-music/metadata/tracks", s.handleLocalMetadataSearch)
	s.mux.HandleFunc("GET /api/local-music/metadata/tracks/{id}", s.handleLocalMetadataTrack)
	s.mux.HandleFunc("PATCH /api/local-music/metadata/tracks/{id}", s.handleLocalMetadataUpdate)
	s.mux.HandleFunc("GET /api/local-music/metadata/lookup", s.handleLocalMetadataLookup)
	s.mux.HandleFunc("POST /api/local-music/metadata/tracks/{id}/autofix", s.handleLocalMetadataAutofix)
	s.mux.HandleFunc("POST /api/local-music/metadata/tracks/{id}/autofix-album", s.handleLocalMetadataAutofixAlbum)
	s.mux.HandleFunc("GET /api/local-music/metadata/tracks/{id}/suggestions", s.handleLocalMetadataSuggestions)
	s.mux.HandleFunc("POST /api/local-music/metadata/autofix-batch", s.handleLocalMetadataAutofixBatch)
}

type localMetadataSummaryResponse struct {
	UnknownArtist int `json:"unknownArtist"`
	UnknownAlbum  int `json:"unknownAlbum"`
	MissingTitle  int `json:"missingTitle"`
	Any           int `json:"any"`
	TotalTracks   int `json:"totalTracks"`
}

type localMetadataTrackView struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Artist      string   `json:"artist"`
	Album       string   `json:"album"`
	AlbumArtist string   `json:"albumArtist"`
	TrackNum    int      `json:"trackNum"`
	DiscNum     int      `json:"discNum"`
	Year        int      `json:"year"`
	Genre       string   `json:"genre"`
	DurationMs  int      `json:"durationMs"`
	Format      string   `json:"format"`
	RelPath     string   `json:"relPath"`
	Issues      []string `json:"issues"`
	CoverArt    string   `json:"coverArt"`
}

type localMetadataSearchResponse struct {
	Tracks []localMetadataTrackView `json:"tracks"`
	Total  int                      `json:"total"`
}

type localMetadataLookupResponse struct {
	Matches []metadata.LookupMatch `json:"matches"`
}

type localMetadataUpdateRequest struct {
	Title       *string `json:"title"`
	Artist      *string `json:"artist"`
	Album       *string `json:"album"`
	AlbumArtist *string `json:"albumArtist"`
	TrackNum    *int    `json:"trackNum"`
	DiscNum     *int    `json:"discNum"`
	Year        *int    `json:"year"`
	Genre       *string `json:"genre"`
}

type localMetadataAutofixRequest struct {
	Match metadata.LookupMatch `json:"match"`
}

type localMetadataAlbumAutofixResponse struct {
	Updated int                      `json:"updated"`
	Failed  int                      `json:"failed"`
	Tracks  []localMetadataTrackView `json:"tracks"`
}

type localMetadataSuggestionsResponse struct {
	Filename metadata.FilenameSuggestion `json:"filename"`
}

type localMetadataBatchRequest struct {
	TrackIDs []string `json:"trackIds"`
	Source   string   `json:"source"`
}

type localMetadataBatchResult struct {
	TrackID string                  `json:"trackId"`
	OK      bool                    `json:"ok"`
	Error   string                  `json:"error,omitempty"`
	Track   *localMetadataTrackView `json:"track,omitempty"`
}

type localMetadataBatchResponse struct {
	Results []localMetadataBatchResult `json:"results"`
}

func (s *Server) activeLocalLibrary(r *http.Request) (store.LocalLibrary, error) {
	userID := UserIDFromContext(r.Context())
	return s.localLibraries.GetActiveForUser(userID)
}

func (s *Server) handleLocalMetadataSummary(w http.ResponseWriter, r *http.Request) {
	lib, err := s.activeLocalLibrary(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	unknownArtist, unknownAlbum, missingTitle, any, err := s.localTracks.CountMetadataIssues(lib.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	present, _, _, err := s.localTracks.CountByStatus(lib.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, localMetadataSummaryResponse{
		UnknownArtist: unknownArtist,
		UnknownAlbum:  unknownAlbum,
		MissingTitle:  missingTitle,
		Any:           any,
		TotalTracks:   present,
	})
}

func (s *Server) handleLocalMetadataSearch(w http.ResponseWriter, r *http.Request) {
	lib, err := s.activeLocalLibrary(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	tracks, total, err := s.localTracks.SearchMetadata(store.MetadataSearchQuery{
		LibraryID: lib.ID,
		Query:     r.URL.Query().Get("q"),
		Issue:     r.URL.Query().Get("issue"),
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	views := make([]localMetadataTrackView, len(tracks))
	for i, track := range tracks {
		views[i] = localMetadataTrackViewFrom(track)
	}
	httputil.WriteJSON(w, http.StatusOK, localMetadataSearchResponse{
		Tracks: views,
		Total:  total,
	})
}

func (s *Server) handleLocalMetadataTrack(w http.ResponseWriter, r *http.Request) {
	lib, err := s.activeLocalLibrary(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	trackID := r.PathValue("id")
	track, err := s.localTracks.Get(lib.ID, trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if track.Status != store.TrackStatusPresent || track.DuplicateOf != "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, localMetadataTrackViewFrom(track))
}

func (s *Server) handleLocalMetadataUpdate(w http.ResponseWriter, r *http.Request) {
	lib, err := s.activeLocalLibrary(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	trackID := r.PathValue("id")
	track, err := s.localTracks.Get(lib.ID, trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if track.Status != store.TrackStatusPresent || track.DuplicateOf != "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var req localMetadataUpdateRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	meta := metaloader.TrackMetadata{
		Title:       track.Title,
		Artist:      track.Artist,
		Album:       track.Album,
		AlbumArtist: track.AlbumArtist,
		TrackNum:    track.TrackNum,
		DiscNum:     track.DiscNum,
		Year:        track.Year,
		Genre:       track.Genre,
	}
	if req.Title != nil {
		meta.Title = strings.TrimSpace(*req.Title)
	}
	if req.Artist != nil {
		meta.Artist = strings.TrimSpace(*req.Artist)
	}
	if req.Album != nil {
		meta.Album = strings.TrimSpace(*req.Album)
	}
	if req.AlbumArtist != nil {
		meta.AlbumArtist = strings.TrimSpace(*req.AlbumArtist)
	}
	if req.TrackNum != nil {
		meta.TrackNum = *req.TrackNum
	}
	if req.DiscNum != nil {
		meta.DiscNum = *req.DiscNum
	}
	if req.Year != nil {
		meta.Year = *req.Year
	}
	if req.Genre != nil {
		meta.Genre = strings.TrimSpace(*req.Genre)
	}

	path, err := localmusic.ResolveTrackPath(lib.Path, track.AbsPath)
	if err != nil {
		http.Error(w, "invalid track path", http.StatusForbidden)
		return
	}
	if err := metaloader.WriteTrackMetadata(path, meta); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	updated, err := s.libraryScanner.RescanFile(lib.ID, trackID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.invalidateLocalLibraryData(lib.ID)
	httputil.WriteJSON(w, http.StatusOK, localMetadataTrackViewFrom(updated))
}

func (s *Server) handleLocalMetadataLookup(w http.ResponseWriter, r *http.Request) {
	if _, err := s.activeLocalLibrary(r); err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	query := metadata.LookupQuery{
		Query:  strings.TrimSpace(r.URL.Query().Get("q")),
		Artist: strings.TrimSpace(r.URL.Query().Get("artist")),
		Title:  strings.TrimSpace(r.URL.Query().Get("title")),
		Album:  strings.TrimSpace(r.URL.Query().Get("album")),
	}
	trackID := strings.TrimSpace(r.URL.Query().Get("trackId"))
	if trackID != "" {
		lib, libErr := s.activeLocalLibrary(r)
		if libErr != nil {
			s.writeLocalMusicError(w, libErr)
			return
		}
		track, err := s.localTracks.Get(lib.ID, trackID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if query.Query == "" {
			query.Query = strings.TrimSpace(strings.Join([]string{track.Artist, track.Title, track.Album}, " "))
		}
		if query.Artist == "" {
			query.Artist = track.Artist
		}
		if query.Title == "" {
			query.Title = track.Title
		}
		if query.Album == "" {
			query.Album = track.Album
		}
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	matches, err := metadata.Lookup(ctx, source, query, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, localMetadataLookupResponse{Matches: matches})
}

func (s *Server) handleLocalMetadataAutofix(w http.ResponseWriter, r *http.Request) {
	lib, err := s.activeLocalLibrary(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	trackID := r.PathValue("id")
	track, err := s.localTracks.Get(lib.ID, trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if track.Status != store.TrackStatusPresent || track.DuplicateOf != "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var req localMetadataAutofixRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	match := req.Match
	if strings.TrimSpace(match.Title) == "" {
		http.Error(w, "match title required", http.StatusBadRequest)
		return
	}

	updated, err := s.writeTrackMetadataMatch(lib, track, match)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.invalidateLocalLibraryData(lib.ID)
	httputil.WriteJSON(w, http.StatusOK, localMetadataTrackViewFrom(updated))
}

func (s *Server) handleLocalMetadataAutofixAlbum(w http.ResponseWriter, r *http.Request) {
	lib, err := s.activeLocalLibrary(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	trackID := r.PathValue("id")
	anchor, err := s.localTracks.Get(lib.ID, trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if anchor.Status != store.TrackStatusPresent || anchor.DuplicateOf != "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var req localMetadataAutofixRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	match := req.Match
	if strings.TrimSpace(match.Album) == "" && strings.TrimSpace(match.Artist) == "" {
		http.Error(w, "match album or artist required", http.StatusBadRequest)
		return
	}

	peers, err := s.localTracks.ListByAlbumArtist(lib.ID, anchor.Album, anchor.Artist)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(peers) > 100 {
		http.Error(w, "too many tracks in album", http.StatusBadRequest)
		return
	}

	updated := make([]localMetadataTrackView, 0, len(peers))
	var failed int
	for _, track := range peers {
		merged, writeErr := s.writeTrackMetadataAlbumFields(lib, track, match)
		if writeErr != nil {
			failed++
			continue
		}
		updated = append(updated, localMetadataTrackViewFrom(merged))
	}
	if len(updated) == 0 {
		http.Error(w, "no tracks updated", http.StatusBadRequest)
		return
	}
	s.invalidateLocalLibraryData(lib.ID)
	httputil.WriteJSON(w, http.StatusOK, localMetadataAlbumAutofixResponse{
		Updated: len(updated),
		Failed:  failed,
		Tracks:  updated,
	})
}

func (s *Server) handleLocalMetadataSuggestions(w http.ResponseWriter, r *http.Request) {
	lib, err := s.activeLocalLibrary(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	trackID := r.PathValue("id")
	track, err := s.localTracks.Get(lib.ID, trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if track.Status != store.TrackStatusPresent || track.DuplicateOf != "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, localMetadataSuggestionsResponse{
		Filename: metadata.ParseFilenameSuggestion(track.RelPath),
	})
}

func (s *Server) handleLocalMetadataAutofixBatch(w http.ResponseWriter, r *http.Request) {
	lib, err := s.activeLocalLibrary(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}

	var req localMetadataBatchRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if len(req.TrackIDs) == 0 {
		http.Error(w, "trackIds required", http.StatusBadRequest)
		return
	}
	if len(req.TrackIDs) > 25 {
		http.Error(w, "too many tracks", http.StatusBadRequest)
		return
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "filename"
	}

	results := make([]localMetadataBatchResult, 0, len(req.TrackIDs))
	for _, trackID := range req.TrackIDs {
		result := localMetadataBatchResult{TrackID: trackID}
		track, err := s.localTracks.Get(lib.ID, trackID)
		if err != nil || track.Status != store.TrackStatusPresent || track.DuplicateOf != "" {
			result.Error = "not found"
			results = append(results, result)
			continue
		}

		match, matchErr := s.metadataMatchForTrack(r.Context(), track, source)
		if matchErr != nil {
			result.Error = matchErr.Error()
			results = append(results, result)
			continue
		}
		if strings.TrimSpace(match.Title) == "" {
			result.Error = "no match found"
			results = append(results, result)
			continue
		}

		updated, writeErr := s.writeTrackMetadataMatch(lib, track, match)
		if writeErr != nil {
			result.Error = writeErr.Error()
			results = append(results, result)
			continue
		}
		view := localMetadataTrackViewFrom(updated)
		result.OK = true
		result.Track = &view
		results = append(results, result)
	}
	s.invalidateLocalLibraryData(lib.ID)
	httputil.WriteJSON(w, http.StatusOK, localMetadataBatchResponse{Results: results})
}

func (s *Server) metadataMatchForTrack(ctx context.Context, track store.LocalTrack, source string) (metadata.LookupMatch, error) {
	if source == "filename" {
		return metadata.SuggestionToMatch(metadata.ParseFilenameSuggestion(track.RelPath)), nil
	}
	if source == "lookup" {
		source = "itunes"
	}
	query := metadata.LookupQuery{
		Query:  strings.TrimSpace(strings.Join([]string{track.Artist, track.Title, track.Album}, " ")),
		Artist: track.Artist,
		Title:  track.Title,
		Album:  track.Album,
	}
	matches, err := metadata.Lookup(ctx, source, query, 1)
	if err != nil {
		return metadata.LookupMatch{}, err
	}
	if len(matches) == 0 {
		return metadata.LookupMatch{}, nil
	}
	return matches[0], nil
}

func (s *Server) writeTrackMetadataAlbumFields(
	lib store.LocalLibrary,
	track store.LocalTrack,
	match metadata.LookupMatch,
) (store.LocalTrack, error) {
	year := track.Year
	if match.Year > 0 {
		year = match.Year
	}
	genre := track.Genre
	if strings.TrimSpace(match.Genre) != "" {
		genre = match.Genre
	}
	meta := metaloader.TrackMetadata{
		Title:       track.Title,
		Artist:      metadata.FirstNonEmpty(match.Artist, track.Artist),
		Album:       metadata.FirstNonEmpty(match.Album, track.Album),
		AlbumArtist: metadata.FirstNonEmpty(match.AlbumArtist, match.Artist, track.AlbumArtist),
		TrackNum:    track.TrackNum,
		DiscNum:     track.DiscNum,
		Year:        year,
		Genre:       genre,
	}
	path, err := localmusic.ResolveTrackPath(lib.Path, track.AbsPath)
	if err != nil {
		return store.LocalTrack{}, err
	}
	if err := metaloader.WriteTrackMetadata(path, meta); err != nil {
		return store.LocalTrack{}, err
	}
	return s.libraryScanner.RescanFile(lib.ID, track.ID)
}

func (s *Server) writeTrackMetadataMatch(lib store.LocalLibrary, track store.LocalTrack, match metadata.LookupMatch) (store.LocalTrack, error) {
	meta := metaloader.TrackMetadata{
		Title:       match.Title,
		Artist:      match.Artist,
		Album:       match.Album,
		AlbumArtist: metadata.FirstNonEmpty(match.AlbumArtist, match.Artist),
		TrackNum:    match.TrackNum,
		Year:        match.Year,
		Genre:       match.Genre,
	}
	path, err := localmusic.ResolveTrackPath(lib.Path, track.AbsPath)
	if err != nil {
		return store.LocalTrack{}, err
	}
	if err := metaloader.WriteTrackMetadata(path, meta); err != nil {
		return store.LocalTrack{}, err
	}
	return s.libraryScanner.RescanFile(lib.ID, track.ID)
}

func localMetadataTrackViewFrom(track store.LocalTrack) localMetadataTrackView {
	return localMetadataTrackView{
		ID:          track.ID,
		Title:       track.Title,
		Artist:      track.Artist,
		Album:       track.Album,
		AlbumArtist: track.AlbumArtist,
		TrackNum:    track.TrackNum,
		DiscNum:     track.DiscNum,
		Year:        track.Year,
		Genre:       track.Genre,
		DurationMs:  track.DurationMs,
		Format:      track.Format,
		RelPath:     filepath.ToSlash(track.RelPath),
		Issues:      metadata.TrackIssues(track),
		CoverArt:    track.ID,
	}
}
