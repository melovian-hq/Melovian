// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package library

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"melovian/internal/api/apishared"
	"melovian/internal/consts"
	"melovian/internal/httputil"
	"melovian/internal/localmusic"
	"melovian/internal/metadata"
	"melovian/internal/store"
)

func (h *Handler) registerLocalMusicRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/local-music/artists", h.handleLocalMusicArtists)
	mux.HandleFunc("GET /api/local-music/artists/{id}", h.handleLocalMusicArtist)
	mux.HandleFunc("GET /api/local-music/albums", h.handleLocalMusicAlbumList)
	mux.HandleFunc("GET /api/local-music/albums/{id}", h.handleLocalMusicAlbum)
	mux.HandleFunc("GET /api/local-music/search", h.handleLocalMusicSearch)
	mux.HandleFunc("GET /api/local-music/songs/{id}", h.handleLocalMusicSong)
	mux.HandleFunc("GET /api/local-music/randomSongs", h.handleLocalMusicRandomSongs)
	mux.HandleFunc("GET /api/local-music/genres", h.handleLocalMusicGenres)
	mux.HandleFunc("GET /api/local-music/songsByGenre", h.handleLocalMusicSongsByGenre)
	mux.HandleFunc("GET /api/local-music/starred", h.handleLocalMusicStarred)
	mux.HandleFunc("GET /api/local-music/songs/{id}/similar", h.handleLocalMusicSimilar)
	mux.HandleFunc("GET /api/local-music/tracks/{id}/stream", h.handleLocalMusicStream)
	mux.HandleFunc("GET /api/local-music/cover/{id}", h.handleLocalMusicCover)
}

func (h *Handler) invalidateLocalLibraryData(libraryID string) {
	h.catalogCache.Invalidate(libraryID)
	h.coverCache.InvalidateLibrary(libraryID)
	h.libraryScanner.ForgetLibrary(libraryID)
}

func (h *Handler) activeLocalCatalog(r *http.Request) (localmusic.Catalog, store.LocalLibrary, error) {
	userID := apishared.UserIDFromContext(r.Context())
	catalog, err := h.LocalCatalogForUser(userID)
	if err != nil {
		return localmusic.Catalog{}, store.LocalLibrary{}, err
	}
	lib, err := h.localLibraries.GetActiveForUser(userID)
	if err != nil {
		return localmusic.Catalog{}, store.LocalLibrary{}, err
	}
	return catalog, lib, nil
}

func (h *Handler) handleLocalMusicArtists(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := h.activeLocalCatalog(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, localArtistsResponse{
		Artists: localArtistViewsFrom(catalog.Artists),
	})
}

func (h *Handler) handleLocalMusicArtist(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := h.activeLocalCatalog(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	artist, albums, ok := catalog.Artist(r.PathValue("id"))
	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, localArtistDetailResponse{
		Artist: localArtistViewFrom(artist),
		Albums: localAlbumViewsFrom(albums),
	})
}

