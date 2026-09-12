// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package democatalog

import (
	"encoding/json"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// maxDemoResults bounds result capacity so a caller-supplied count cannot
// trigger a huge allocation.
const maxDemoResults = 500

// Handler serves Subsonic REST JSON for the fake catalog under /api/subsonic.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Range, X-Instance-Id, X-Device-Id")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/api/subsonic")
		path = strings.TrimPrefix(path, "/rest/")
		path = strings.TrimPrefix(path, "/")
		endpoint := strings.TrimSuffix(path, ".view")
		endpoint = strings.Split(endpoint, "?")[0]
		q := r.URL.Query()
		c := Get()

		switch endpoint {
		case "ping", "getLicense":
			writeOK(w, map[string]any{
				"type":          ServerName,
				"serverVersion": ServerVersion,
				"openSubsonic":  true,
			})
		case "getCoverArt":
			serveCover(w, q.Get("id"), atoiDefault(q.Get("size"), 300))
		case "stream", "download":
			serveStream(w, r, q.Get("id"))
		case "getArtists":
			writeOK(w, map[string]any{"artists": artistsPayload(c)})
		case "getArtist":
			serveArtist(w, c, q.Get("id"))
		case "getArtistInfo", "getArtistInfo2":
			serveArtistInfo(w, c, q.Get("id"))
		case "getAlbum":
			serveAlbum(w, c, q.Get("id"))
		case "getAlbumList", "getAlbumList2":
			serveAlbumList(w, c, q.Get("type"), atoiDefault(q.Get("size"), 20), atoiDefault(q.Get("offset"), 0), q.Get("genre"))
		case "getSong":
			serveSong(w, c, q.Get("id"))
		case "getRandomSongs":
			serveRandomSongs(w, c, atoiDefault(q.Get("size"), 20))
		case "getGenres":
			serveGenres(w, c)
		case "getSongsByGenre":
			serveSongsByGenre(w, c, q.Get("genre"), atoiDefault(q.Get("count"), 20), atoiDefault(q.Get("offset"), 0))
		case "search2", "search3":
			serveSearch(w, c, q.Get("query"),
				atoiDefault(q.Get("artistCount"), 20),
				atoiDefault(q.Get("albumCount"), 20),
				atoiDefault(q.Get("songCount"), 20),
			)
		case "getPlaylists":
			servePlaylists(w, c)
		case "getPlaylist":
			servePlaylist(w, c, q.Get("id"))
		case "getStarred", "getStarred2":
			serveStarred(w, c)
		case "getSimilarSongs", "getSimilarSongs2":
			serveSimilar(w, c, q.Get("id"), atoiDefault(q.Get("count"), 20))
		case "getScanStatus":
			songs, _, _ := Stats()
			writeOK(w, map[string]any{
				"scanStatus": map[string]any{
					"scanning":    false,
					"count":       songs,
					"folderCount": 1,
					"lastScan":    "2025-08-01T00:00:00Z",
				},
			})
		case "getInternetRadioStations":
			writeOK(w, map[string]any{
				"internetRadioStations": map[string]any{"internetRadioStation": []any{}},
			})
		case "getLyrics", "getLyricsBySongId":
			serveLyrics(w, c, q.Get("id"), q.Get("artist"), q.Get("title"))
		case "star", "unstar", "scrobble", "createPlaylist", "updatePlaylist", "deletePlaylist":
			writeOK(w, map[string]any{})
		default:
			writeErr(w, 0, "unsupported demo endpoint: "+endpoint)
		}
	})
}

func serveStream(w http.ResponseWriter, r *http.Request, id string) {
	c := Get()
	seconds := 200
	if id != "" {
		s, ok := c.Song(id)
		if !ok {
			writeErr(w, 70, "song not found")
			return
		}
		if s.Duration > 0 {
			seconds = s.Duration
		}
	}
	body := NewSilentWAVReader(seconds)
	w.Header().Set("Content-Type", "audio/wav")
	http.ServeContent(w, r, "track.wav", time.Time{}, body)
}

func artistsPayload(c *Catalog) map[string]any {
	indexes := map[string][]map[string]any{}
	for _, a := range c.Artists {
		letter := strings.ToUpper(a.Name[:1])
		indexes[letter] = append(indexes[letter], artistJSON(a))
	}
	indexList := make([]map[string]any, 0, len(indexes))
	for letter, artists := range indexes {
		indexList = append(indexList, map[string]any{
			"name":   letter,
			"artist": artists,
		})
	}
	return map[string]any{"index": indexList}
}

