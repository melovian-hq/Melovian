// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

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

	"melovian/internal/httputil"
	"melovian/internal/localmusic"
	"melovian/internal/metadata"
	"melovian/internal/store"
)

func (s *Server) registerLocalMusicRoutes() {
	s.mux.HandleFunc("GET /api/local-music/artists", s.handleLocalMusicArtists)
	s.mux.HandleFunc("GET /api/local-music/artists/{id}", s.handleLocalMusicArtist)
	s.mux.HandleFunc("GET /api/local-music/albums", s.handleLocalMusicAlbumList)
	s.mux.HandleFunc("GET /api/local-music/albums/{id}", s.handleLocalMusicAlbum)
	s.mux.HandleFunc("GET /api/local-music/search", s.handleLocalMusicSearch)
	s.mux.HandleFunc("GET /api/local-music/songs/{id}", s.handleLocalMusicSong)
	s.mux.HandleFunc("GET /api/local-music/randomSongs", s.handleLocalMusicRandomSongs)
	s.mux.HandleFunc("GET /api/local-music/genres", s.handleLocalMusicGenres)
	s.mux.HandleFunc("GET /api/local-music/songsByGenre", s.handleLocalMusicSongsByGenre)
	s.mux.HandleFunc("GET /api/local-music/starred", s.handleLocalMusicStarred)
	s.mux.HandleFunc("GET /api/local-music/songs/{id}/similar", s.handleLocalMusicSimilar)
	s.mux.HandleFunc("GET /api/local-music/tracks/{id}/stream", s.handleLocalMusicStream)
	s.mux.HandleFunc("GET /api/local-music/cover/{id}", s.handleLocalMusicCover)
}

func (s *Server) invalidateLocalLibraryData(libraryID string) {
	s.catalogCache.Invalidate(libraryID)
	s.coverCache.InvalidateLibrary(libraryID)
	s.libraryScanner.ForgetLibrary(libraryID)
}

func (s *Server) activeLocalCatalog(r *http.Request) (localmusic.Catalog, store.LocalLibrary, error) {
	userID := UserIDFromContext(r.Context())
	catalog, err := s.localCatalogForUser(userID)
	if err != nil {
		return localmusic.Catalog{}, store.LocalLibrary{}, err
	}
	lib, err := s.localLibraries.GetActiveForUser(userID)
	if err != nil {
		return localmusic.Catalog{}, store.LocalLibrary{}, err
	}
	return catalog, lib, nil
}

func (s *Server) handleLocalMusicArtists(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := s.activeLocalCatalog(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, localArtistsResponse{
		Artists: localArtistViewsFrom(catalog.Artists),
	})
}

func (s *Server) handleLocalMusicArtist(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := s.activeLocalCatalog(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	artist, albums, ok := catalog.Artist(r.PathValue("id"))
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, localArtistDetailResponse{
		Artist: localArtistViewFrom(artist),
		Albums: localAlbumViewsFrom(albums),
	})
}