func (h *Handler) handleLocalMusicAlbumList(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := h.activeLocalCatalog(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	listType := r.URL.Query().Get("type")
	if listType == "" {
		listType = "newest"
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	httputil.WriteJSON(w, http.StatusOK, localAlbumsResponse{
		Albums: localAlbumViewsFrom(catalog.AlbumList(listType, size, offset)),
	})
}

func (h *Handler) handleLocalMusicAlbum(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := h.activeLocalCatalog(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	album, songs, ok := catalog.Album(r.PathValue("id"))
	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, localAlbumDetailResponse{
		Album: localAlbumViewFrom(album),
		Songs: h.decorateLocalSongViews(r, LocalSongViewsFrom(songs)),
	})
}

func (h *Handler) handleLocalMusicSearch(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := h.activeLocalCatalog(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	artists, albums, songs := catalog.Search(r.URL.Query().Get("q"), limit)
	httputil.WriteJSON(w, http.StatusOK, localSearchResponse{
		Artists: localArtistViewsFrom(artists),
		Albums:  localAlbumViewsFrom(albums),
		Songs:   h.decorateLocalSongViews(r, LocalSongViewsFrom(songs)),
	})
}

func (h *Handler) handleLocalMusicSong(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := h.activeLocalCatalog(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	song, ok := catalog.Song(r.PathValue("id"))
	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	views := h.decorateLocalSongViews(r, []LocalSongView{LocalSongViewFrom(song)})
	httputil.WriteJSON(w, http.StatusOK, views[0])
}

func (h *Handler) handleLocalMusicRandomSongs(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := h.activeLocalCatalog(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	httputil.WriteJSON(w, http.StatusOK, LocalSongsResponse{
		Songs: h.decorateLocalSongViews(r, LocalSongViewsFrom(catalog.RandomSongs(size))),
	})
}

func (h *Handler) handleLocalMusicGenres(w http.ResponseWriter, r *http.Request) {
	_, lib, err := h.activeLocalCatalog(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	counts, err := h.localTracks.ListGenreCounts([]string{lib.ID})
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load genres")
		return
	}
	genres := make([]map[string]any, 0, len(counts))
	for _, item := range counts {
		genres = append(genres, map[string]any{
			"name":       item.Name,
			"songCount":  item.Count,
			"albumCount": 0,
		})
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"genres": genres})
}

func (h *Handler) handleLocalMusicSongsByGenre(w http.ResponseWriter, r *http.Request) {
	_, lib, err := h.activeLocalCatalog(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	genre := strings.TrimSpace(r.URL.Query().Get("genre"))
	if genre == "" {
		httputil.WriteJSON(w, http.StatusOK, LocalSongsResponse{Songs: []LocalSongView{}})
		return
	}
	count, _ := strconv.Atoi(r.URL.Query().Get("count"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	tracks, err := h.localTracks.ListByGenre(lib.ID, genre, count, offset)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load songs")
		return
	}
	views := make([]LocalSongView, 0, len(tracks))
	for _, track := range tracks {
		views = append(views, LocalSongViewFromTrack(track))
	}
	httputil.WriteJSON(w, http.StatusOK, LocalSongsResponse{
		Songs: h.decorateLocalSongViews(r, views),
	})
}

func (h *Handler) handleLocalMusicStarred(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := h.activeLocalCatalog(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	userID := apishared.ResolveProgressUserID(r.Context())
	favorites, err := h.listen.ListFavorites(userID, 2000)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load favorites")
		return
	}
	songs := make([]LocalSongView, 0, len(favorites))
	for _, fav := range favorites {
		if !strings.HasPrefix(fav.TrackID, "trk_") {
			continue
		}
		if song, ok := catalog.Song(fav.TrackID); ok {
			view := LocalSongViewFrom(song)
			view.Starred = fav.FavoritedAt.UTC().Format(time.RFC3339)
			songs = append(songs, view)
			continue
		}
		// Track is favorited but no longer in the catalog (missing file).
		// Still surface it so the favorites page does not lose the entry.
		songs = append(songs, LocalSongView{
			ID:       fav.TrackID,
			Title:    fav.TrackTitle,
			Artist:   fav.ArtistName,
			Album:    fav.AlbumTitle,
			AlbumID:  fav.AlbumID,
			Duration: fav.DurationMs / 1000,
			CoverArt: fav.CoverArtID,
			Starred:  fav.FavoritedAt.UTC().Format(time.RFC3339),
		})
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"songs":   h.decorateLocalSongViews(r, songs),
		"albums":  []localAlbumView{},
		"artists": []localArtistView{},
	})
}

func (h *Handler) handleLocalMusicSimilar(w http.ResponseWriter, r *http.Request) {
	userID := apishared.UserIDFromContext(r.Context())
	trackID := r.PathValue("id")
	track, err := h.localTracks.GetByID(trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleLocalMusicSimilar", err)
		return
	}
	if _, err := h.localLibraries.GetForUser(userID, track.LibraryID); err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	count, _ := strconv.Atoi(r.URL.Query().Get("count"))
	tracks, err := h.localTracks.ListSimilarTracks(track.LibraryID, track.Genre, track.Artist, trackID, count)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load similar songs")
		return
	}
	views := make([]LocalSongView, 0, len(tracks))
	for _, item := range tracks {
		views = append(views, LocalSongViewFromTrack(item))
	}
	httputil.WriteJSON(w, http.StatusOK, LocalSongsResponse{
		Songs: h.decorateLocalSongViews(r, views),
	})
}

func (h *Handler) handleLocalMusicStream(w http.ResponseWriter, r *http.Request) {
	userID := apishared.UserIDFromContext(r.Context())
	trackID := r.PathValue("id")
	track, err := h.localTracks.GetByID(trackID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		httputil.WriteInternalError(w, r, "handleLocalMusicStream", err)
		return
	}
	lib, err := h.localLibraries.GetForUser(userID, track.LibraryID)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	if track.Status != store.TrackStatusPresent || track.DuplicateOf != "" {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	path, err := localmusic.ResolveTrackPath(lib.Path, track.AbsPath)
	if err != nil {
		httputil.WriteError(w, http.StatusForbidden, "invalid_track_path", "invalid track path")
		return
	}

	file, err := os.Open(path) //#nosec G304 -- path validated against library root
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "open_track", "open track")
		return
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "stat track")
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", localmusic.ContentTypeForFormat(track.Format))
	http.ServeContent(w, r, filepath.Base(path), info.ModTime(), file)
}

func (h *Handler) handleLocalMusicCover(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := h.activeLocalCatalog(r)
	if err != nil {
		h.WriteLocalMusicError(w, r, err)
		return
	}
	id := r.PathValue("id")
	data, contentType, ok := h.localCoverData(r, catalog, id)
	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, "cover", time.Now(), bytes.NewReader(data))
}

// decorateLocalSongViews fills play counts and last-played timestamps from the
// user's listen history. Best effort: failures leave the fields empty.
func (h *Handler) decorateLocalSongViews(r *http.Request, views []LocalSongView) []LocalSongView {
	if len(views) == 0 {
		return views
	}
	userID := apishared.ResolveProgressUserID(r.Context())
	ids := make([]string, 0, len(views))
	for _, view := range views {
		ids = append(ids, view.ID)
	}
	progress, err := h.listen.Batch(userID, ids)
	if err != nil {
		return views
	}
	for i := range views {
		entry, ok := progress[views[i].ID]
		if !ok {
			continue
		}
		views[i].PlayCount = entry.PlayCount
		if !entry.LastPlayedAt.IsZero() {
			views[i].LastPlayedAt = entry.LastPlayedAt.UTC().Format(time.RFC3339)
		}
	}
	return views
}

func (h *Handler) WriteLocalMusicError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		httputil.WriteError(w, http.StatusNotFound, "no_active_local_library", "no active local library")
		return
	}
	httputil.WriteInternalError(w, r, "local music", err)
}

