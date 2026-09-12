// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package library

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

	"melovian/internal/api/apishared"
	"melovian/internal/consts"
	"melovian/internal/httputil"
	"melovian/internal/localmusic"
	"melovian/internal/metadata"
	"melovian/internal/metaloader"
	"melovian/internal/store"
)

func (h *Handler) registerLocalMetadataRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/local-music/metadata/summary", h.handleLocalMetadataSummary)
	mux.HandleFunc("GET /api/local-music/metadata/tracks", h.handleLocalMetadataSearch)
	mux.HandleFunc("GET /api/local-music/metadata/tracks/{id}", h.handleLocalMetadataTrack)
	mux.HandleFunc("PATCH /api/local-music/metadata/tracks/{id}", h.handleLocalMetadataUpdate)
	mux.HandleFunc("GET /api/local-music/metadata/lookup", h.handleLocalMetadataLookup)
	mux.HandleFunc("POST /api/local-music/metadata/tracks/{id}/autofix", h.handleLocalMetadataAutofix)
	mux.HandleFunc("POST /api/local-music/metadata/tracks/{id}/autofix-album", h.handleLocalMetadataAutofixAlbum)
	mux.HandleFunc("GET /api/local-music/metadata/tracks/{id}/suggestions", h.handleLocalMetadataSuggestions)
	mux.HandleFunc("POST /api/local-music/metadata/autofix-batch", h.handleLocalMetadataAutofixBatch)
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

func (h *Handler) activeLocalLibrary(r *http.Request) (store.LocalLibrary, error) {
	userID := apishared.UserIDFromContext(r.Context())
	return h.localLibraries.GetActiveForUser(userID)
}

