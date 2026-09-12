// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package sharing

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"melovian/internal/api/apishared"
	"melovian/internal/api/library"
	"melovian/internal/api/realtime"
	"melovian/internal/appconfig"
	"melovian/internal/brand"
	"melovian/internal/httputil"
	"melovian/internal/localmusic"
	"melovian/internal/store"
	"melovian/internal/subsonic"
)

// Handler serves the share, listen-together party, and smart playlist
// routes.
type Handler struct {
	cfg            appconfig.Config
	auth           *store.AuthStore
	shares         *store.ShareStore
	instances      *store.InstanceStore
	localLibraries *store.LocalLibraryStore
	localTracks    *store.LocalTrackStore
	listen         *store.ListenStore
	preferences    *store.PreferencesStore
	resolver       *apishared.Resolver
	devices        *realtime.DeviceRegistry
	limiter        *apishared.RateLimiter
	library        *library.Handler
	notify         func(store.CreateNotificationInput) (store.Notification, error)
}

type Deps struct {
	Config      appconfig.Config
	Auth        *store.AuthStore
	Shares      *store.ShareStore
	Instances   *store.InstanceStore
	Libraries   *store.LocalLibraryStore
	Tracks      *store.LocalTrackStore
	Listen      *store.ListenStore
	Preferences *store.PreferencesStore
	Resolver    *apishared.Resolver
	Devices     *realtime.DeviceRegistry
	Limiter     *apishared.RateLimiter
	Library     *library.Handler
	Notify      func(store.CreateNotificationInput) (store.Notification, error)
}

func New(d Deps) *Handler {
	return &Handler{
		cfg:            d.Config,
		auth:           d.Auth,
		shares:         d.Shares,
		instances:      d.Instances,
		localLibraries: d.Libraries,
		localTracks:    d.Tracks,
		listen:         d.Listen,
		preferences:    d.Preferences,
		resolver:       d.Resolver,
		devices:        d.Devices,
		limiter:        d.Limiter,
		library:        d.Library,
		notify:         d.Notify,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	h.registerSmartPlaylistRoutes(mux)
	h.registerShareRoutes(mux)
	h.registerPartyRoutes(mux)
}

func (h *Handler) localLibraryEnabled() bool {
	return h.cfg.LocalLibraryEffective().Enabled
}

func (h *Handler) registerShareRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/music/shares", h.handleListShares)
	mux.HandleFunc("GET /api/music/shares/inbox", h.handleListShareInbox)
	mux.HandleFunc("POST /api/music/shares", h.handleCreateShare)
	mux.HandleFunc("DELETE /api/music/shares/{id}", h.handleDeleteShare)
}

