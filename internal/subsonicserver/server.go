// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonicserver

import (
	"context"
	"crypto/md5" //#nosec G501 -- Subsonic token auth requires MD5 per protocol spec
	"encoding/hex"
	"net/http"
	"strings"

	"melovian/internal/brand"
	"melovian/internal/localmusic"
	"melovian/internal/store"
)

const (
	Version      = "1.16.1"
	OpenSubsonic = "1"
)

var (
	ServerName = brand.Name
	ServerType = brand.Slug
)

type CatalogProvider interface {
	Catalog(ctx context.Context, userID string) (localmusic.Catalog, error)
	Track(ctx context.Context, userID, trackID string) (store.LocalTrack, store.LocalLibrary, error)
	Cover(ctx context.Context, userID, id string) ([]byte, string, error)
	StreamPath(ctx context.Context, userID string, track store.LocalTrack, lib store.LocalLibrary) (string, error)
}

type AuthChecker interface {
	AuthRequired() bool
	Authenticate(username, password string) (string, error)
	AuthenticateToken(username, token, salt string) (string, error)
}

type Server struct {
	provider CatalogProvider
	auth     AuthChecker
	enabled  bool
}

func New(provider CatalogProvider, auth AuthChecker, enabled bool) *Server {
	return &Server{
		provider: provider,
		auth:     auth,
		enabled:  enabled,
	}
}

func (s *Server) Enabled() bool {
	return s.enabled && s.provider != nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.Enabled() {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	endpoint := restEndpoint(r.URL.Path)
	if endpoint == "" {
		http.NotFound(w, r)
		return
	}

	userID, ok := s.authenticate(w, r)
	if !ok {
		return
	}
	ctx := context.WithValue(r.Context(), userIDKey{}, userID)

	switch endpoint {
	case "ping":
		s.handlePing(w, r)
	case "getLicense":
		s.handleLicense(w, r)
	case "getArtists":
		s.handleGetArtists(w, r, ctx, userID)
	case "getArtist":
		s.handleGetArtist(w, r, ctx, userID)
	case "getAlbum":
		s.handleGetAlbum(w, r, ctx, userID)
	case "getAlbumList2":
		s.handleGetAlbumList2(w, r, ctx, userID)
	case "getSong":
		s.handleGetSong(w, r, ctx, userID)
	case "getRandomSongs":
		s.handleGetRandomSongs(w, r, ctx, userID)
	case "search3":
		s.handleSearch3(w, r, ctx, userID)
	case "stream":
		s.handleStream(w, r, ctx, userID)
	case "getCoverArt":
		s.handleGetCoverArt(w, r, ctx, userID)
	case "getPlaylists":
		s.handleGetPlaylists(w, r, ctx, userID)
	case "getPlaylist":
		s.handleGetPlaylist(w, r, ctx, userID)
	case "getStarred2":
		s.handleGetStarred2(w, r, ctx, userID)
	case "star":
		s.handleStar(w, r, ctx, userID)
	case "unstar":
		s.handleUnstar(w, r, ctx, userID)
	case "scrobble":
		s.handleScrobble(w, r, ctx, userID)
	case "getGenres":
		s.handleGetGenres(w, r, ctx, userID)
	case "createPlaylist":
		s.handleCreatePlaylist(w, r, ctx, userID)
	case "updatePlaylist":
		s.handleUpdatePlaylist(w, r, ctx, userID)
	case "deletePlaylist":
		s.handleDeletePlaylist(w, r, ctx, userID)
	case "getShares":
		s.handleGetShares(w, r, ctx, userID)
	case "createShare":
		s.handleCreateShare(w, r, ctx, userID)
	case "deleteShare":
		s.handleDeleteShare(w, r, ctx, userID)
	case "getJukeboxStatus":
		s.handleGetJukeboxStatus(w, r, ctx)
	case "jukeboxControl":
		s.handleJukeboxControl(w, r, ctx, userID)
	default:
		writeError(w, r, 70, "unsupported endpoint")
	}
}

type userIDKey struct{}

func restEndpoint(path string) string {
	path = strings.TrimPrefix(path, "/rest/")
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return ""
	}
	if i := strings.Index(path, "."); i >= 0 {
		path = path[:i]
	}
	return strings.TrimSpace(path)
}

func (s *Server) authenticate(w http.ResponseWriter, r *http.Request) (string, bool) {
	if s.auth == nil || !s.auth.AuthRequired() {
		return "", true
	}
	query := r.URL.Query()
	username := strings.TrimSpace(query.Get("u"))
	if username == "" {
		writeError(w, r, 10, "missing authentication")
		return "", false
	}
	password := query.Get("p")
	token := query.Get("t")
	salt := query.Get("s")
	if password != "" {
		userID, err := s.auth.Authenticate(username, password)
		if err != nil {
			writeError(w, r, 40, "wrong username or password")
			return "", false
		}
		return userID, true
	}
	if token != "" && salt != "" {
		userID, err := s.auth.AuthenticateToken(username, token, salt)
		if err != nil {
			writeError(w, r, 40, "wrong username or password")
			return "", false
		}
		return userID, true
	}
	writeError(w, r, 10, "missing authentication")
	return "", false
}

func VerifyToken(password, token, salt string) bool {
	sum := md5.Sum([]byte(password + salt)) //#nosec G401 -- Subsonic API mandated token = md5(password + salt)
	return strings.EqualFold(hex.EncodeToString(sum[:]), token)
}
