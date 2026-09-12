// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"melovian/internal/brand"
	"melovian/internal/httputil"
	"melovian/internal/localmusic"
	"melovian/internal/store"
	"melovian/internal/subsonic"
)

func (s *Server) registerShareRoutes() {
	s.mux.HandleFunc("GET /api/music/shares", s.handleListShares)
	s.mux.HandleFunc("GET /api/music/shares/inbox", s.handleListShareInbox)
	s.mux.HandleFunc("POST /api/music/shares", s.handleCreateShare)
	s.mux.HandleFunc("DELETE /api/music/shares/{id}", s.handleDeleteShare)
}

func (s *Server) handlePublicShare(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/s/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	token := parts[0]

	share, err := s.shares.GetByToken(token)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if share.Expired() {
		httputil.WriteError(w, http.StatusGone, "share_expired", "share expired")
		return
	}

	if len(parts) >= 2 && parts[1] == "unlock" {
		if r.Method != http.MethodPost {
			httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		s.handleShareUnlock(w, r, share)
		return
	}

	if !s.shareAccessAllowed(r, share) {
		s.writeShareAccessDenied(w, share)
		return
	}
	_ = s.shares.TouchVisit(token)

	if len(parts) >= 4 && parts[1] == "tracks" {
		trackID := parts[2]
		action := "stream"
		if len(parts) >= 4 {
			action = parts[3]
		}
		switch action {
		case "stream":
			s.serveShareTrack(w, r, share, trackID, false)
		case "download":
			s.serveShareTrack(w, r, share, trackID, true)
		default:
			http.NotFound(w, r)
		}
		return
	}

	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	switch action {
	case "stream":
		// Legacy single-resource stream (song or first album track).
		trackID := r.URL.Query().Get("trackId")
		if trackID == "" {
			trackID = share.ResourceID
			if share.ResourceType == "album" || strings.HasPrefix(share.ResourceID, "alb_") {
				if songs, err := s.shareSongs(share); err == nil && len(songs) > 0 {
					trackID = songs[0].ID
				}
			}
		}
		s.serveShareTrack(w, r, share, trackID, false)
	case "cover":
		s.serveShareCover(w, r, share)
	case "":
		s.writeShareDetail(w, r, share)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleShareUnlock(w http.ResponseWriter, r *http.Request, share store.Share) {
	if share.AccessMode != store.ShareAccessPassword {
		httputil.WriteError(w, http.StatusBadRequest, "not_password_share", "share is not password protected")
		return
	}
	limitKeys := []string{"share|" + share.Token}
	if addr, ok := clientIP(r, s.cfg.TrustProxy); ok {
		limitKeys = append(limitKeys, "share|"+share.Token+"|"+addr.String())
	}
	if s.shareLimiter != nil && s.shareLimiter.blocked(limitKeys...) {
		writeRateLimited(w, s.shareLimiter.retryAfterSeconds(limitKeys...))
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if !s.shares.CheckPassword(share, req.Password) {
		if s.shareLimiter != nil {
			s.shareLimiter.record(limitKeys...)
		}
		httputil.WriteError(w, http.StatusUnauthorized, "invalid_password", "invalid password")
		return
	}
	if s.shareLimiter != nil {
		s.shareLimiter.reset(limitKeys...)
	}
	expires := time.Now().Add(24 * time.Hour)
	setHTTPOnlyCookie(w, r, shareUnlockCookieName(share.Token), shareUnlockCookieValue(s.cfg.AuthSecret, share), "/s/"+share.Token, int(24*time.Hour/time.Second), expires)
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleListShares(w http.ResponseWriter, r *http.Request) {
	userID := shareOwnerUserID(UserIDFromContext(r.Context()))
	items, err := s.shares.ListForUser(userID)
	if err != nil {
		httputil.WriteInternalError(w, r, "handleListShares", err)
		return
	}
	out := make([]map[string]any, len(items))
	for i, item := range items {
		out[i] = s.shareJSON(item)
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) handleListShareInbox(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" || !s.cfg.AuthEnabled() {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{"items": []any{}})
		return
	}
	items, err := s.shares.ListInbox(userID)
	if err != nil {
		httputil.WriteInternalError(w, r, "handleListShareInbox", err)
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if item.Expired() {
			continue
		}
		detail, err := s.shareDetailPayload(item, false)
		if err != nil {
			out = append(out, s.shareJSON(item))
			continue
		}
		out = append(out, detail)
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) handleCreateShare(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourceType string   `json:"resourceType"`
		ResourceID   string   `json:"resourceId"`
		Description  string   `json:"description"`
		ExpiresInSec int      `json:"expiresInSec"`
		AccessMode   string   `json:"accessMode"`
		Password     string   `json:"password"`
		Usernames    []string `json:"usernames"`
		InstanceID   string   `json:"instanceId"`
	}
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	resourceType := strings.ToLower(strings.TrimSpace(req.ResourceType))
	if resourceType == "" {
		if strings.HasPrefix(req.ResourceID, "alb_") {
			resourceType = "album"
		} else if strings.HasPrefix(req.ResourceID, "pl") {
			resourceType = "playlist"
		} else {
			resourceType = "song"
		}
	}
	switch resourceType {
	case "playlist", "album", "song":
	default:
		httputil.WriteError(w, http.StatusBadRequest, "invalid_resource_type", "resourceType must be playlist, album, or song")
		return
	}
	accessMode := strings.ToLower(strings.TrimSpace(req.AccessMode))
	if accessMode == "" {
		accessMode = store.ShareAccessPublic
	}

	if accessMode == store.ShareAccessRestricted || len(req.Usernames) > 0 {
		if !s.cfg.AuthEnabled() {
			httputil.WriteError(w, http.StatusBadRequest, "auth_required", "username sharing requires auth")
			return
		}
		accessMode = store.ShareAccessRestricted
	}

	var recipientIDs []string
	if accessMode == store.ShareAccessRestricted {
		for _, name := range req.Usernames {
			user, err := s.auth.GetUserByUsername(name)
			if err != nil {
				httputil.WriteError(w, http.StatusBadRequest, "unknown_user", "unknown user: "+strings.TrimSpace(name))
				return
			}
			recipientIDs = append(recipientIDs, user.ID)
		}
	}

	var expires *time.Time
	if req.ExpiresInSec > 0 {
		t := time.Now().Add(time.Duration(req.ExpiresInSec) * time.Second)
		expires = &t
	}

	instanceID := strings.TrimSpace(req.InstanceID)
	if instanceID == "" {
		instanceID = InstanceIDFromContext(r.Context())
	}
	ownerID := shareOwnerUserID(UserIDFromContext(r.Context()))
	ownerAuthID := shareOwnerAuthID(ownerID)
	if instanceID != "" && !s.shareInstanceAllowed(ownerAuthID, instanceID) {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_instance", "instance not found")
		return
	}

	ownerScope := ResolveProgressUserID(r.Context())
	probe := store.Share{
		UserID:       ownerID,
		ResourceType: resourceType,
		ResourceID:   strings.TrimSpace(req.ResourceID),
		InstanceID:   instanceID,
		OwnerScope:   ownerScope,
	}
	if _, _, err := s.shareSongsWithTitle(probe); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "resource_unavailable", "share resource not found or inaccessible")
		return
	}

	share, err := s.shares.Create(store.CreateShareInput{
		UserID:       ownerID,
		ResourceType: resourceType,
		ResourceID:   req.ResourceID,
		Description:  req.Description,
		ExpiresAt:    expires,
		AccessMode:   accessMode,
		Password:     req.Password,
		InstanceID:   instanceID,
		OwnerScope:   ownerScope,
		RecipientIDs: recipientIDs,
	})
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if accessMode == store.ShareAccessRestricted {
		s.notifyShareRecipients(ownerID, share)
	}
	httputil.WriteJSON(w, http.StatusCreated, s.shareJSON(share))
}

func (s *Server) notifyShareRecipients(ownerID string, share store.Share) {
	ownerName := "Someone"
	if owner, err := s.auth.GetUser(ownerID); err == nil && owner.Username != "" {
		ownerName = owner.Username
	}
	label := strings.TrimSpace(share.Description)
	if label == "" {
		label = share.ResourceType
	}
	payload, _ := json.Marshal(map[string]any{
		"shareId":      share.ID,
		"token":        share.Token,
		"resourceType": share.ResourceType,
		"resourceId":   share.ResourceID,
	})
	href := "/share/" + share.Token
	for _, recipientID := range share.RecipientIDs {
		if recipientID == "" || recipientID == ownerID {
			continue
		}
		_, _ = s.notifyUser(store.CreateNotificationInput{
			UserID:  recipientID,
			Kind:    store.NotificationShareReceived,
			Title:   ownerName + " shared with you",
			Body:    label,
			Href:    href,
			Payload: string(payload),
		})
	}
}

func (s *Server) handleDeleteShare(w http.ResponseWriter, r *http.Request) {
	userID := shareOwnerUserID(UserIDFromContext(r.Context()))
	if err := s.shares.Delete(userID, r.PathValue("id")); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		httputil.WriteInternalError(w, r, "handleDeleteShare", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type shareSongView struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	Album      string `json:"album"`
	AlbumID    string `json:"albumId,omitempty"`
	DurationMs int    `json:"durationMs"`
	CoverArtID string `json:"coverArtId,omitempty"`
}

func (s *Server) writeShareDetail(w http.ResponseWriter, r *http.Request, share store.Share) {
	payload, err := s.shareDetailPayload(share, true)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, payload)
}

func (s *Server) shareDetailPayload(share store.Share, includeTracks bool) (map[string]any, error) {
	out := s.shareJSON(share)
	out["accessMode"] = share.AccessMode
	title := share.Description
	songs := []shareSongView{}
	if includeTracks {
		var err error
		songs, title, err = s.shareSongsWithTitle(share)
		if err != nil {
			return nil, err
		}
	} else {
		if t, err := s.shareTitle(share); err == nil {
			title = t
		}
	}
	if title != "" {
		out["title"] = title
	}
	if includeTracks {
		out["tracks"] = songs
	}
	return out, nil
}

func (s *Server) shareSongs(share store.Share) ([]shareSongView, error) {
	songs, _, err := s.shareSongsWithTitle(share)
	return songs, err
}

func (s *Server) shareTitle(share store.Share) (string, error) {
	_, title, err := s.shareSongsWithTitle(share)
	return title, err
}

func (s *Server) shareSongsWithTitle(share store.Share) ([]shareSongView, string, error) {
	switch share.ResourceType {
	case "playlist":
		scope := share.OwnerScope
		if scope == "" {
			if authID := shareOwnerAuthID(share.UserID); authID != "" {
				scope = "user:" + authID
			} else {
				scope = "local"
			}
		}
		if pl, err := s.listen.GetPlaylist(scope, share.ResourceID); err == nil {
			songs := make([]shareSongView, 0, len(pl.Tracks))
			for _, t := range pl.Tracks {
				if strings.HasPrefix(t.TrackID, "trk_") && !s.shareOwnerMayAccessLocalTrack(share, t.TrackID) {
					continue
				}
				songs = append(songs, shareSongView{
					ID:         t.TrackID,
					Title:      t.TrackTitle,
					Artist:     t.ArtistName,
					Album:      t.AlbumTitle,
					AlbumID:    t.AlbumID,
					DurationMs: t.DurationMs,
					CoverArtID: t.CoverArtID,
				})
			}
			return songs, pl.Name, nil
		}
		client := s.subsonicClientForShare(share)
		if !client.Enabled() {
			return nil, "", errors.New("playlist not found")
		}
		pl, err := client.GetPlaylist(share.ResourceID)
		if err != nil {
			return nil, "", err
		}
		return subsonicSongsToView(pl.Songs), pl.Name, nil
	case "album":
		if strings.HasPrefix(share.ResourceID, "alb_") {
			catalog, err := s.localCatalogForShare(share)
			if err != nil {
				return nil, "", err
			}
			album, songs, ok := catalog.Album(share.ResourceID)
			if !ok {
				return nil, "", sql.ErrNoRows
			}
			out := make([]shareSongView, 0, len(songs))
			for _, song := range songs {
				out = append(out, localSongToView(song))
			}
			return out, album.Name, nil
		}
		client := s.subsonicClientForShare(share)
		if !client.Enabled() {
			return nil, "", errors.New("subsonic not configured for share")
		}
		album, err := client.GetAlbum(share.ResourceID)
		if err != nil {
			return nil, "", err
		}
		return subsonicSongsToView(album.Songs), album.Name, nil
	default:
		if strings.HasPrefix(share.ResourceID, "trk_") {
			if !s.shareOwnerMayAccessLocalTrack(share, share.ResourceID) {
				return nil, "", sql.ErrNoRows
			}
			track, err := s.localTracks.GetByID(share.ResourceID)
			if err != nil {
				return nil, "", err
			}
			return []shareSongView{{
				ID:         track.ID,
				Title:      track.Title,
				Artist:     track.Artist,
				Album:      track.Album,
				DurationMs: track.DurationMs,
				CoverArtID: track.ID,
			}}, track.Title, nil
		}
		client := s.subsonicClientForShare(share)
		if client.Enabled() {
			title := share.Description
			if title == "" {
				title = share.ResourceID
			}
			return []shareSongView{{ID: share.ResourceID, Title: title}}, title, nil
		}
		return nil, "", sql.ErrNoRows
	}
}

func subsonicSongsToView(songs []subsonic.PlaylistSong) []shareSongView {
	out := make([]shareSongView, 0, len(songs))
	for _, song := range songs {
		out = append(out, shareSongView{
			ID:         song.ID,
			Title:      song.Title,
			Artist:     song.Artist,
			Album:      song.Album,
			AlbumID:    song.AlbumID,
			DurationMs: song.Duration * 1000,
			CoverArtID: song.CoverArt,
		})
	}
	return out
}

func localSongToView(song localmusic.Song) shareSongView {
	return shareSongView{
		ID:         song.ID,
		Title:      song.Title,
		Artist:     song.Artist,
		Album:      song.Album,
		AlbumID:    song.AlbumID,
		DurationMs: song.Duration * 1000,
		CoverArtID: song.CoverArt,
	}
}

func (s *Server) serveShareTrack(w http.ResponseWriter, r *http.Request, share store.Share, trackID string, asDownload bool) {
	trackID = strings.TrimSpace(trackID)
	if trackID == "" {
		http.NotFound(w, r)
		return
	}
	if share.ResourceType == "playlist" || share.ResourceType == "album" {
		songs, err := s.shareSongs(share)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		found := false
		for _, song := range songs {
			if song.ID == trackID {
				found = true
				break
			}
		}
		if !found {
			httputil.WriteError(w, http.StatusForbidden, "track_not_in_share", "track not in share")
			return
		}
	} else if trackID != share.ResourceID {
		httputil.WriteError(w, http.StatusForbidden, "track_not_in_share", "track not in share")
		return
	}

	if strings.HasPrefix(trackID, "trk_") {
		if !s.shareOwnerMayAccessLocalTrack(share, trackID) {
			httputil.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
			return
		}
		s.serveLocalTrackFile(w, r, trackID, asDownload)
		return
	}

	client := s.subsonicClientForShare(share)
	if !client.Enabled() {
		httputil.WriteError(w, http.StatusServiceUnavailable, "service_unavailable", "subsonic not configured")
		return
	}
	body, contentType, err := client.Stream(trackID)
	if err != nil {
		httputil.WriteInternalError(w, r, "stream failed:", err)
		return
	}
	defer func() { _ = body.Close() }()
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if asDownload {
		title := trackID
		artist := ""
		if songs, err := s.shareSongs(share); err == nil {
			for _, song := range songs {
				if song.ID == trackID {
					title = song.Title
					artist = song.Artist
					break
				}
			}
		}
		setAttachmentFilename(w, downloadExportFilename(title, artist, contentType))
	}
	_, _ = io.Copy(w, io.LimitReader(body, maxDownloadBytes))
}

func (s *Server) shareOwnerMayAccessLocalTrack(share store.Share, trackID string) bool {
	track, err := s.localTracks.GetByID(trackID)
	if err != nil {
		return false
	}
	ownerID := shareOwnerAuthID(share.UserID)
	if ownerID != "" {
		_, err := s.localLibraries.GetForUser(ownerID, track.LibraryID)
		return err == nil
	}
	lib, err := s.localLibraries.Get(track.LibraryID)
	if err != nil {
		return false
	}
	// Unscoped desktop owner may only serve libraries that are not bound to another account.
	return lib.UserID == ""
}

// callerMayAccessLocalTrack reports whether the request user may read a local track.
// When auth is off, any present library is allowed (single-tenant desktop).
func (s *Server) callerMayAccessLocalTrack(ctx context.Context, trackID string) bool {
	track, err := s.localTracks.GetByID(trackID)
	if err != nil {
		return false
	}
	userID := UserIDFromContext(ctx)
	if userID != "" {
		_, err := s.localLibraries.GetForUser(userID, track.LibraryID)
		return err == nil
	}
	_, err = s.localLibraries.Get(track.LibraryID)
	return err == nil
}

func (s *Server) serveLocalTrackFile(w http.ResponseWriter, r *http.Request, trackID string, asDownload bool) {
	track, err := s.localTracks.GetByID(trackID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	lib, err := s.localLibraries.Get(track.LibraryID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	path, err := localmusic.ResolveTrackPath(lib.Path, track.AbsPath)
	if err != nil {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
		return
	}
	file, err := os.Open(path) //#nosec G304 -- path resolved under library jail
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = file.Close() }()
	contentType := localmusic.ContentTypeForFormat(track.Format)
	w.Header().Set("Content-Type", contentType)
	if asDownload {
		filename := downloadExportFilename(track.Title, track.Artist, contentType)
		if ext := filepath.Ext(path); ext != "" && len(ext) <= 8 {
			base := strings.TrimSuffix(filename, filepath.Ext(filename))
			filename = base + strings.ToLower(ext)
		}
		setAttachmentFilename(w, filename)
	}
	http.ServeContent(w, r, filepath.Base(path), track.UpdatedAt, file)
}

func (s *Server) serveShareCover(w http.ResponseWriter, r *http.Request, share store.Share) {
	coverID := r.URL.Query().Get("id")
	if coverID == "" {
		coverID = share.ResourceID
	}
	if !s.shareCoverAllowed(share, coverID) {
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(coverID, "trk_") || strings.HasPrefix(coverID, "alb_") || strings.HasPrefix(share.ResourceID, "pl_") {
		catalog, err := s.localCatalogForShare(share)
		if err == nil {
			data, mime, ok := s.localCoverDataForOwner(share, catalog, coverID)
			if ok {
				w.Header().Set("Content-Type", mime)
				_, _ = w.Write(data) //#nosec G705 -- cover bytes with explicit image content type
				return
			}
		}
	}
	client := s.subsonicClientForShare(share)
	if !client.Enabled() {
		http.NotFound(w, r)
		return
	}
	body, contentType, err := client.CoverArt(coverID, 300)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = body.Close() }()
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	_, _ = io.Copy(w, io.LimitReader(body, 8<<20))
}

func (s *Server) shareCoverAllowed(share store.Share, coverID string) bool {
	coverID = strings.TrimSpace(coverID)
	if coverID == "" {
		return false
	}
	if coverID == share.ResourceID {
		return true
	}
	songs, err := s.shareSongs(share)
	if err != nil {
		return false
	}
	for _, song := range songs {
		if song.ID == coverID || song.CoverArtID == coverID || song.AlbumID == coverID {
			return true
		}
	}
	return false
}

func (s *Server) localCoverDataForOwner(share store.Share, catalog localmusic.Catalog, id string) ([]byte, string, bool) {
	trackID, ok := catalog.CoverTrackID(id)
	if !ok {
		return nil, "", false
	}
	if !s.shareOwnerMayAccessLocalTrack(share, trackID) {
		return nil, "", false
	}
	track, err := s.localTracks.GetByID(trackID)
	if err != nil {
		return nil, "", false
	}
	ownerID := shareOwnerAuthID(share.UserID)
	lib, err := s.localLibraries.GetForUser(ownerID, track.LibraryID)
	if err != nil {
		return nil, "", false
	}
	return s.coverBytesFromTrackFile(lib, trackID)
}

func (s *Server) localCatalogForShare(share store.Share) (localmusic.Catalog, error) {
	return s.localCatalogForUser(shareOwnerAuthID(share.UserID))
}

func (s *Server) subsonicClientForShare(share store.Share) *subsonic.Client {
	ownerAuthID := shareOwnerAuthID(share.UserID)
	instanceID := strings.TrimSpace(share.InstanceID)
	if instanceID != "" && s.shareInstanceAllowed(ownerAuthID, instanceID) {
		inst, err := s.instances.Get(instanceID)
		if err == nil {
			return s.cachedClient(instanceID, inst.ServerURL, inst.Username, inst.Password)
		}
	}
	if ownerAuthID != "" {
		activeID, err := s.instances.GetActiveIDForUser(ownerAuthID)
		if err == nil && activeID != "" {
			inst, err := s.instances.GetForUser(ownerAuthID, activeID)
			if err == nil {
				return s.cachedClient(activeID, inst.ServerURL, inst.Username, inst.Password)
			}
		}
		return subsonic.NewClient("", "", "")
	}
	s.mu.RLock()
	client := s.subsonic
	s.mu.RUnlock()
	if client == nil {
		return subsonic.NewClient("", "", "")
	}
	return client
}

func (s *Server) shareInstanceAllowed(ownerAuthID, instanceID string) bool {
	inst, err := s.instances.Get(instanceID)
	if err != nil {
		return false
	}
	if ownerAuthID == "" {
		return inst.UserID == ""
	}
	return inst.UserID == ownerAuthID
}

func (s *Server) shareAccessAllowed(r *http.Request, share store.Share) bool {
	switch share.AccessMode {
	case store.ShareAccessPublic:
		return true
	case store.ShareAccessPassword:
		cookie, err := r.Cookie(shareUnlockCookieName(share.Token))
		if err != nil {
			return false
		}
		return hmac.Equal([]byte(cookie.Value), []byte(shareUnlockCookieValue(s.cfg.AuthSecret, share)))
	case store.ShareAccessRestricted:
		userID := s.sessionUserID(r)
		if userID == "" {
			return false
		}
		if userID == shareOwnerAuthID(share.UserID) || userID == share.UserID {
			return true
		}
		return s.shares.IsRecipient(share, userID)
	default:
		return false
	}
}

func (s *Server) writeShareAccessDenied(w http.ResponseWriter, share store.Share) {
	switch share.AccessMode {
	case store.ShareAccessPassword:
		httputil.WriteJSON(w, http.StatusUnauthorized, map[string]any{
			"error":            "password_required",
			"requiresPassword": true,
			"token":            share.Token,
			"accessMode":       share.AccessMode,
		})
	case store.ShareAccessRestricted:
		httputil.WriteJSON(w, http.StatusUnauthorized, map[string]any{
			"error":         "login_required",
			"requiresLogin": true,
			"token":         share.Token,
			"accessMode":    share.AccessMode,
		})
	default:
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
	}
}

func (s *Server) sessionUserID(r *http.Request) string {
	if !s.cfg.AuthEnabled() || s.auth == nil {
		return ""
	}
	token := sessionTokenFromRequest(r)
	if token == "" {
		return ""
	}
	userID, err := s.auth.UserIDFromToken(token)
	if err != nil {
		return ""
	}
	return userID
}

func (s *Server) shareJSON(share store.Share) map[string]any {
	base := s.publicBaseURL()
	url := strings.TrimRight(base, "/") + "/share/" + share.Token
	usernames := make([]string, 0, len(share.RecipientIDs))
	if s.auth != nil {
		for _, id := range share.RecipientIDs {
			if user, err := s.auth.GetUser(id); err == nil {
				usernames = append(usernames, user.Username)
			}
		}
	}
	out := map[string]any{
		"id":           share.ID,
		"token":        share.Token,
		"url":          url,
		"resourceType": share.ResourceType,
		"resourceId":   share.ResourceID,
		"description":  share.Description,
		"visitCount":   share.VisitCount,
		"createdAt":    share.CreatedAt.UTC().Format(time.RFC3339),
		"accessMode":   share.AccessMode,
		"instanceId":   share.InstanceID,
		"usernames":    usernames,
	}
	if share.ExpiresAt != nil {
		out["expiresAt"] = share.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return out
}

func shareOwnerUserID(userID string) string {
	if strings.TrimSpace(userID) == "" {
		return "local"
	}
	return userID
}

func shareOwnerAuthID(userID string) string {
	if after, ok := strings.CutPrefix(userID, "subsonic:"); ok {
		return after
	}
	if userID == "subsonic" || userID == "local" {
		return ""
	}
	return userID
}

func shareUnlockCookieName(token string) string {
	suffix := token
	if len(suffix) > 24 {
		suffix = suffix[:24]
	}
	return brand.Slug + "_share_" + suffix
}

func shareUnlockCookieValue(secret string, share store.Share) string {
	key := secret
	if key == "" {
		key = brand.Slug + "-share"
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(share.Token))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(share.PasswordHash))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Server) publicBaseURL() string {
	if base := strings.TrimSpace(s.cfg.PublicURL); base != "" {
		return normalizePublicBaseURL(base)
	}
	return normalizePublicBaseURL("http://" + s.cfg.ListenAddr)
}

func normalizePublicBaseURL(raw string) string {
	base := strings.TrimRight(strings.TrimSpace(raw), "/")
	if base == "" {
		return ""
	}
	base = strings.Replace(base, "://0.0.0.0", "://127.0.0.1", 1)
	base = strings.Replace(base, "://[::]", "://127.0.0.1", 1)
	if after, ok := strings.CutPrefix(base, "0.0.0.0:"); ok {
		base = "http://127.0.0.1:" + after
	}
	if after, ok := strings.CutPrefix(base, "[::]:"); ok {
		base = "http://127.0.0.1:" + after
	}
	return base
}
