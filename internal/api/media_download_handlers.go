// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"melovian/internal/localmusic"
)

func (s *Server) registerMediaDownloadRoutes() {
	s.mux.HandleFunc("GET /api/media/tracks/{trackId}/download", s.handleMediaTrackDownload)
	s.mux.HandleFunc("GET /api/media/albums/{albumId}/download.zip", s.handleMediaAlbumDownloadZip)
	s.mux.HandleFunc("GET /api/media/playlists/{playlistId}/download.zip", s.handleMediaPlaylistDownloadZip)
	s.mux.HandleFunc("GET /api/media/server-playlists/{playlistId}/download.zip", s.handleMediaServerPlaylistDownloadZip)
}

func (s *Server) handleMediaTrackDownload(w http.ResponseWriter, r *http.Request) {
	trackID := r.PathValue("trackId")
	title := r.URL.Query().Get("title")
	artist := r.URL.Query().Get("artist")
	if strings.HasPrefix(trackID, "trk_") {
		if !s.callerMayAccessLocalTrack(r.Context(), trackID) {
			http.NotFound(w, r)
			return
		}
		s.serveLocalTrackFile(w, r, trackID, true)
		return
	}
	client := s.subsonicForContext(r.Context())
	if !client.Enabled() {
		http.Error(w, "subsonic not configured", http.StatusServiceUnavailable)
		return
	}
	body, contentType, err := client.Stream(trackID)
	if err != nil {
		http.Error(w, "download failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer func() { _ = body.Close() }()
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if title == "" {
		title = trackID
	}
	setAttachmentFilename(w, downloadExportFilename(title, artist, contentType))
	_, _ = io.Copy(w, io.LimitReader(body, maxDownloadBytes))
}

func (s *Server) handleMediaAlbumDownloadZip(w http.ResponseWriter, r *http.Request) {
	albumID := r.PathValue("albumId")
	var songs []mediaZipTrack
	var name string

	if strings.HasPrefix(albumID, "alb_") {
		catalog, err := s.localCatalogForUser(UserIDFromContext(r.Context()))
		if err != nil {
			http.Error(w, "album not found", http.StatusNotFound)
			return
		}
		album, localSongs, ok := catalog.Album(albumID)
		if !ok {
			http.Error(w, "album not found", http.StatusNotFound)
			return
		}
		name = album.Name
		for _, song := range localSongs {
			songs = append(songs, mediaZipTrack{ID: song.ID, Title: song.Title, Artist: song.Artist})
		}
	} else {
		client := s.subsonicForContext(r.Context())
		if !client.Enabled() {
			http.Error(w, "subsonic not configured", http.StatusServiceUnavailable)
			return
		}
		album, err := client.GetAlbum(albumID)
		if err != nil {
			http.Error(w, "album not found", http.StatusNotFound)
			return
		}
		name = album.Name
		for _, song := range album.Songs {
			songs = append(songs, mediaZipTrack{ID: song.ID, Title: song.Title, Artist: song.Artist})
		}
	}
	s.writeTracksZip(w, r, sanitizeDownloadFilename(name)+".zip", songs)
}

func (s *Server) handleMediaPlaylistDownloadZip(w http.ResponseWriter, r *http.Request) {
	playlistID := r.PathValue("playlistId")
	userID := ResolveProgressUserID(r.Context())
	pl, err := s.listen.GetPlaylist(userID, playlistID)
	if err != nil {
		http.Error(w, "playlist not found", http.StatusNotFound)
		return
	}
	songs := make([]mediaZipTrack, 0, len(pl.Tracks))
	for _, t := range pl.Tracks {
		songs = append(songs, mediaZipTrack{ID: t.TrackID, Title: t.TrackTitle, Artist: t.ArtistName})
	}
	s.writeTracksZip(w, r, sanitizeDownloadFilename(pl.Name)+".zip", songs)
}

func (s *Server) handleMediaServerPlaylistDownloadZip(w http.ResponseWriter, r *http.Request) {
	playlistID := r.PathValue("playlistId")
	client := s.subsonicForContext(r.Context())
	if !client.Enabled() {
		http.Error(w, "subsonic not configured", http.StatusServiceUnavailable)
		return
	}
	pl, err := client.GetPlaylist(playlistID)
	if err != nil {
		http.Error(w, "playlist not found", http.StatusNotFound)
		return
	}
	songs := make([]mediaZipTrack, 0, len(pl.Songs))
	for _, song := range pl.Songs {
		songs = append(songs, mediaZipTrack{ID: song.ID, Title: song.Title, Artist: song.Artist})
	}
	s.writeTracksZip(w, r, sanitizeDownloadFilename(pl.Name)+".zip", songs)
}

type mediaZipTrack struct {
	ID     string
	Title  string
	Artist string
}

func (s *Server) writeTracksZip(w http.ResponseWriter, r *http.Request, filename string, songs []mediaZipTrack) {
	if len(songs) == 0 {
		http.Error(w, "no tracks", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	setAttachmentFilename(w, filename)

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
			data, contentType, err := s.readTrackBytes(r, song.ID)
			results[i] = result{
				filename: downloadExportFilename(song.Title, song.Artist, contentType),
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

func (s *Server) readTrackBytes(r *http.Request, trackID string) ([]byte, string, error) {
	if strings.HasPrefix(trackID, "trk_") {
		if !s.callerMayAccessLocalTrack(r.Context(), trackID) {
			return nil, "", fmt.Errorf("not found")
		}
		track, err := s.localTracks.GetByID(trackID)
		if err != nil {
			return nil, "", err
		}
		lib, err := s.localLibraries.Get(track.LibraryID)
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
		data, err := io.ReadAll(io.LimitReader(file, maxDownloadBytes))
		if err != nil {
			return nil, "", err
		}
		return data, localmusic.ContentTypeForFormat(track.Format), nil
	}
	client := s.subsonicForContext(r.Context())
	if !client.Enabled() {
		return nil, "", fmt.Errorf("subsonic not configured")
	}
	body, contentType, err := client.Stream(trackID)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = body.Close() }()
	data, err := io.ReadAll(io.LimitReader(body, maxDownloadBytes))
	if err != nil {
		return nil, "", err
	}
	return data, contentType, nil
}