func (h *Handler) HandlePublicShare(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/s/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	token := parts[0]

	share, err := h.shares.GetByToken(token)
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
		h.handleShareUnlock(w, r, share)
		return
	}

	if !h.ShareAccessAllowed(r, share) {
		h.writeShareAccessDenied(w, share)
		return
	}
	_ = h.shares.TouchVisit(token)

	if len(parts) >= 4 && parts[1] == "tracks" {
		trackID := parts[2]
		action := "stream"
		if len(parts) >= 4 {
			action = parts[3]
		}
		switch action {
		case "stream":
			h.serveShareTrack(w, r, share, trackID, false)
		case "download":
			h.serveShareTrack(w, r, share, trackID, true)
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
				if songs, err := h.shareSongs(share); err == nil && len(songs) > 0 {
					trackID = songs[0].ID
				}
			}
		}
		h.serveShareTrack(w, r, share, trackID, false)
	case "cover":
		h.serveShareCover(w, r, share)
	case "":
		h.writeShareDetail(w, r, share)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleShareUnlock(w http.ResponseWriter, r *http.Request, share store.Share) {
	if share.AccessMode != store.ShareAccessPassword {
		httputil.WriteError(w, http.StatusBadRequest, "not_password_share", "share is not password protected")
		return
	}
	limitKeys := []string{"share|" + share.Token}
	if addr, ok := apishared.ClientIP(r, h.cfg.TrustProxy); ok {
		limitKeys = append(limitKeys, "share|"+share.Token+"|"+addr.String())
	}
	if h.limiter != nil && h.limiter.Blocked(limitKeys...) {
		apishared.WriteRateLimited(w, h.limiter.RetryAfterSeconds(limitKeys...))
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if !h.shares.CheckPassword(share, req.Password) {
		if h.limiter != nil {
			h.limiter.Record(limitKeys...)
		}
		httputil.WriteError(w, http.StatusUnauthorized, "invalid_password", "invalid password")
		return
	}
	if h.limiter != nil {
		h.limiter.Reset(limitKeys...)
	}
	expires := time.Now().Add(24 * time.Hour)
	apishared.SetHTTPOnlyCookie(w, r, shareUnlockCookieName(share.Token), shareUnlockCookieValue(h.cfg.AuthSecret, share), "/s/"+share.Token, int(24*time.Hour/time.Second), expires)
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) handleListShares(w http.ResponseWriter, r *http.Request) {
	userID := shareOwnerUserID(apishared.UserIDFromContext(r.Context()))
	items, err := h.shares.ListForUser(userID)
	if err != nil {
		httputil.WriteInternalError(w, r, "handleListShares", err)
		return
	}
	out := make([]map[string]any, len(items))
	for i, item := range items {
		out[i] = h.shareJSON(item)
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) handleListShareInbox(w http.ResponseWriter, r *http.Request) {
	userID := apishared.UserIDFromContext(r.Context())
	if userID == "" || !h.cfg.AuthEnabled() {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{"items": []any{}})
		return
	}
	items, err := h.shares.ListInbox(userID)
	if err != nil {
		httputil.WriteInternalError(w, r, "handleListShareInbox", err)
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if item.Expired() {
			continue
		}
		detail, err := h.shareDetailPayload(item, false)
		if err != nil {
			out = append(out, h.shareJSON(item))
			continue
		}
		out = append(out, detail)
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) handleCreateShare(w http.ResponseWriter, r *http.Request) {
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
		if !h.cfg.AuthEnabled() {
			httputil.WriteError(w, http.StatusBadRequest, "auth_required", "username sharing requires auth")
			return
		}
		accessMode = store.ShareAccessRestricted
	}

	var recipientIDs []string
	if accessMode == store.ShareAccessRestricted {
		for _, name := range req.Usernames {
			user, err := h.auth.GetUserByUsername(name)
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
		instanceID = apishared.InstanceIDFromContext(r.Context())
	}
	ownerID := shareOwnerUserID(apishared.UserIDFromContext(r.Context()))
	ownerAuthID := shareOwnerAuthID(ownerID)
	if instanceID != "" && !h.shareInstanceAllowed(ownerAuthID, instanceID) {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_instance", "instance not found")
		return
	}

	ownerScope := apishared.ResolveProgressUserID(r.Context())
	probe := store.Share{
		UserID:       ownerID,
		ResourceType: resourceType,
		ResourceID:   strings.TrimSpace(req.ResourceID),
		InstanceID:   instanceID,
		OwnerScope:   ownerScope,
	}
	if _, _, err := h.shareSongsWithTitle(probe); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "resource_unavailable", "share resource not found or inaccessible")
		return
	}

	share, err := h.shares.Create(store.CreateShareInput{
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
		h.notifyShareRecipients(ownerID, share)
	}
	httputil.WriteJSON(w, http.StatusCreated, h.shareJSON(share))
}

func (h *Handler) notifyShareRecipients(ownerID string, share store.Share) {
	ownerName := "Someone"
	if owner, err := h.auth.GetUser(ownerID); err == nil && owner.Username != "" {
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
		_, _ = h.notify(store.CreateNotificationInput{
			UserID:  recipientID,
			Kind:    store.NotificationShareReceived,
			Title:   ownerName + " shared with you",
			Body:    label,
			Href:    href,
			Payload: string(payload),
		})
	}
}

func (h *Handler) handleDeleteShare(w http.ResponseWriter, r *http.Request) {
	userID := shareOwnerUserID(apishared.UserIDFromContext(r.Context()))
	if err := h.shares.Delete(userID, r.PathValue("id")); err != nil {
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

func (h *Handler) writeShareDetail(w http.ResponseWriter, r *http.Request, share store.Share) {
	payload, err := h.shareDetailPayload(share, true)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, payload)
}

func (h *Handler) shareDetailPayload(share store.Share, includeTracks bool) (map[string]any, error) {
	out := h.shareJSON(share)
	out["accessMode"] = share.AccessMode
	title := share.Description
	songs := []shareSongView{}
	if includeTracks {
		var err error
		songs, title, err = h.shareSongsWithTitle(share)
		if err != nil {
			return nil, err
		}
	} else {
		if t, err := h.shareTitle(share); err == nil {
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

func (h *Handler) shareSongs(share store.Share) ([]shareSongView, error) {
	songs, _, err := h.shareSongsWithTitle(share)
	return songs, err
}

func (h *Handler) shareTitle(share store.Share) (string, error) {
	_, title, err := h.shareSongsWithTitle(share)
	return title, err
}

func (h *Handler) shareSongsWithTitle(share store.Share) ([]shareSongView, string, error) {
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
		if pl, err := h.listen.GetPlaylist(scope, share.ResourceID); err == nil {
			songs := make([]shareSongView, 0, len(pl.Tracks))
			for _, t := range pl.Tracks {
				if strings.HasPrefix(t.TrackID, "trk_") && !h.shareOwnerMayAccessLocalTrack(share, t.TrackID) {
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
		client := h.subsonicClientForShare(share)
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
			catalog, err := h.localCatalogForShare(share)
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
		client := h.subsonicClientForShare(share)
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
			if !h.shareOwnerMayAccessLocalTrack(share, share.ResourceID) {
				return nil, "", sql.ErrNoRows
			}
			track, err := h.localTracks.GetByID(share.ResourceID)
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
		client := h.subsonicClientForShare(share)
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

func (h *Handler) serveShareTrack(w http.ResponseWriter, r *http.Request, share store.Share, trackID string, asDownload bool) {
	trackID = strings.TrimSpace(trackID)
	if trackID == "" {
		http.NotFound(w, r)
		return
	}
	if share.ResourceType == "playlist" || share.ResourceType == "album" {
		songs, err := h.shareSongs(share)
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
		if !h.shareOwnerMayAccessLocalTrack(share, trackID) {
			httputil.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
			return
		}
		h.library.ServeLocalTrackFile(w, r, trackID, asDownload)
		return
	}

	client := h.subsonicClientForShare(share)
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
		if songs, err := h.shareSongs(share); err == nil {
			for _, song := range songs {
				if song.ID == trackID {
					title = song.Title
					artist = song.Artist
					break
				}
			}
		}
		apishared.SetAttachmentFilename(w, apishared.DownloadExportFilename(title, artist, contentType))
	}
	_, _ = io.Copy(w, io.LimitReader(body, apishared.MaxDownloadBytes))
}

func (h *Handler) shareOwnerMayAccessLocalTrack(share store.Share, trackID string) bool {
	track, err := h.localTracks.GetByID(trackID)
	if err != nil {
		return false
	}
	ownerID := shareOwnerAuthID(share.UserID)
	if ownerID != "" {
		_, err := h.localLibraries.GetForUser(ownerID, track.LibraryID)
		return err == nil
	}
	lib, err := h.localLibraries.Get(track.LibraryID)
	if err != nil {
		return false
	}
	// Unscoped desktop owner may only serve libraries that are not bound to another account.
	return lib.UserID == ""
}

func (h *Handler) serveShareCover(w http.ResponseWriter, r *http.Request, share store.Share) {
	coverID := r.URL.Query().Get("id")
	if coverID == "" {
		coverID = share.ResourceID
	}
	if !h.ShareCoverAllowed(share, coverID) {
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(coverID, "trk_") || strings.HasPrefix(coverID, "alb_") || strings.HasPrefix(share.ResourceID, "pl_") {
		catalog, err := h.localCatalogForShare(share)
		if err == nil {
			data, mime, ok := h.localCoverDataForOwner(share, catalog, coverID)
			if ok {
				w.Header().Set("Content-Type", mime)
				_, _ = w.Write(data) //#nosec G705 -- cover bytes with explicit image content type
				return
			}
		}
	}
	client := h.subsonicClientForShare(share)
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

func (h *Handler) ShareCoverAllowed(share store.Share, coverID string) bool {
	coverID = strings.TrimSpace(coverID)
	if coverID == "" {
		return false
	}
	if coverID == share.ResourceID {
		return true
	}
	songs, err := h.shareSongs(share)
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

func (h *Handler) localCoverDataForOwner(share store.Share, catalog localmusic.Catalog, id string) ([]byte, string, bool) {
	trackID, ok := catalog.CoverTrackID(id)
	if !ok {
		return nil, "", false
	}
	if !h.shareOwnerMayAccessLocalTrack(share, trackID) {
		return nil, "", false
	}
	track, err := h.localTracks.GetByID(trackID)
	if err != nil {
		return nil, "", false
	}
	ownerID := shareOwnerAuthID(share.UserID)
	lib, err := h.localLibraries.GetForUser(ownerID, track.LibraryID)
	if err != nil {
		return nil, "", false
	}
	return h.library.CoverBytesFromTrackFile(lib, trackID)
}

func (h *Handler) localCatalogForShare(share store.Share) (localmusic.Catalog, error) {
	return h.library.LocalCatalogForUser(shareOwnerAuthID(share.UserID))
}

func (h *Handler) subsonicClientForShare(share store.Share) *subsonic.Client {
	ownerAuthID := shareOwnerAuthID(share.UserID)
	instanceID := strings.TrimSpace(share.InstanceID)
	if instanceID != "" && h.shareInstanceAllowed(ownerAuthID, instanceID) {
		inst, err := h.instances.Get(instanceID)
		if err == nil {
			return h.resolver.CachedClient(instanceID, inst.ServerURL, inst.Username, inst.Password)
		}
	}
	if ownerAuthID != "" {
		activeID, err := h.instances.GetActiveIDForUser(ownerAuthID)
		if err == nil && activeID != "" {
			inst, err := h.instances.GetForUser(ownerAuthID, activeID)
			if err == nil {
				return h.resolver.CachedClient(activeID, inst.ServerURL, inst.Username, inst.Password)
			}
		}
		return subsonic.NewClient("", "", "")
	}
	client := h.resolver.Active()
	if client == nil {
		return subsonic.NewClient("", "", "")
	}
	return client
}

func (h *Handler) shareInstanceAllowed(ownerAuthID, instanceID string) bool {
	inst, err := h.instances.Get(instanceID)
	if err != nil {
		return false
	}
	if ownerAuthID == "" {
		return inst.UserID == ""
	}
	return inst.UserID == ownerAuthID
}

func (h *Handler) ShareAccessAllowed(r *http.Request, share store.Share) bool {
	switch share.AccessMode {
	case store.ShareAccessPublic:
		return true
	case store.ShareAccessPassword:
		cookie, err := r.Cookie(shareUnlockCookieName(share.Token))
		if err != nil {
			return false
		}
		return hmac.Equal([]byte(cookie.Value), []byte(shareUnlockCookieValue(h.cfg.AuthSecret, share)))
	case store.ShareAccessRestricted:
		userID := h.sessionUserID(r)
		if userID == "" {
			return false
		}
		if userID == shareOwnerAuthID(share.UserID) || userID == share.UserID {
			return true
		}
		return h.shares.IsRecipient(share, userID)
	default:
		return false
	}
}

func (h *Handler) writeShareAccessDenied(w http.ResponseWriter, share store.Share) {
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

func (h *Handler) sessionUserID(r *http.Request) string {
	if !h.cfg.AuthEnabled() || h.auth == nil {
		return ""
	}
	token := apishared.SessionTokenFromRequest(r)
	if token == "" {
		return ""
	}
	userID, err := h.auth.UserIDFromToken(token)
	if err != nil {
		return ""
	}
	return userID
}

func (h *Handler) shareJSON(share store.Share) map[string]any {
	base := apishared.PublicBaseURL(h.cfg)
	url := strings.TrimRight(base, "/") + "/share/" + share.Token
	usernames := make([]string, 0, len(share.RecipientIDs))
	if h.auth != nil {
		for _, id := range share.RecipientIDs {
			if user, err := h.auth.GetUser(id); err == nil {
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
