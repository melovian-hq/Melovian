// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package sharing

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"melovian/internal/api/apishared"
	"melovian/internal/api/realtime"
	"melovian/internal/httputil"
	"melovian/internal/localmusic"
	"melovian/internal/store"
	"melovian/internal/subsonic"
)

func (h *Handler) registerPartyRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/party/invite-link", h.handlePartyInviteLink)
	mux.HandleFunc("POST /api/party/invite", h.handlePartyInvite)
	mux.HandleFunc("POST /api/party/join", h.handlePartyJoin)
	mux.HandleFunc("GET /api/party/{sessionId}/status", h.handlePartyStatus)
	mux.HandleFunc("GET /api/party/{sessionId}/stream/{trackId}", h.handlePartyStream)
	mux.HandleFunc("GET /api/party/{sessionId}/cover/{trackId}", h.handlePartyCover)
}

func (h *Handler) partyAuthScope(r *http.Request) (userID, deviceID, scope string, ok bool) {
	userID = apishared.UserIDFromContext(r.Context())
	deviceID = strings.TrimSpace(apishared.DeviceIDFromContext(r.Context()))
	if deviceID == "" {
		deviceID = strings.TrimSpace(r.Header.Get("X-Device-Id"))
	}
	authEnabled := h.auth != nil && h.auth.Enabled()
	if authEnabled && userID == "" {
		return "", "", "", false
	}
	scope = realtime.ResolveDeviceScope(userID, deviceID, authEnabled)
	return userID, deviceID, scope, true
}

func (h *Handler) handlePartyInviteLink(w http.ResponseWriter, r *http.Request) {
	_, deviceID, scope, ok := h.partyAuthScope(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "auth_required", "authentication required")
		return
	}
	if deviceID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "device_required", "X-Device-Id header is required")
		return
	}
	if h.devices.HostedSessionID(scope, deviceID) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "not_hosting", "start listen together before creating an invite link")
		return
	}
	sessionID, token, ok := h.devices.EnsureInviteToken(scope, deviceID)
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "not_hosting", "start listen together before creating an invite link")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"sessionId": sessionID,
		"token":     token,
		"url":       "/listen/" + token,
	})
}

func (h *Handler) handlePartyInvite(w http.ResponseWriter, r *http.Request) {
	userID, deviceID, scope, ok := h.partyAuthScope(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "auth_required", "authentication required")
		return
	}
	if deviceID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "device_required", "X-Device-Id header is required")
		return
	}
	var req struct {
		Username string `json:"username"`
	}
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		httputil.WriteError(w, http.StatusBadRequest, "username_required", "username is required")
		return
	}
	if h.devices.HostedSessionID(scope, deviceID) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "not_hosting", "start listen together before inviting")
		return
	}
	sessionID, token, ok := h.devices.EnsureInviteToken(scope, deviceID)
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "not_hosting", "start listen together before inviting")
		return
	}
	if h.auth == nil {
		httputil.WriteError(w, http.StatusBadRequest, "auth_required", "user invites require auth")
		return
	}
	target, err := h.auth.GetUserByUsername(username)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "unknown_user", "unknown user: "+username)
		return
	}
	if target.ID == userID {
		httputil.WriteError(w, http.StatusBadRequest, "self_invite", "you are already in this party")
		return
	}

	hostName := "Someone"
	if owner, err := h.auth.GetUser(userID); err == nil && owner.Username != "" {
		hostName = owner.Username
	}
	payload, _ := json.Marshal(map[string]any{
		"sessionId": sessionID,
		"token":     token,
	})
	_, _ = h.notify(store.CreateNotificationInput{
		UserID:  target.ID,
		Kind:    store.NotificationPartyInvite,
		Title:   hostName + " invited you to listen together",
		Body:    "Open the invite to join their session.",
		Href:    "/listen/" + token,
		Payload: string(payload),
	})
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"sessionId": sessionID,
		"token":     token,
		"username":  target.Username,
		"userId":    target.ID,
	})
}

func (h *Handler) handlePartyJoin(w http.ResponseWriter, r *http.Request) {
	userID, deviceID, scope, ok := h.partyAuthScope(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "auth_required", "authentication required")
		return
	}
	if deviceID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "device_required", "X-Device-Id header is required")
		return
	}
	var req struct {
		Token string `json:"token"`
	}
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	sessionID, snap, ok := h.devices.JoinSessionByToken(scope, deviceID, userID, req.Token)
	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "session_unavailable", "invite is invalid or the party ended")
		return
	}

	session, found := h.devices.GetSession(sessionID)
	if found && session.HostUserID != "" && session.HostUserID != userID {
		guestName := "A guest"
		if h.auth != nil {
			if u, err := h.auth.GetUser(userID); err == nil && u.Username != "" {
				guestName = u.Username
			}
		}
		payload, _ := json.Marshal(map[string]any{
			"sessionId": sessionID,
			"deviceId":  deviceID,
		})
		_, _ = h.notify(store.CreateNotificationInput{
			UserID:  session.HostUserID,
			Kind:    store.NotificationPartyJoined,
			Title:   guestName + " joined your party",
			Body:    "They are listening with you now.",
			Href:    "/listen/" + session.InviteToken,
			Payload: string(payload),
		})
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"sessionId": sessionID,
		"state":     snap,
	})
}