func serveArtist(w http.ResponseWriter, c *Catalog, id string) {
	a, ok := c.Artist(id)
	if !ok {
		writeErr(w, 70, "artist not found")
		return
	}
	albums := c.ArtistAlbums(id)
	albumJSON := make([]map[string]any, 0, len(albums))
	for _, al := range albums {
		albumJSON = append(albumJSON, albumMap(al))
	}
	payload := artistJSON(*a)
	payload["album"] = albumJSON
	writeOK(w, map[string]any{"artist": payload})
}

func serveArtistInfo(w http.ResponseWriter, c *Catalog, id string) {
	a, ok := c.Artist(id)
	if !ok {
		writeErr(w, 70, "artist not found")
		return
	}
	similar := make([]map[string]any, 0, len(a.SimilarIDs))
	for _, sid := range a.SimilarIDs {
		if sa, ok := c.Artist(sid); ok {
			similar = append(similar, map[string]any{
				"id":       sa.ID,
				"name":     sa.Name,
				"coverArt": sa.CoverArt,
			})
		}
	}
	writeOK(w, map[string]any{
		"artistInfo": map[string]any{
			"biography":      a.Biography,
			"similarArtist":  similar,
			"largeImageUrl":  "",
			"mediumImageUrl": "",
			"smallImageUrl":  "",
		},
		"artistInfo2": map[string]any{
			"biography":      a.Biography,
			"similarArtist":  similar,
			"largeImageUrl":  "",
			"mediumImageUrl": "",
			"smallImageUrl":  "",
		},
	})
}

func serveAlbum(w http.ResponseWriter, c *Catalog, id string) {
	al, ok := c.Album(id)
	if !ok {
		writeErr(w, 70, "album not found")
		return
	}
	songs := c.AlbumSongs(id)
	songJSON := make([]map[string]any, 0, len(songs))
	for _, s := range songs {
		songJSON = append(songJSON, songMap(s))
	}
	payload := albumMap(*al)
	payload["song"] = songJSON
	writeOK(w, map[string]any{"album": payload})
}

func serveAlbumList(w http.ResponseWriter, c *Catalog, listType string, size, offset int, genre string) {
	albums := append([]Album(nil), c.Albums...)
	switch listType {
	case "newest":
		sortAlbumsByYearDesc(albums)
	case "alphabeticalByName":
		sortAlbumsByName(albums)
	case "frequent", "recent":
		sortAlbumsByPlayWeight(c, albums)
	case "byGenre":
		filtered := make([]Album, 0)
		for _, a := range albums {
			if strings.EqualFold(a.Genre, genre) {
				filtered = append(filtered, a)
			}
		}
		albums = filtered
	case "random":
		// Keep stable order so screenshots stay consistent.
	default:
		sortAlbumsByYearDesc(albums)
	}
	if offset > len(albums) {
		offset = len(albums)
	}
	end := min(offset+size, len(albums))
	slice := albums[offset:end]
	out := make([]map[string]any, 0, len(slice))
	for _, a := range slice {
		out = append(out, albumMap(a))
	}
	writeOK(w, map[string]any{"albumList2": map[string]any{"album": out}})
}

func serveSong(w http.ResponseWriter, c *Catalog, id string) {
	s, ok := c.Song(id)
	if !ok {
		writeErr(w, 70, "song not found")
		return
	}
	writeOK(w, map[string]any{"song": songMap(*s)})
}

func serveRandomSongs(w http.ResponseWriter, c *Catalog, size int) {
	if size <= 0 {
		size = 20
	}
	if size > maxDemoResults {
		size = maxDemoResults
	}
	songs := c.Songs
	if size > len(songs) {
		size = len(songs)
	}
	out := make([]map[string]any, 0, size)
	step := 3
	for i := 0; i < size; i++ {
		idx := (i * step) % len(songs)
		out = append(out, songMap(songs[idx]))
	}
	writeOK(w, map[string]any{"randomSongs": map[string]any{"song": out}})
}

func serveGenres(w http.ResponseWriter, c *Catalog) {
	out := make([]map[string]any, 0, len(c.Genres))
	for _, g := range c.Genres {
		out = append(out, map[string]any{
			"value":      g.Name,
			"songCount":  g.SongCount,
			"albumCount": g.AlbumCount,
		})
	}
	writeOK(w, map[string]any{"genres": map[string]any{"genre": out}})
}