func (s *Server) handleLocalMusicAlbumList(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := s.activeLocalCatalog(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
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

func (s *Server) handleLocalMusicAlbum(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := s.activeLocalCatalog(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	album, songs, ok := catalog.Album(r.PathValue("id"))
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, localAlbumDetailResponse{
		Album: localAlbumViewFrom(album),
		Songs: s.decorateLocalSongViews(r, localSongViewsFrom(songs)),
	})
}

func (s *Server) handleLocalMusicSearch(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := s.activeLocalCatalog(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	artists, albums, songs := catalog.Search(r.URL.Query().Get("q"), limit)
	httputil.WriteJSON(w, http.StatusOK, localSearchResponse{
		Artists: localArtistViewsFrom(artists),
		Albums:  localAlbumViewsFrom(albums),
		Songs:   s.decorateLocalSongViews(r, localSongViewsFrom(songs)),
	})
}

func (s *Server) handleLocalMusicSong(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := s.activeLocalCatalog(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	song, ok := catalog.Song(r.PathValue("id"))
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	views := s.decorateLocalSongViews(r, []localSongView{localSongViewFrom(song)})
	httputil.WriteJSON(w, http.StatusOK, views[0])
}

func (s *Server) handleLocalMusicRandomSongs(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := s.activeLocalCatalog(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	httputil.WriteJSON(w, http.StatusOK, localSongsResponse{
		Songs: s.decorateLocalSongViews(r, localSongViewsFrom(catalog.RandomSongs(size))),
	})
}

func (s *Server) handleLocalMusicGenres(w http.ResponseWriter, r *http.Request) {
	_, lib, err := s.activeLocalCatalog(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	counts, err := s.localTracks.ListGenreCounts([]string{lib.ID})
	if err != nil {
		http.Error(w, "failed to load genres", http.StatusInternalServerError)
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

func (s *Server) handleLocalMusicSongsByGenre(w http.ResponseWriter, r *http.Request) {
	_, lib, err := s.activeLocalCatalog(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	genre := strings.TrimSpace(r.URL.Query().Get("genre"))
	if genre == "" {
		httputil.WriteJSON(w, http.StatusOK, localSongsResponse{Songs: []localSongView{}})
		return
	}
	count, _ := strconv.Atoi(r.URL.Query().Get("count"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	tracks, err := s.localTracks.ListByGenre(lib.ID, genre, count, offset)
	if err != nil {
		http.Error(w, "failed to load songs", http.StatusInternalServerError)
		return
	}
	views := make([]localSongView, 0, len(tracks))
	for _, track := range tracks {
		views = append(views, localSongViewFromTrack(track))
	}
	httputil.WriteJSON(w, http.StatusOK, localSongsResponse{
		Songs: s.decorateLocalSongViews(r, views),
	})
}

func (s *Server) handleLocalMusicStarred(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := s.activeLocalCatalog(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	userID := ResolveProgressUserID(r.Context())
	favorites, err := s.listen.ListFavorites(userID, 2000)
	if err != nil {
		http.Error(w, "failed to load favorites", http.StatusInternalServerError)
		return
	}
	songs := make([]localSongView, 0, len(favorites))
	for _, fav := range favorites {
		if !strings.HasPrefix(fav.TrackID, "trk_") {
			continue
		}
		if song, ok := catalog.Song(fav.TrackID); ok {
			view := localSongViewFrom(song)
			view.Starred = fav.FavoritedAt.UTC().Format(time.RFC3339)
			songs = append(songs, view)
			continue
		}
		// Track is favorited but no longer in the catalog (missing file).
		// Still surface it so the favorites page does not lose the entry.
		songs = append(songs, localSongView{
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
		"songs":   s.decorateLocalSongViews(r, songs),
		"albums":  []localAlbumView{},
		"artists": []localArtistView{},
	})
}

func (s *Server) handleLocalMusicSimilar(w http.ResponseWriter, r *http.Request) {
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
	if _, err := s.localLibraries.GetForUser(userID, track.LibraryID); err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	count, _ := strconv.Atoi(r.URL.Query().Get("count"))
	tracks, err := s.localTracks.ListSimilarTracks(track.LibraryID, track.Genre, track.Artist, trackID, count)
	if err != nil {
		http.Error(w, "failed to load similar songs", http.StatusInternalServerError)
		return
	}
	views := make([]localSongView, 0, len(tracks))
	for _, item := range tracks {
		views = append(views, localSongViewFromTrack(item))
	}
	httputil.WriteJSON(w, http.StatusOK, localSongsResponse{
		Songs: s.decorateLocalSongViews(r, views),
	})
}

func (s *Server) handleLocalMusicStream(w http.ResponseWriter, r *http.Request) {
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
	lib, err := s.localLibraries.GetForUser(userID, track.LibraryID)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	if track.Status != store.TrackStatusPresent || track.DuplicateOf != "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	path, err := localmusic.ResolveTrackPath(lib.Path, track.AbsPath)
	if err != nil {
		http.Error(w, "invalid track path", http.StatusForbidden)
		return
	}

	file, err := os.Open(path) //#nosec G304 -- path validated against library root
	if err != nil {
		http.Error(w, "open track", http.StatusNotFound)
		return
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		http.Error(w, "stat track", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", localmusic.ContentTypeForFormat(track.Format))
	http.ServeContent(w, r, filepath.Base(path), info.ModTime(), file)
}

func (s *Server) handleLocalMusicCover(w http.ResponseWriter, r *http.Request) {
	catalog, _, err := s.activeLocalCatalog(r)
	if err != nil {
		s.writeLocalMusicError(w, err)
		return
	}
	id := r.PathValue("id")
	data, contentType, ok := s.localCoverData(r, catalog, id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, "cover", time.Now(), bytes.NewReader(data))
}

// decorateLocalSongViews fills play counts and last-played timestamps from the
// user's listen history. Best effort: failures leave the fields empty.
func (s *Server) decorateLocalSongViews(r *http.Request, views []localSongView) []localSongView {
	if len(views) == 0 {
		return views
	}
	userID := ResolveProgressUserID(r.Context())
	ids := make([]string, 0, len(views))
	for _, view := range views {
		ids = append(ids, view.ID)
	}
	progress, err := s.listen.Batch(userID, ids)
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

func (s *Server) writeLocalMusicError(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "no active local library", http.StatusNotFound)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func (s *Server) localCoverData(r *http.Request, catalog localmusic.Catalog, id string) ([]byte, string, bool) {
	trackID, ok := catalog.CoverTrackID(id)
	if !ok {
		return nil, "", false
	}
	track, err := s.localTracks.GetByID(trackID)
	if err != nil {
		return nil, "", false
	}
	userID := UserIDFromContext(r.Context())
	lib, err := s.localLibraries.GetForUser(userID, track.LibraryID)
	if err != nil {
		return nil, "", false
	}
	return s.coverBytesFromTrackFile(lib, trackID)
}

func (s *Server) coverBytesFromTrackFile(lib store.LocalLibrary, trackID string) ([]byte, string, bool) {
	if data, mime, ok := s.coverCache.Get(lib.ID, trackID); ok {
		return data, mime, true
	}

	track, err := s.localTracks.Get(lib.ID, trackID)
	if err != nil {
		return nil, "", false
	}
	path, err := localmusic.ResolveTrackPath(lib.Path, track.AbsPath)
	if err != nil {
		return nil, "", false
	}
	data, mime, ok := localmusic.ReadEmbeddedCover(path)
	if ok {
		s.coverCache.Set(lib.ID, trackID, data, mime)
		return data, mime, true
	}

	data, mime, ok = s.fetchRemoteCoverForTrack(track)
	if !ok {
		return nil, "", false
	}
	s.coverCache.Set(lib.ID, trackID, data, mime)
	return data, mime, true
}

func (s *Server) fetchRemoteCoverForTrack(track store.LocalTrack) ([]byte, string, bool) {
	artist := strings.TrimSpace(track.AlbumArtist)
	if artist == "" {
		artist = strings.TrimSpace(track.Artist)
	}
	album := strings.TrimSpace(track.Album)
	title := strings.TrimSpace(track.Title)
	if artist == "" && album == "" && title == "" {
		return nil, "", false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
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
