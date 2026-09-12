// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"melovian/internal/httputil"
	"melovian/internal/localmusic"
	"melovian/internal/store"
	"melovian/internal/subsonic"
)

func (s *Server) registerPartyRoutes() {
	s.mux.HandleFunc("POST /api/party/invite-link", s.handlePartyInviteLink)
	s.mux.HandleFunc("POST /api/party/invite", s.handlePartyInvite)
	s.mux.HandleFunc("POST /api/party/join", s.handlePartyJoin)
	s.mux.HandleFunc("GET /api/party/{sessionId}/status", s.handlePartyStatus)
	s.mux.HandleFunc("GET /api/party/{sessionId}/stream/{trackId}", s.handlePartyStream)
	s.mux.HandleFunc("GET /api/party/{sessionId}/cover/{trackId}", s.handlePartyCover)
}

func (s *Server) partyAuthScope(r *http.Request) (userID, deviceID, scope string, ok bool) {
	userID = UserIDFromContext(r.Context())
	deviceID = strings.TrimSpace(DeviceIDFromContext(r.Context()))
	if deviceID == "" {
		deviceID = strings.TrimSpace(r.Header.Get("X-Device-Id"))
	}
	authEnabled := s.auth != nil && s.auth.Enabled()
	if authEnabled && userID == "" {
		return "", "", "", false
	}
	scope = ResolveDeviceScope(userID, deviceID, authEnabled)
	return userID, deviceID, scope, true
}

func (s *Server) handlePartyInviteLink(w http.ResponseWriter, r *http.Request) {
	_, deviceID, scope, ok := s.partyAuthScope(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "auth_required", "authentication required")
		return
	}
	if deviceID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "device_required", "X-Device-Id header is required")
		return
	}
	if s.devices.HostedSessionID(scope, deviceID) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "not_hosting", "start listen together before creating an invite link")
		return
	}
	sessionID, token, ok := s.devices.EnsureInviteToken(scope, deviceID)
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