func (h *Handler) localCoverData(r *http.Request, catalog localmusic.Catalog, id string) ([]byte, string, bool) {
	trackID, ok := catalog.CoverTrackID(id)
	if !ok {
		return nil, "", false
	}
	track, err := h.localTracks.GetByID(trackID)
	if err != nil {
		return nil, "", false
	}
	userID := apishared.UserIDFromContext(r.Context())
	lib, err := h.localLibraries.GetForUser(userID, track.LibraryID)
	if err != nil {
		return nil, "", false
	}
	return h.coverBytesFromTrackFileCtx(r.Context(), lib, trackID)
}

func (h *Handler) CoverBytesFromTrackFile(lib store.LocalLibrary, trackID string) ([]byte, string, bool) {
	return h.coverBytesFromTrackFileCtx(context.Background(), lib, trackID)
}

func (h *Handler) coverBytesFromTrackFileCtx(ctx context.Context, lib store.LocalLibrary, trackID string) ([]byte, string, bool) {
	if data, mime, ok := h.coverCache.Get(lib.ID, trackID); ok {
		return data, mime, true
	}

	track, err := h.localTracks.Get(lib.ID, trackID)
	if err != nil {
		return nil, "", false
	}
	path, err := localmusic.ResolveTrackPath(lib.Path, track.AbsPath)
	if err != nil {
		return nil, "", false
	}
	data, mime, ok := localmusic.ReadEmbeddedCover(path)
	if ok {
		h.coverCache.Set(lib.ID, trackID, data, mime)
		return data, mime, true
	}

	data, mime, ok = h.fetchRemoteCoverForTrack(ctx, track)
	if !ok {
		return nil, "", false
	}
	h.coverCache.Set(lib.ID, trackID, data, mime)
	return data, mime, true
}

func (h *Handler) fetchRemoteCoverForTrack(ctx context.Context, track store.LocalTrack) ([]byte, string, bool) {
	artist := strings.TrimSpace(track.AlbumArtist)
	if artist == "" {
		artist = strings.TrimSpace(track.Artist)
	}
	album := strings.TrimSpace(track.Album)
	title := strings.TrimSpace(track.Title)
	if artist == "" && album == "" && title == "" {
		return nil, "", false
	}

	ctx, cancel := context.WithTimeout(ctx, consts.CoverFetchTimeout)
	defer cancel()

	var artworkURL string
	var err error
	if album != "" {
		artworkURL, err = metadata.LookupAlbumArtworkURL(ctx, artist, album)
		if err != nil {
			slog.Debug("local cover itunes album lookup failed", "track", track.ID, "err", err)
		}
	}
	if artworkURL == "" && title != "" {
		artworkURL, err = metadata.LookupSongArtworkURL(ctx, artist, title, album)
		if err != nil {
			slog.Debug("local cover itunes song lookup failed", "track", track.ID, "err", err)
		}
	}
	if artworkURL == "" && artist != "" {
		artworkURL, err = metadata.LookupArtistArtworkURL(ctx, artist)
		if err != nil {
			slog.Debug("local cover itunes artist lookup failed", "track", track.ID, "err", err)
		}
	}
	if artworkURL == "" {
		return nil, "", false
	}

	body, contentType, err := metadata.FetchArtwork(ctx, artworkURL)
	if err != nil || len(body) == 0 {
		slog.Debug("local cover itunes fetch failed", "track", track.ID, "err", err)
		return nil, "", false
	}
	if contentType == "" || !strings.HasPrefix(contentType, "image/") {
		contentType = "image/jpeg"
	}
	return body, contentType, true
}
