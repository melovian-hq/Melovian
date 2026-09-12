// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonic

import (
	"crypto/md5" //#nosec G501 -- Subsonic token auth requires MD5 per protocol spec
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"time"

	"melovian/internal/consts"
	"melovian/internal/democatalog"
	"melovian/internal/httputil"
)

const ClientName = "melovian"
const Version = consts.SubsonicVersion

type Client struct {
	ServerURL string
	Username  string
	Password  string
	client    *http.Client
	stream    *http.Client
}

type PingResponse struct {
	XMLName xml.Name `xml:"subsonic-response"`
	Status  string   `xml:"status,attr"`
	Version string   `xml:"version,attr"`
	Type    string   `xml:"type,attr"`
	Server  string   `xml:"serverVersion,attr"`
	Open    string   `xml:"openSubsonic,attr"`
	Error   *struct {
		Code    int    `xml:"code,attr"`
		Message string `xml:"message,attr"`
	} `xml:"error"`
}

type StatusResponse struct {
	Enabled    bool   `json:"enabled"`
	Connected  bool   `json:"connected"`
	ServerName string `json:"serverName,omitempty"`
	Version    string `json:"version,omitempty"`
	Error      string `json:"error,omitempty"`
}

type LibraryStatsResponse struct {
	SongCount   int    `json:"songCount"`
	AlbumCount  int    `json:"albumCount"`
	ArtistCount int    `json:"artistCount"`
	FolderCount int    `json:"folderCount"`
	Scanning    bool   `json:"scanning"`
	LastScan    string `json:"lastScan,omitempty"`
}

func NewClient(serverURL, username, password string) *Client {
	return &Client{
		ServerURL: strings.TrimRight(serverURL, "/"),
		Username:  username,
		Password:  password,
		client:    httputil.NewRetryHTTPClient(),
		stream:    httputil.NewStreamingHTTPClient(),
	}
}

func (s *Client) Enabled() bool {
	return s.ServerURL != "" && s.Username != "" && s.Password != ""
}

func (s *Client) authParams() url.Values {
	salt := randomSalt()
	token := md5.Sum([]byte(s.Password + salt)) //#nosec G401 -- Subsonic API mandated token = md5(password + salt)
	return url.Values{
		"u": {s.Username},
		"t": {hex.EncodeToString(token[:])},
		"s": {salt},
		"v": {Version},
		"c": {ClientName},
	}
}

func randomSalt() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func (s *Client) InjectAuth(query url.Values) url.Values {
	auth := s.authParams()
	merged := make(url.Values, len(query)+len(auth))
	for key, values := range query {
		if isAuthParam(key) {
			continue
		}
		for _, value := range values {
			merged.Add(key, value)
		}
	}
	maps.Copy(merged, auth)
	return merged
}

func isAuthParam(key string) bool {
	switch strings.ToLower(key) {
	case "u", "p", "t", "s", "jwt":
		return true
	default:
		return false
	}
}

