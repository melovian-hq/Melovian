// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonicserver

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode"

	"melovian/internal/localmusic"
)

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	writeOK(w, r, map[string]any{})
}

func (s *Server) handleLicense(w http.ResponseWriter, r *http.Request) {
	writeOK(w, r, map[string]any{
		"license": map[string]any{
			"valid":   true,
			"email":   "",
			"expires": "9999-12-31T23:59:59",
			"trial":   false,
			"error":   nil,
		},
	})
}

func (s *Server) handleGetArtists(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	catalog, err := s.provider.Catalog(ctx, userID)
	if err != nil {
		writeError(w, r, 0, err.Error())
		return
	}
	writeOK(w, r, map[string]any{
		"artists": map[string]any{
			"index": artistIndexes(catalog.Artists),
		},
	})
}

func (s *Server) handleGetArtist(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	catalog, err := s.provider.Catalog(ctx, userID)
	if err != nil {
		writeError(w, r, 0, err.Error())
		return
	}
	artist, albums, ok := catalog.Artist(id)
	if !ok {
		writeError(w, r, 70, "artist not found")
		return
	}
	writeOK(w, r, map[string]any{
		"artist": map[string]any{
			"id":         artist.ID,
			"name":       artist.Name,
			"albumCount": artist.AlbumCount,
			"coverArt":   artist.CoverArt,
			"album":      albumViews(albums),
		},
	})
}

func (s *Server) handleGetAlbum(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	catalog, err := s.provider.Catalog(ctx, userID)
	if err != nil {
		writeError(w, r, 0, err.Error())
		return
	}
	album, songs, ok := catalog.Album(id)
	if !ok {
		writeError(w, r, 70, "album not found")
		return
	}
	writeOK(w, r, map[string]any{
		"album": map[string]any{
			"id":        album.ID,
			"name":      album.Name,
			"artist":    album.Artist,
			"artistId":  album.ArtistID,
			"coverArt":  album.CoverArt,
			"songCount": album.SongCount,
			"duration":  album.Duration,
			"song":      songViews(songs),
		},
	})
}

func (s *Server) handleGetAlbumList2(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	catalog, err := s.provider.Catalog(ctx, userID)
	if err != nil {
		writeError(w, r, 0, err.Error())
		return
	}
	listType := r.URL.Query().Get("type")
	if listType == "" {
		listType = "newest"
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	writeOK(w, r, map[string]any{
		"albumList2": map[string]any{
			"album": albumViews(catalog.AlbumList(listType, size, offset)),
		},
	})
}

func (s *Server) handleGetSong(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	catalog, err := s.provider.Catalog(ctx, userID)
	if err != nil {
		writeError(w, r, 0, err.Error())
		return
	}
	song, ok := catalog.Song(id)
	if !ok {
		writeError(w, r, 70, "song not found")
		return
	}
	writeOK(w, r, map[string]any{"song": songView(song)})
}

func (s *Server) handleGetRandomSongs(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	catalog, err := s.provider.Catalog(ctx, userID)
	if err != nil {
		writeError(w, r, 0, err.Error())
		return
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	writeOK(w, r, map[string]any{
		"randomSongs": map[string]any{
			"song": songViews(catalog.RandomSongs(size)),
		},
	})
}

func (s *Server) handleSearch3(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	catalog, err := s.provider.Catalog(ctx, userID)
	if err != nil {
		writeError(w, r, 0, err.Error())
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("songCount"))
	if limit <= 0 {
		limit, _ = strconv.Atoi(r.URL.Query().Get("artistCount"))
	}
	artists, albums, songs := catalog.Search(query, limit)
	writeOK(w, r, map[string]any{
		"searchResult3": map[string]any{
			"artist": artistViews(artists),
			"album":  albumViews(albums),
			"song":   songViews(songs),
		},
	})
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	track, lib, err := s.provider.Track(ctx, userID, id)
	if err != nil {
		writeError(w, r, 70, "track not found")
		return
	}
	path, err := s.provider.StreamPath(ctx, userID, track, lib)
	if err != nil {
		writeError(w, r, 0, err.Error())
		return
	}
	if streamWithTranscode(w, r, ctx, path, track.Format) {
		return
	}
	file, err := os.Open(path) //#nosec G304,G703 -- path from StreamPath under library root
	if err != nil {
		writeError(w, r, 0, err.Error())
		return
	}
	defer func() { _ = file.Close() }()
	w.Header().Set("Content-Type", localmusic.ContentTypeForFormat(track.Format))
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, track.RelPath, track.UpdatedAt, file)
}

func (s *Server) handleGetCoverArt(w http.ResponseWriter, r *http.Request, ctx context.Context, userID string) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	data, contentType, err := s.provider.Cover(ctx, userID, id)
	if err != nil {
		writeError(w, r, 70, "cover not found")
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(data) //#nosec G705 -- cover bytes with explicit image content type
}

func artistIndexes(artists []localmusic.Artist) []map[string]any {
	grouped := make(map[string][]map[string]any)
	order := make([]string, 0)
	for _, artist := range artists {
		key := indexKey(artist.Name)
		if _, ok := grouped[key]; !ok {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], artistView(artist))
	}
	out := make([]map[string]any, 0, len(order))
	for _, key := range order {
		out = append(out, map[string]any{
			"name":   key,
			"artist": grouped[key],
		})
	}
	return out
}

func indexKey(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "#"
	}
	runes := []rune(strings.ToUpper(trimmed))
	first := runes[0]
	if unicode.IsLetter(first) || unicode.IsDigit(first) {
		return string(first)
	}
	return "#"
}

func artistViews(artists []localmusic.Artist) []map[string]any {
	out := make([]map[string]any, len(artists))
	for i, artist := range artists {
		out[i] = artistView(artist)
	}
	return out
}

func artistView(artist localmusic.Artist) map[string]any {
	return map[string]any{
		"id":         artist.ID,
		"name":       artist.Name,
		"albumCount": artist.AlbumCount,
		"coverArt":   artist.CoverArt,
	}
}

func albumViews(albums []localmusic.Album) []map[string]any {
	out := make([]map[string]any, len(albums))
	for i, album := range albums {
		out[i] = albumView(album)
	}
	return out
}

func albumView(album localmusic.Album) map[string]any {
	return map[string]any{
		"id":        album.ID,
		"name":      album.Name,
		"artist":    album.Artist,
		"artistId":  album.ArtistID,
		"coverArt":  album.CoverArt,
		"songCount": album.SongCount,
		"duration":  album.Duration,
	}
}

func songViews(songs []localmusic.Song) []map[string]any {
	out := make([]map[string]any, len(songs))
	for i, song := range songs {
		out[i] = songView(song)
	}
	return out
}

func songView(song localmusic.Song) map[string]any {
	return map[string]any{
		"id":       song.ID,
		"title":    song.Title,
		"album":    song.Album,
		"albumId":  song.AlbumID,
		"artist":   song.Artist,
		"artistId": song.ArtistID,
		"track":    song.Track,
		"duration": song.Duration,
		"coverArt": song.CoverArt,
		"suffix":   song.Suffix,
		"type":     "music",
		"isDir":    false,
	}
}