func (h *Handler) handlePartyStatus(w http.ResponseWriter, r *http.Request) {
	userID, deviceID, scope, ok := h.partyAuthScope(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "auth_required", "authentication required")
		return
	}
	sessionID := r.PathValue("sessionId")
	if !h.devices.IsSessionMember(sessionID, userID, scope, deviceID) {
		httputil.WriteError(w, http.StatusForbidden, "not_member", "not a party member")
		return
	}
	status, found := h.devices.SessionStatus(sessionID)
	if !found {
		http.NotFound(w, r)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, status)
}

func (h *Handler) handlePartyStream(w http.ResponseWriter, r *http.Request) {
	userID, deviceID, scope, ok := h.partyAuthScope(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "auth_required", "authentication required")
		return
	}
	sessionID := r.PathValue("sessionId")
	trackID := strings.TrimSpace(r.PathValue("trackId"))
	if trackID == "" {
		http.NotFound(w, r)
		return
	}
	if !h.devices.IsSessionMember(sessionID, userID, scope, deviceID) {
		httputil.WriteError(w, http.StatusForbidden, "not_member", "not a party member")
		return
	}
	session, found := h.devices.GetSession(sessionID)
	if !found {
		http.NotFound(w, r)
		return
	}

	if strings.HasPrefix(trackID, "trk_") {
		if !h.partyHostMayAccessLocalTrack(session.HostUserID, trackID) {
			httputil.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
			return
		}
		h.library.ServeLocalTrackFile(w, r, trackID, false)
		return
	}

	client := h.subsonicClientForPartyHost(session.HostUserID)
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
	_, _ = io.Copy(w, io.LimitReader(body, apishared.MaxDownloadBytes))
}

func (h *Handler) handlePartyCover(w http.ResponseWriter, r *http.Request) {
	userID, deviceID, scope, ok := h.partyAuthScope(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "auth_required", "authentication required")
		return
	}
	sessionID := r.PathValue("sessionId")
	trackID := strings.TrimSpace(r.PathValue("trackId"))
	if trackID == "" {
		http.NotFound(w, r)
		return
	}
	if !h.devices.IsSessionMember(sessionID, userID, scope, deviceID) {
		httputil.WriteError(w, http.StatusForbidden, "not_member", "not a party member")
		return
	}
	session, found := h.devices.GetSession(sessionID)
	if !found {
		http.NotFound(w, r)
		return
	}

	coverID := trackID
	if strings.HasPrefix(coverID, "trk_") || strings.HasPrefix(coverID, "alb_") {
		catalog, err := h.library.LocalCatalogForUser(session.HostUserID)
		if err == nil {
			data, mime, ok := h.partyLocalCover(session.HostUserID, catalog, coverID)
			if ok {
				w.Header().Set("Content-Type", mime)
				_, _ = w.Write(data) //#nosec G705 -- cover bytes with explicit image content type
				return
			}
		}
	}

	client := h.subsonicClientForPartyHost(session.HostUserID)
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

func (h *Handler) subsonicClientForPartyHost(hostUserID string) *subsonic.Client {
	hostUserID = strings.TrimSpace(hostUserID)
	if hostUserID != "" {
		activeID, err := h.instances.GetActiveIDForUser(hostUserID)
		if err == nil && activeID != "" {
			inst, err := h.instances.GetForUser(hostUserID, activeID)
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

func (h *Handler) partyHostMayAccessLocalTrack(hostUserID, trackID string) bool {
	track, err := h.localTracks.GetByID(trackID)
	if err != nil {
		return false
	}
	if hostUserID != "" {
		_, err := h.localLibraries.GetForUser(hostUserID, track.LibraryID)
		return err == nil
	}
	lib, err := h.localLibraries.Get(track.LibraryID)
	if err != nil {
		return false
	}
	return lib.UserID == ""
}

func (h *Handler) partyLocalCover(hostUserID string, catalog localmusic.Catalog, id string) ([]byte, string, bool) {
	trackID, ok := catalog.CoverTrackID(id)
	if !ok {
		return nil, "", false
	}
	if !h.partyHostMayAccessLocalTrack(hostUserID, trackID) {
		return nil, "", false
	}
	track, err := h.localTracks.GetByID(trackID)
	if err != nil {
		return nil, "", false
	}
	var lib store.LocalLibrary
	if hostUserID != "" {
		lib, err = h.localLibraries.GetForUser(hostUserID, track.LibraryID)
	} else {
		lib, err = h.localLibraries.Get(track.LibraryID)
	}
	if err != nil {
		return nil, "", false
	}
	return h.library.CoverBytesFromTrackFile(lib, trackID)
}