func (s *Server) handlePartyInvite(w http.ResponseWriter, r *http.Request) {
	userID, deviceID, scope, ok := s.partyAuthScope(r)
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
	if s.devices.HostedSessionID(scope, deviceID) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "not_hosting", "start listen together before inviting")
		return
	}
	sessionID, token, ok := s.devices.EnsureInviteToken(scope, deviceID)
	if !ok {
		httputil.WriteError(w, http.StatusBadRequest, "not_hosting", "start listen together before inviting")
		return
	}
	if s.auth == nil {
		httputil.WriteError(w, http.StatusBadRequest, "auth_required", "user invites require auth")
		return
	}
	target, err := s.auth.GetUserByUsername(username)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "unknown_user", "unknown user: "+username)
		return
	}
	if target.ID == userID {
		httputil.WriteError(w, http.StatusBadRequest, "self_invite", "you are already in this party")
		return
	}

	hostName := "Someone"
	if owner, err := s.auth.GetUser(userID); err == nil && owner.Username != "" {
		hostName = owner.Username
	}
	payload, _ := json.Marshal(map[string]any{
		"sessionId": sessionID,
		"token":     token,
	})
	_, _ = s.notifyUser(store.CreateNotificationInput{
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

func (s *Server) handlePartyJoin(w http.ResponseWriter, r *http.Request) {
	userID, deviceID, scope, ok := s.partyAuthScope(r)
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
	sessionID, snap, ok := s.devices.JoinSessionByToken(scope, deviceID, userID, req.Token)
	if !ok {
		httputil.WriteError(w, http.StatusNotFound, "session_unavailable", "invite is invalid or the party ended")
		return
	}

	session, found := s.devices.GetSession(sessionID)
	if found && session.HostUserID != "" && session.HostUserID != userID {
		guestName := "A guest"
		if s.auth != nil {
			if u, err := s.auth.GetUser(userID); err == nil && u.Username != "" {
				guestName = u.Username
			}
		}
		payload, _ := json.Marshal(map[string]any{
			"sessionId": sessionID,
			"deviceId":  deviceID,
		})
		_, _ = s.notifyUser(store.CreateNotificationInput{
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

func (s *Server) handlePartyStatus(w http.ResponseWriter, r *http.Request) {
	userID, deviceID, scope, ok := s.partyAuthScope(r)
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "auth_required", "authentication required")
		return
	}
	sessionID := r.PathValue("sessionId")
	if !s.devices.IsSessionMember(sessionID, userID, scope, deviceID) {
		httputil.WriteError(w, http.StatusForbidden, "not_member", "not a party member")
		return
	}
	status, found := s.devices.SessionStatus(sessionID)
	if !found {
		http.NotFound(w, r)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, status)
}

func (s *Server) handlePartyStream(w http.ResponseWriter, r *http.Request) {
	userID, deviceID, scope, ok := s.partyAuthScope(r)
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
	if !s.devices.IsSessionMember(sessionID, userID, scope, deviceID) {
		httputil.WriteError(w, http.StatusForbidden, "not_member", "not a party member")
		return
	}
	session, found := s.devices.GetSession(sessionID)
	if !found {
		http.NotFound(w, r)
		return
	}

	if strings.HasPrefix(trackID, "trk_") {
		if !s.partyHostMayAccessLocalTrack(session.HostUserID, trackID) {
			httputil.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
			return
		}
		s.serveLocalTrackFile(w, r, trackID, false)
		return
	}

	client := s.subsonicClientForPartyHost(session.HostUserID)
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
	_, _ = io.Copy(w, io.LimitReader(body, maxDownloadBytes))
}

func (s *Server) handlePartyCover(w http.ResponseWriter, r *http.Request) {
	userID, deviceID, scope, ok := s.partyAuthScope(r)
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
	if !s.devices.IsSessionMember(sessionID, userID, scope, deviceID) {
		httputil.WriteError(w, http.StatusForbidden, "not_member", "not a party member")
		return
	}
	session, found := s.devices.GetSession(sessionID)
	if !found {
		http.NotFound(w, r)
		return
	}

	coverID := trackID
	if strings.HasPrefix(coverID, "trk_") || strings.HasPrefix(coverID, "alb_") {
		catalog, err := s.localCatalogForUser(session.HostUserID)
		if err == nil {
			data, mime, ok := s.partyLocalCover(session.HostUserID, catalog, coverID)
			if ok {
				w.Header().Set("Content-Type", mime)
				_, _ = w.Write(data) //#nosec G705 -- cover bytes with explicit image content type
				return
			}
		}
	}

	client := s.subsonicClientForPartyHost(session.HostUserID)
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

func (s *Server) subsonicClientForPartyHost(hostUserID string) *subsonic.Client {
	hostUserID = strings.TrimSpace(hostUserID)
	if hostUserID != "" {
		activeID, err := s.instances.GetActiveIDForUser(hostUserID)
		if err == nil && activeID != "" {
			inst, err := s.instances.GetForUser(hostUserID, activeID)
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

func (s *Server) partyHostMayAccessLocalTrack(hostUserID, trackID string) bool {
	track, err := s.localTracks.GetByID(trackID)
	if err != nil {
		return false
	}
	if hostUserID != "" {
		_, err := s.localLibraries.GetForUser(hostUserID, track.LibraryID)
		return err == nil
	}
	lib, err := s.localLibraries.Get(track.LibraryID)
	if err != nil {
		return false
	}
	return lib.UserID == ""
}

func (s *Server) partyLocalCover(hostUserID string, catalog localmusic.Catalog, id string) ([]byte, string, bool) {
	trackID, ok := catalog.CoverTrackID(id)
	if !ok {
		return nil, "", false
	}
	if !s.partyHostMayAccessLocalTrack(hostUserID, trackID) {
		return nil, "", false
	}
	track, err := s.localTracks.GetByID(trackID)
	if err != nil {
		return nil, "", false
	}
	var lib store.LocalLibrary
	if hostUserID != "" {
		lib, err = s.localLibraries.GetForUser(hostUserID, track.LibraryID)
	} else {
		lib, err = s.localLibraries.Get(track.LibraryID)
	}
	if err != nil {
		return nil, "", false
	}
	return s.coverBytesFromTrackFile(lib, trackID)
}