func serveSongsByGenre(w http.ResponseWriter, c *Catalog, genre string, count, offset int) {
	songs := c.SongsByGenre(genre)
	if offset > len(songs) {
		offset = len(songs)
	}
	end := min(offset+count, len(songs))
	slice := songs[offset:end]
	out := make([]map[string]any, 0, len(slice))
	for _, s := range slice {
		out = append(out, songMap(s))
	}
	writeOK(w, map[string]any{"songsByGenre": map[string]any{"song": out}})
}

func serveSearch(w http.ResponseWriter, c *Catalog, query string, artistCount, albumCount, songCount int) {
	artists, albums, songs := c.Search(query, artistCount, albumCount, songCount)
	aOut := make([]map[string]any, 0, len(artists))
	for _, a := range artists {
		aOut = append(aOut, artistJSON(a))
	}
	alOut := make([]map[string]any, 0, len(albums))
	for _, a := range albums {
		alOut = append(alOut, albumMap(a))
	}
	sOut := make([]map[string]any, 0, len(songs))
	for _, s := range songs {
		sOut = append(sOut, songMap(s))
	}
	result := map[string]any{
		"artist": aOut,
		"album":  alOut,
		"song":   sOut,
	}
	writeOK(w, map[string]any{"searchResult2": result, "searchResult3": result})
}

func servePlaylists(w http.ResponseWriter, c *Catalog) {
	out := make([]map[string]any, 0, len(c.Playlists))
	for _, p := range c.Playlists {
		out = append(out, playlistSummary(c, p))
	}
	writeOK(w, map[string]any{"playlists": map[string]any{"playlist": out}})
}

func servePlaylist(w http.ResponseWriter, c *Catalog, id string) {
	p, ok := c.Playlist(id)
	if !ok {
		writeErr(w, 70, "playlist not found")
		return
	}
	payload := playlistSummary(c, *p)
	entries := make([]map[string]any, 0, len(p.SongIDs))
	for _, sid := range p.SongIDs {
		if s, ok := c.Song(sid); ok {
			entries = append(entries, songMap(*s))
		}
	}
	payload["entry"] = entries
	writeOK(w, map[string]any{"playlist": payload})
}

func serveStarred(w http.ResponseWriter, c *Catalog) {
	songs := make([]map[string]any, 0)
	albums := make([]map[string]any, 0)
	artists := make([]map[string]any, 0)
	for _, s := range c.Songs {
		if s.Starred {
			songs = append(songs, songMap(s))
		}
	}
	for i, a := range c.Albums {
		if i%4 == 0 {
			albums = append(albums, albumMap(a))
		}
	}
	for i, a := range c.Artists {
		if i%3 == 0 {
			artists = append(artists, artistJSON(a))
		}
	}
	payload := map[string]any{
		"song":   songs,
		"album":  albums,
		"artist": artists,
	}
	writeOK(w, map[string]any{"starred": payload, "starred2": payload})
}

func serveSimilar(w http.ResponseWriter, c *Catalog, id string, count int) {
	song, ok := c.Song(id)
	if !ok {
		writeErr(w, 70, "song not found")
		return
	}
	if count <= 0 {
		count = 20
	}
	if count > maxDemoResults {
		count = maxDemoResults
	}
	out := make([]map[string]any, 0, count)
	for _, s := range c.Songs {
		if s.ID == id {
			continue
		}
		if s.Genre == song.Genre || s.ArtistID == song.ArtistID {
			out = append(out, songMap(s))
			if len(out) >= count {
				break
			}
		}
	}
	writeOK(w, map[string]any{"similarSongs": map[string]any{"song": out}, "similarSongs2": map[string]any{"song": out}})
}

func serveLyrics(w http.ResponseWriter, c *Catalog, id, artist, title string) {
	var song *Song
	if id != "" {
		if s, ok := c.Song(id); ok {
			song = s
		}
	}
	if song == nil && title != "" {
		for i := range c.Songs {
			if strings.EqualFold(c.Songs[i].Title, title) &&
				(artist == "" || strings.EqualFold(c.Songs[i].Artist, artist)) {
				song = &c.Songs[i]
				break
			}
		}
	}
	if song == nil {
		writeOK(w, map[string]any{})
		return
	}
	text := song.Title + "\n\n" +
		"Verse one walks through invented rooms\n" +
		"Chorus lifts a name that never charted\n" +
		"Bridge turns quiet for the demo shelf\n" +
		"Outro fades into silent WAV"
	writeOK(w, map[string]any{
		"lyrics": map[string]any{
			"artist": song.Artist,
			"title":  song.Title,
			"value":  text,
		},
	})
}