func (s *Client) Ping() (serverName, version string, err error) {
	if democatalog.IsFakeURL(s.ServerURL) {
		return democatalog.ServerName, democatalog.ServerVersion, nil
	}

	query := s.authParams()
	query.Set("f", "json")

	endpoint := s.ServerURL + "/rest/ping.view?" + query.Encode()
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", "", err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", "", httputil.SanitizeErrorURL(err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := httputil.ReadLimited(resp.Body, 4<<20)
	if err != nil {
		return "", "", err
	}

	var payload struct {
		SubsonicResponse struct {
			Status        string `json:"status"`
			Version       string `json:"version"`
			Type          string `json:"type"`
			ServerVersion string `json:"serverVersion"`
			Server        struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"server"`
			Error *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		} `json:"subsonic-response"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		var xmlResp PingResponse
		if xmlErr := xml.Unmarshal(body, &xmlResp); xmlErr != nil {
			return "", "", fmt.Errorf("parse ping response: %w", err)
		}
		if xmlResp.Status != "ok" {
			msg := "subsonic ping failed"
			if xmlResp.Error != nil {
				msg = xmlResp.Error.Message
			}
			return "", "", fmt.Errorf("%s", msg)
		}
		name := xmlResp.Type
		if name == "" {
			name = "Subsonic"
		}
		return name, xmlResp.Server, nil
	}

	if payload.SubsonicResponse.Status != "ok" {
		msg := "subsonic ping failed"
		if payload.SubsonicResponse.Error != nil {
			msg = payload.SubsonicResponse.Error.Message
		}
		return "", "", fmt.Errorf("%s", msg)
	}

	// OpenSubsonic (Navidrome etc.) uses type + serverVersion. Older payloads
	// nest name/version under server.
	name := payload.SubsonicResponse.Type
	if name == "" {
		name = payload.SubsonicResponse.Server.Name
	}
	if name == "" {
		name = "Subsonic"
	}
	ver := payload.SubsonicResponse.ServerVersion
	if ver == "" {
		ver = payload.SubsonicResponse.Server.Version
	}
	if ver == "" {
		ver = payload.SubsonicResponse.Version
	}
	return name, ver, nil
}

func (s *Client) getJSON(endpoint string, query url.Values) ([]byte, error) {
	if query == nil {
		query = url.Values{}
	}
	query.Set("f", "json")
	merged := s.InjectAuth(query)
	target := s.ServerURL + "/rest/" + endpoint + "?" + merged.Encode()
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, httputil.SanitizeErrorURL(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := httputil.ReadLimited(resp.Body, 4<<20)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subsonic %s: %s", endpoint, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// PlaylistSong is a song entry from getPlaylist or getAlbum.
type PlaylistSong struct {
	ID       string
	Title    string
	Artist   string
	Album    string
	AlbumID  string
	Duration int
	CoverArt string
	Track    int
}

// PlaylistDetail is a Subsonic playlist with song entries.
type PlaylistDetail struct {
	ID    string
	Name  string
	Songs []PlaylistSong
}

// AlbumDetail is a Subsonic album with song entries.
type AlbumDetail struct {
	ID     string
	Name   string
	Artist string
	Songs  []PlaylistSong
}

// GetPlaylist fetches a playlist and its songs from the Subsonic server.
func (s *Client) GetPlaylist(id string) (PlaylistDetail, error) {
	body, err := s.getJSON("getPlaylist.view", url.Values{"id": {id}})
	if err != nil {
		return PlaylistDetail{}, err
	}
	var payload struct {
		SubsonicResponse struct {
			Status   string `json:"status"`
			Playlist struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Entry []struct {
					ID       string `json:"id"`
					Title    string `json:"title"`
					Artist   string `json:"artist"`
					Album    string `json:"album"`
					AlbumID  string `json:"albumId"`
					Duration int    `json:"duration"`
					CoverArt string `json:"coverArt"`
					Track    int    `json:"track"`
				} `json:"entry"`
			} `json:"playlist"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return PlaylistDetail{}, err
	}
	if payload.SubsonicResponse.Status != "ok" {
		msg := "subsonic getPlaylist failed"
		if payload.SubsonicResponse.Error != nil {
			msg = payload.SubsonicResponse.Error.Message
		}
		return PlaylistDetail{}, fmt.Errorf("%s", msg)
	}
	pl := PlaylistDetail{
		ID:   payload.SubsonicResponse.Playlist.ID,
		Name: payload.SubsonicResponse.Playlist.Name,
	}
	for _, entry := range payload.SubsonicResponse.Playlist.Entry {
		pl.Songs = append(pl.Songs, PlaylistSong{
			ID:       entry.ID,
			Title:    entry.Title,
			Artist:   entry.Artist,
			Album:    entry.Album,
			AlbumID:  entry.AlbumID,
			Duration: entry.Duration,
			CoverArt: entry.CoverArt,
			Track:    entry.Track,
		})
	}
	return pl, nil
}

// GetAlbum fetches an album and its songs from the Subsonic server.
func (s *Client) GetAlbum(id string) (AlbumDetail, error) {
	body, err := s.getJSON("getAlbum.view", url.Values{"id": {id}})
	if err != nil {
		return AlbumDetail{}, err
	}
	var payload struct {
		SubsonicResponse struct {
			Status string `json:"status"`
			Album  struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Artist string `json:"artist"`
				Song   []struct {
					ID       string `json:"id"`
					Title    string `json:"title"`
					Artist   string `json:"artist"`
					Album    string `json:"album"`
					AlbumID  string `json:"albumId"`
					Duration int    `json:"duration"`
					CoverArt string `json:"coverArt"`
					Track    int    `json:"track"`
				} `json:"song"`
			} `json:"album"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return AlbumDetail{}, err
	}
	if payload.SubsonicResponse.Status != "ok" {
		msg := "subsonic getAlbum failed"
		if payload.SubsonicResponse.Error != nil {
			msg = payload.SubsonicResponse.Error.Message
		}
		return AlbumDetail{}, fmt.Errorf("%s", msg)
	}
	album := AlbumDetail{
		ID:     payload.SubsonicResponse.Album.ID,
		Name:   payload.SubsonicResponse.Album.Name,
		Artist: payload.SubsonicResponse.Album.Artist,
	}
	for _, entry := range payload.SubsonicResponse.Album.Song {
		album.Songs = append(album.Songs, PlaylistSong{
			ID:       entry.ID,
			Title:    entry.Title,
			Artist:   entry.Artist,
			Album:    entry.Album,
			AlbumID:  entry.AlbumID,
			Duration: entry.Duration,
			CoverArt: entry.CoverArt,
			Track:    entry.Track,
		})
	}
	return album, nil
}

// Stream fetches the raw audio bytes for a track. The caller owns the returned
// reader and must close it. contentType is the upstream MIME type when known.
func (s *Client) Stream(id string) (body io.ReadCloser, contentType string, err error) {
	query := url.Values{"id": {id}}
	merged := s.InjectAuth(query)
	target := s.ServerURL + "/rest/stream.view?" + merged.Encode()
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := s.stream.Do(req)
	if err != nil {
		return nil, "", httputil.SanitizeErrorURL(err)
	}
	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, "", fmt.Errorf("subsonic stream %s: status %d %s", id, resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return resp.Body, resp.Header.Get("Content-Type"), nil
}

// CoverArt fetches cover art bytes for an album, artist, or track id.
// The caller owns the returned reader and must close it.
func (s *Client) CoverArt(id string, size int) (body io.ReadCloser, contentType string, err error) {
	query := url.Values{"id": {id}}
	if size > 0 {
		query.Set("size", fmt.Sprintf("%d", size))
	}
	merged := s.InjectAuth(query)
	target := s.ServerURL + "/rest/getCoverArt.view?" + merged.Encode()
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := s.stream.Do(req)
	if err != nil {
		return nil, "", httputil.SanitizeErrorURL(err)
	}
	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, "", fmt.Errorf("subsonic cover %s: status %d %s", id, resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return resp.Body, resp.Header.Get("Content-Type"), nil
}

func (s *Client) LibraryStats() (LibraryStatsResponse, error) {
	var stats LibraryStatsResponse

	if democatalog.IsFakeURL(s.ServerURL) {
		songs, albums, artists := democatalog.Stats()
		return LibraryStatsResponse{
			SongCount:   songs,
			AlbumCount:  albums,
			ArtistCount: artists,
			FolderCount: 1,
			Scanning:    false,
			LastScan:    "2025-08-01T00:00:00Z",
		}, nil
	}

	scanBody, err := s.getJSON("getScanStatus.view", nil)
	if err != nil {
		return stats, err
	}

	var scanPayload struct {
		SubsonicResponse struct {
			Status     string `json:"status"`
			ScanStatus struct {
				Scanning    bool   `json:"scanning"`
				Count       int    `json:"count"`
				FolderCount int    `json:"folderCount"`
				LastScan    string `json:"lastScan"`
			} `json:"scanStatus"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(scanBody, &scanPayload); err != nil {
		return stats, err
	}
	if scanPayload.SubsonicResponse.Status != "ok" {
		msg := "subsonic getScanStatus failed"
		if scanPayload.SubsonicResponse.Error != nil {
			msg = scanPayload.SubsonicResponse.Error.Message
		}
		return stats, fmt.Errorf("%s", msg)
	}

	stats.SongCount = scanPayload.SubsonicResponse.ScanStatus.Count
	stats.FolderCount = scanPayload.SubsonicResponse.ScanStatus.FolderCount
	stats.Scanning = scanPayload.SubsonicResponse.ScanStatus.Scanning
	stats.LastScan = scanPayload.SubsonicResponse.ScanStatus.LastScan

	artistCount, albumCount, countErr := s.artistAlbumCounts()
	if countErr == nil {
		stats.ArtistCount = artistCount
		stats.AlbumCount = albumCount
	}

	return stats, nil
}

func (s *Client) artistAlbumCounts() (artists, albums int, err error) {
	body, err := s.getJSON("getArtists.view", nil)
	if err != nil {
		return 0, 0, err
	}

	var payload struct {
		SubsonicResponse struct {
			Status  string `json:"status"`
			Artists struct {
				Index []struct {
					Artist []struct {
						AlbumCount int `json:"albumCount"`
					} `json:"artist"`
				} `json:"index"`
			} `json:"artists"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, 0, err
	}
	if payload.SubsonicResponse.Status != "ok" {
		msg := "subsonic getArtists failed"
		if payload.SubsonicResponse.Error != nil {
			msg = payload.SubsonicResponse.Error.Message
		}
		return 0, 0, fmt.Errorf("%s", msg)
	}

	for _, index := range payload.SubsonicResponse.Artists.Index {
		for _, artist := range index.Artist {
			artists++
			albums += artist.AlbumCount
		}
	}
	return artists, albums, nil
}

func CacheTTL(path string) time.Duration {
	switch {
	case strings.Contains(path, "getCoverArt"):
		return 24 * time.Hour
	case strings.Contains(path, "ping"):
		return 15 * time.Minute
	case strings.Contains(path, "getIndexes"), strings.Contains(path, "getArtists"):
		return 10 * time.Minute
	case strings.Contains(path, "getScanStatus"):
		return 10 * time.Minute
	case strings.Contains(path, "getAlbum"), strings.Contains(path, "getArtist"):
		return 5 * time.Minute
	case strings.Contains(path, "search"):
		return 90 * time.Second
	case strings.Contains(path, "getAlbumList"), strings.Contains(path, "getRandomSongs"):
		return 5 * time.Minute
	case strings.Contains(path, "getGenres"):
		return 10 * time.Minute
	default:
		return 2 * time.Minute
	}
}

func ShouldCacheRequest(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	if strings.Contains(path, "stream") || strings.Contains(path, "download") {
		return false
	}
	if strings.Contains(path, "getCoverArt") {
		return false
	}
	if strings.Contains(path, "scrobble") {
		return false
	}
	if strings.Contains(path, "getPlaylists") {
		return true
	}
	if strings.Contains(path, "getPlaylist") {
		return false
	}
	return true
}