func (h *Handler) handleLocalMetadataSummary(w http.ResponseWriter, r *http.Request) {
	lib, err := h.activeLocalLibrary(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	unknownArtist, unknownAlbum, missingTitle, any, err := h.localTracks.CountMetadataIssues(lib.ID)
	if err != nil {
		httputil.WriteInternalError(w, r, "handleLocalMetadataSummary", err)
		return
	}
	present, _, _, err := h.localTracks.CountByStatus(lib.ID)
	if err != nil {
		httputil.WriteInternalError(w, r, "handleLocalMetadataSummary", err)
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

func (h *Handler) handleLocalMetadataSearch(w http.ResponseWriter, r *http.Request) {
	lib, err := h.activeLocalLibrary(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	tracks, total, err := h.localTracks.SearchMetadata(store.MetadataSearchQuery{
		LibraryID: lib.ID,
		Query:     r.URL.Query().Get("q"),
		Issue:     r.URL.Query().Get("issue"),
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		httputil.WriteInternalError(w, r, "handleLocalMetadataSearch", err)
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

func (h *Handler) handleLocalMetadataTrack(w http.ResponseWriter, r *http.Request) {
	lib, err := h.activeLocalLibrary(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	trackID := r.PathValue("id")
	track, err := h.localTracks.Get(lib.ID, trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleLocalMetadataTrack", err)
		return
	}
	if track.Status != store.TrackStatusPresent || track.DuplicateOf != "" {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, localMetadataTrackViewFrom(track))
}

func (h *Handler) handleLocalMetadataUpdate(w http.ResponseWriter, r *http.Request) {
	lib, err := h.activeLocalLibrary(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	trackID := r.PathValue("id")
	track, err := h.localTracks.Get(lib.ID, trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleLocalMetadataUpdate", err)
		return
	}
	if track.Status != store.TrackStatusPresent || track.DuplicateOf != "" {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	var req localMetadataUpdateRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid json")
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
		httputil.WriteError(w, http.StatusForbidden, "invalid_track_path", "invalid track path")
		return
	}
	if err := metaloader.WriteTrackMetadata(path, meta); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	updated, err := h.libraryScanner.RescanFile(lib.ID, trackID)
	if err != nil {
		httputil.WriteInternalError(w, r, "handleLocalMetadataUpdate", err)
		return
	}
	h.invalidateLocalLibraryData(lib.ID)
	httputil.WriteJSON(w, http.StatusOK, localMetadataTrackViewFrom(updated))
}

func (h *Handler) handleLocalMetadataLookup(w http.ResponseWriter, r *http.Request) {
	if _, err := h.activeLocalLibrary(r); err != nil {
		h.WriteLocalMusicError(w, r, err)
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
		lib, libErr := h.activeLocalLibrary(r)
		if libErr != nil {
			h.WriteLocalMusicError(w, r, libErr)
			return
		}
		track, err := h.localTracks.Get(lib.ID, trackID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
				return
			}
			httputil.WriteInternalError(w, r, "handleLocalMetadataLookup", err)
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
	ctx, cancel := context.WithTimeout(r.Context(), consts.MetadataFetchTimeout)
	defer cancel()
	matches, err := metadata.Lookup(ctx, source, query, limit)
	if err != nil {
		httputil.WriteInternalError(w, r, "handleLocalMetadataLookup", err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, localMetadataLookupResponse{Matches: matches})
}

func (h *Handler) handleLocalMetadataAutofix(w http.ResponseWriter, r *http.Request) {
	lib, err := h.activeLocalLibrary(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	trackID := r.PathValue("id")
	track, err := h.localTracks.Get(lib.ID, trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleLocalMetadataAutofix", err)
		return
	}
	if track.Status != store.TrackStatusPresent || track.DuplicateOf != "" {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	var req localMetadataAutofixRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	match := req.Match
	if strings.TrimSpace(match.Title) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "match_title_required", "match title required")
		return
	}

	updated, err := h.writeTrackMetadataMatch(lib, track, match)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	h.invalidateLocalLibraryData(lib.ID)
	httputil.WriteJSON(w, http.StatusOK, localMetadataTrackViewFrom(updated))
}

func (h *Handler) handleLocalMetadataAutofixAlbum(w http.ResponseWriter, r *http.Request) {
	lib, err := h.activeLocalLibrary(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	trackID := r.PathValue("id")
	anchor, err := h.localTracks.Get(lib.ID, trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleLocalMetadataAutofixAlbum", err)
		return
	}
	if anchor.Status != store.TrackStatusPresent || anchor.DuplicateOf != "" {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	var req localMetadataAutofixRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	match := req.Match
	if strings.TrimSpace(match.Album) == "" && strings.TrimSpace(match.Artist) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "match_album_or_artist_required", "match album or artist required")
		return
	}

	peers, err := h.localTracks.ListByAlbumArtist(lib.ID, anchor.Album, anchor.Artist)
	if err != nil {
		httputil.WriteInternalError(w, r, "handleLocalMetadataAutofixAlbum", err)
		return
	}
	if len(peers) > 100 {
		httputil.WriteError(w, http.StatusBadRequest, "too_many_tracks_in_album", "too many tracks in album")
		return
	}

	updated := make([]localMetadataTrackView, 0, len(peers))
	var failed int
	for _, track := range peers {
		merged, writeErr := h.writeTrackMetadataAlbumFields(lib, track, match)
		if writeErr != nil {
			failed++
			continue
		}
		updated = append(updated, localMetadataTrackViewFrom(merged))
	}
	if len(updated) == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "no_tracks_updated", "no tracks updated")
		return
	}
	h.invalidateLocalLibraryData(lib.ID)
	httputil.WriteJSON(w, http.StatusOK, localMetadataAlbumAutofixResponse{
		Updated: len(updated),
		Failed:  failed,
		Tracks:  updated,
	})
}

func (h *Handler) handleLocalMetadataSuggestions(w http.ResponseWriter, r *http.Request) {
	lib, err := h.activeLocalLibrary(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	trackID := r.PathValue("id")
	track, err := h.localTracks.Get(lib.ID, trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleLocalMetadataSuggestions", err)
		return
	}
	if track.Status != store.TrackStatusPresent || track.DuplicateOf != "" {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, localMetadataSuggestionsResponse{
		Filename: metadata.ParseFilenameSuggestion(track.RelPath),
	})
}

func (h *Handler) handleLocalMetadataAutofixBatch(w http.ResponseWriter, r *http.Request) {
	lib, err := h.activeLocalLibrary(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}

	var req localMetadataBatchRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	if len(req.TrackIDs) == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "trackids_required", "trackIds required")
		return
	}
	if len(req.TrackIDs) > 25 {
		httputil.WriteError(w, http.StatusBadRequest, "too_many_tracks", "too many tracks")
		return
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "filename"
	}

	results := make([]localMetadataBatchResult, 0, len(req.TrackIDs))
	for _, trackID := range req.TrackIDs {
		result := localMetadataBatchResult{TrackID: trackID}
		track, err := h.localTracks.Get(lib.ID, trackID)
		if err != nil || track.Status != store.TrackStatusPresent || track.DuplicateOf != "" {
			result.Error = "not found"
			results = append(results, result)
			continue
		}

		match, matchErr := h.metadataMatchForTrack(r.Context(), track, source)
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

		updated, writeErr := h.writeTrackMetadataMatch(lib, track, match)
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
	h.invalidateLocalLibraryData(lib.ID)
	httputil.WriteJSON(w, http.StatusOK, localMetadataBatchResponse{Results: results})
}

func (h *Handler) metadataMatchForTrack(ctx context.Context, track store.LocalTrack, source string) (metadata.LookupMatch, error) {
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

func (h *Handler) writeTrackMetadataAlbumFields(
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
	return h.libraryScanner.RescanFile(lib.ID, track.ID)
}

func (h *Handler) writeTrackMetadataMatch(lib store.LocalLibrary, track store.LocalTrack, match metadata.LookupMatch) (store.LocalTrack, error) {
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
	return h.libraryScanner.RescanFile(lib.ID, track.ID)
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