func artistJSON(a Artist) map[string]any {
	return map[string]any{
		"id":         a.ID,
		"name":       a.Name,
		"albumCount": len(a.AlbumIDs),
		"coverArt":   a.CoverArt,
	}
}

func albumMap(a Album) map[string]any {
	return map[string]any{
		"id":        a.ID,
		"name":      a.Name,
		"artist":    a.Artist,
		"artistId":  a.ArtistID,
		"year":      a.Year,
		"songCount": len(a.SongIDs),
		"duration":  albumDuration(a),
		"coverArt":  a.CoverArt,
		"genre":     a.Genre,
		"created":   a.CreatedAt,
	}
}

func albumDuration(a Album) int {
	c := Get()
	total := 0
	for _, id := range a.SongIDs {
		if s, ok := c.Song(id); ok {
			total += s.Duration
		}
	}
	return total
}

func songMap(s Song) map[string]any {
	m := map[string]any{
		"id":          s.ID,
		"title":       s.Title,
		"album":       s.Album,
		"albumId":     s.AlbumID,
		"artist":      s.Artist,
		"artistId":    s.ArtistID,
		"track":       s.Track,
		"duration":    s.Duration,
		"year":        s.Year,
		"genre":       s.Genre,
		"coverArt":    s.CoverArt,
		"bitRate":     s.BitRate,
		"contentType": s.ContentType,
		"suffix":      s.Suffix,
		"playCount":   s.PlayCount,
		"type":        "music",
		"isDir":       false,
	}
	if s.Starred {
		m["starred"] = "2025-01-15T12:00:00Z"
	}
	return m
}

func playlistSummary(c *Catalog, p Playlist) map[string]any {
	duration := 0
	var cover string
	for i, sid := range p.SongIDs {
		if s, ok := c.Song(sid); ok {
			duration += s.Duration
			if i == 0 {
				cover = s.CoverArt
			}
		}
	}
	return map[string]any{
		"id":        p.ID,
		"name":      p.Name,
		"comment":   p.Comment,
		"owner":     p.Owner,
		"public":    p.Public,
		"songCount": len(p.SongIDs),
		"duration":  duration,
		"created":   p.Created,
		"changed":   p.Changed,
		"coverArt":  cover,
	}
}

func writeOK(w http.ResponseWriter, extra map[string]any) {
	payload := map[string]any{
		"status":        "ok",
		"version":       SubsonicVer,
		"type":          ServerName,
		"serverVersion": ServerVersion,
		"openSubsonic":  true,
	}
	maps.Copy(payload, extra)
	writeJSON(w, http.StatusOK, map[string]any{"subsonic-response": payload})
}

func writeErr(w http.ResponseWriter, code int, message string) {
	writeJSON(w, http.StatusOK, map[string]any{
		"subsonic-response": map[string]any{
			"status":  "failed",
			"version": SubsonicVer,
			"error":   map[string]any{"code": code, "message": message},
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func atoiDefault(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}

func sortAlbumsByYearDesc(albums []Album) {
	for i := range albums {
		for j := i + 1; j < len(albums); j++ {
			if albums[j].Year > albums[i].Year ||
				(albums[j].Year == albums[i].Year && albums[j].Name < albums[i].Name) {
				albums[i], albums[j] = albums[j], albums[i]
			}
		}
	}
}

func sortAlbumsByName(albums []Album) {
	for i := range albums {
		for j := i + 1; j < len(albums); j++ {
			if albums[j].Name < albums[i].Name {
				albums[i], albums[j] = albums[j], albums[i]
			}
		}
	}
}

func sortAlbumsByPlayWeight(c *Catalog, albums []Album) {
	weight := func(a Album) int {
		total := 0
		for _, id := range a.SongIDs {
			if s, ok := c.Song(id); ok {
				total += s.PlayCount
			}
		}
		return total
	}
	for i := range albums {
		for j := i + 1; j < len(albums); j++ {
			if weight(albums[j]) > weight(albums[i]) {
				albums[i], albums[j] = albums[j], albums[i]
			}
		}
	}
}
