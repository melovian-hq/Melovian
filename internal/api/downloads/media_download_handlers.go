// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package downloads

import (
	"archive/zip"
	"fmt"
	"io"
	"melovian/internal/api/apishared"
	"melovian/internal/httputil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"melovian/internal/localmusic"
)

func (h *Handler) registerMediaDownloadRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/media/tracks/{trackId}/download", h.handleMediaTrackDownload)
	mux.HandleFunc("GET /api/media/albums/{albumId}/download.zip", h.handleMediaAlbumDownloadZip)
	mux.HandleFunc("GET /api/media/playlists/{playlistId}/download.zip", h.handleMediaPlaylistDownloadZip)
	mux.HandleFunc("GET /api/media/server-playlists/{playlistId}/download.zip", h.handleMediaServerPlaylistDownloadZip)
}

func (h *Handler) handleMediaTrackDownload(w http.ResponseWriter, r *http.Request) {
	trackID := r.PathValue("trackId")
	title := r.URL.Query().Get("title")
	artist := r.URL.Query().Get("artist")
	if strings.HasPrefix(trackID, "trk_") {
		if !h.library.CallerMayAccessLocalTrack(r.Context(), trackID) {
			http.NotFound(w, r)
			return
		}
		h.library.ServeLocalTrackFile(w, r, trackID, true)
		return
	}
	client := h.resolver.ForContext(r.Context())
	if !client.Enabled() {
		httputil.WriteError(w, http.StatusServiceUnavailable, "service_unavailable", "subsonic not configured")
		return
	}
	body, contentType, err := client.Stream(trackID)
	if err != nil {
		httputil.WriteInternalError(w, r, "download failed:", err)
		return
	}
	defer func() { _ = body.Close() }()
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if title == "" {
		title = trackID
	}
	apishared.SetAttachmentFilename(w, apishared.DownloadExportFilename(title, artist, contentType))
	_, _ = io.Copy(w, io.LimitReader(body, apishared.MaxDownloadBytes))
}

func (h *Handler) handleMediaAlbumDownloadZip(w http.ResponseWriter, r *http.Request) {
	albumID := r.PathValue("albumId")
	var songs []mediaZipTrack
	var name string

	if strings.HasPrefix(albumID, "alb_") {
		catalog, err := h.library.LocalCatalogForUser(apishared.UserIDFromContext(r.Context()))
		if err != nil {
			httputil.WriteError(w, http.StatusNotFound, "album_not_found", "album not found")
			return
		}
		album, localSongs, ok := catalog.Album(albumID)
		if !ok {
			httputil.WriteError(w, http.StatusNotFound, "album_not_found", "album not found")
			return
		}
		name = album.Name
		for _, song := range localSongs {
			songs = append(songs, mediaZipTrack{ID: song.ID, Title: song.Title, Artist: song.Artist})
		}
	} else {
		client := h.resolver.ForContext(r.Context())
		if !client.Enabled() {
			httputil.WriteError(w, http.StatusServiceUnavailable, "service_unavailable", "subsonic not configured")
			return
		}
		album, err := client.GetAlbum(albumID)
		if err != nil {
			httputil.WriteError(w, http.StatusNotFound, "album_not_found", "album not found")
			return
		}
		name = album.Name
		for _, song := range album.Songs {
			songs = append(songs, mediaZipTrack{ID: song.ID, Title: song.Title, Artist: song.Artist})
		}
	}
	h.writeTracksZip(w, r, apishared.SanitizeDownloadFilename(name)+".zip", songs)
}

func (h *Handler) handleMediaPlaylistDownloadZip(w http.ResponseWriter, r *http.Request) {
	playlistID := r.PathValue("playlistId")
	userID := apishared.ResolveProgressUserID(r.Context())
	pl, err := h.listen.GetPlaylist(userID, playlistID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "playlist_not_found", "playlist not found")
		return
	}
	songs := make([]mediaZipTrack, 0, len(pl.Tracks))
	for _, t := range pl.Tracks {
		songs = append(songs, mediaZipTrack{ID: t.TrackID, Title: t.TrackTitle, Artist: t.ArtistName})
	}
	h.writeTracksZip(w, r, apishared.SanitizeDownloadFilename(pl.Name)+".zip", songs)
}

func (h *Handler) handleMediaServerPlaylistDownloadZip(w http.ResponseWriter, r *http.Request) {
	playlistID := r.PathValue("playlistId")
	client := h.resolver.ForContext(r.Context())
	if !client.Enabled() {
		httputil.WriteError(w, http.StatusServiceUnavailable, "service_unavailable", "subsonic not configured")
		return
	}
	pl, err := client.GetPlaylist(playlistID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "playlist_not_found", "playlist not found")
		return
	}
	songs := make([]mediaZipTrack, 0, len(pl.Songs))
	for _, song := range pl.Songs {
		songs = append(songs, mediaZipTrack{ID: song.ID, Title: song.Title, Artist: song.Artist})
	}
	h.writeTracksZip(w, r, apishared.SanitizeDownloadFilename(pl.Name)+".zip", songs)
}

type mediaZipTrack struct {
	ID     string
	Title  string
	Artist string
}

func (h *Handler) writeTracksZip(w http.ResponseWriter, r *http.Request, filename string, songs []mediaZipTrack) {
	if len(songs) == 0 {
		httputil.WriteError(w, http.StatusNotFound, "no_tracks", "no tracks")
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	apishared.SetAttachmentFilename(w, filename)

	zipWriter := zip.NewWriter(w)
	defer func() { _ = zipWriter.Close() }()

	usedNames := make(map[string]int, len(songs))
	sem := make(chan struct{}, 3)
	type result struct {
		filename string
		data     []byte
		err      error
	}
	results := make([]result, len(songs))
	var wg sync.WaitGroup

	for i, song := range songs {
		wg.Add(1)
		go func(i int, song mediaZipTrack) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-r.Context().Done():
				results[i] = result{err: r.Context().Err()}
				return
			}
			data, contentType, err := h.readTrackBytes(r, song.ID)
			results[i] = result{
				filename: apishared.DownloadExportFilename(song.Title, song.Artist, contentType),
				data:     data,
				err:      err,
			}
		}(i, song)
	}
	wg.Wait()

	for _, res := range results {
		if res.err != nil || len(res.data) == 0 {
			continue
		}
		name := res.filename
		if count := usedNames[name]; count > 0 {
			ext := filepath.Ext(name)
			base := strings.TrimSuffix(name, ext)
			name = fmt.Sprintf("%s (%d)%s", base, count+1, ext)
		}
		usedNames[name]++

		header := &zip.FileHeader{
			Name:     name,
			Method:   zip.Deflate,
			Modified: time.Now(),
		}
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			continue
		}
		_, _ = writer.Write(res.data)
	}
}

func (h *Handler) readTrackBytes(r *http.Request, trackID string) ([]byte, string, error) {
	if strings.HasPrefix(trackID, "trk_") {
		if !h.library.CallerMayAccessLocalTrack(r.Context(), trackID) {
			return nil, "", fmt.Errorf("not found")
		}
		track, err := h.localTracks.GetByID(trackID)
		if err != nil {
			return nil, "", err
		}
		lib, err := h.localLibraries.Get(track.LibraryID)
		if err != nil {
			return nil, "", err
		}
		path, err := localmusic.ResolveTrackPath(lib.Path, track.AbsPath)
		if err != nil {
			return nil, "", err
		}
		file, err := os.Open(path) //#nosec G304 -- path resolved under library jail
		if err != nil {
			return nil, "", err
		}
		defer func() { _ = file.Close() }()
		data, err := io.ReadAll(io.LimitReader(file, apishared.MaxDownloadBytes))
		if err != nil {
			return nil, "", err
		}
		return data, localmusic.ContentTypeForFormat(track.Format), nil
	}
	client := h.resolver.ForContext(r.Context())
	if !client.Enabled() {
		return nil, "", fmt.Errorf("subsonic not configured")
	}
	body, contentType, err := client.Stream(trackID)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = body.Close() }()
	data, err := io.ReadAll(io.LimitReader(body, apishared.MaxDownloadBytes))
	if err != nil {
		return nil, "", err
	}
	return data, contentType, nil
}
