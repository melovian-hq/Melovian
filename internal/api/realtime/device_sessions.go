// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package realtime

import (
	"strings"
)

func (r *DeviceRegistry) CreateSession(client *WSClient) string {
	r.mu.Lock()
	dev := r.devices[deviceKey(client.ScopeKey(), client.DeviceID())]
	if dev == nil || dev.client != client {
		r.mu.Unlock()
		return ""
	}
	if dev.sessionID != "" {
		id := dev.sessionID
		r.mu.Unlock()
		return id
	}
	id := "lt-" + client.DeviceID()
	key := deviceKey(client.ScopeKey(), client.DeviceID())
	memberUsers, memberKeys := newMemberMaps()
	memberKeys[key] = struct{}{}
	if client.UserID != "" {
		memberUsers[client.UserID] = struct{}{}
	}
	r.sessions[id] = &ListenSession{
		ID:            id,
		HostID:        client.DeviceID(),
		Scope:         client.ScopeKey(),
		HostUserID:    client.UserID,
		MemberUserIDs: memberUsers,
		MemberKeys:    memberKeys,
	}
	dev.sessionID = id
	dev.isHost = true
	dev.info.SessionID = id
	dev.info.IsHost = true
	r.setActiveLocked(client.ScopeKey(), client.DeviceID())
	scope := client.ScopeKey()
	r.mu.Unlock()
	r.broadcastDevices(scope)
	r.broadcastToScope(scope, Event{
		Type: EventSessionUpdated,
		Payload: map[string]any{
			"sessionId": id,
			"hostId":    client.DeviceID(),
			"action":    "created",
		},
	})
	return id
}

func (r *DeviceRegistry) JoinSession(client *WSClient, sessionID string) bool {
	r.mu.Lock()
	session := r.sessions[sessionID]
	if session == nil || session.Scope != client.ScopeKey() {
		r.mu.Unlock()
		return false
	}
	dev := r.devices[deviceKey(client.ScopeKey(), client.DeviceID())]
	if dev == nil || dev.client != client {
		r.mu.Unlock()
		return false
	}
	key := deviceKey(client.ScopeKey(), client.DeviceID())
	r.addMemberLocked(session, key, client.UserID)
	dev.sessionID = sessionID
	dev.isHost = false
	dev.info.SessionID = sessionID
	dev.info.IsHost = false
	hostID := session.HostID
	snap := r.hostPlaybackLocked(session)
	scope := client.ScopeKey()
	partyKeys := copyStringSet(session.MemberKeys)
	partyCross := session.CrossUser
	name, username := r.memberLabelsLocked(scope, client.DeviceID())
	r.mu.Unlock()
	r.broadcastDevicesTargets(scope, partyKeys, partyCross)
	r.broadcastSessionTargets(scope, partyKeys, partyCross, Event{
		Type: EventSessionUpdated,
		Payload: map[string]any{
			"sessionId": sessionID,
			"hostId":    hostID,
			"action":    "joined",
			"deviceId":  client.DeviceID(),
			"name":      name,
			"username":  username,
			"state":     snap,
		},
	})
	return true
}

// InviteToSession adds a same-scope device to the caller's hosted listen session.
func (r *DeviceRegistry) InviteToSession(client *WSClient, targetID string) string {
	targetID = strings.TrimSpace(targetID)
	if targetID == "" {
		return "missing_target"
	}
	r.mu.Lock()
	scope := client.ScopeKey()
	host := r.devices[deviceKey(scope, client.DeviceID())]
	if host == nil || host.client != client {
		r.mu.Unlock()
		return "not_registered"
	}
	if targetID == client.DeviceID() {
		r.mu.Unlock()
		return "missing_target"
	}
	if !host.isHost || host.sessionID == "" {
		r.mu.Unlock()
		return "not_hosting"
	}
	session := r.sessions[host.sessionID]
	if session == nil || session.Scope != scope {
		r.mu.Unlock()
		return "session_unavailable"
	}
	target := r.devices[deviceKey(scope, targetID)]
	if target == nil {
		r.mu.Unlock()
		return "target_offline"
	}
	if target.sessionID == host.sessionID {
		r.mu.Unlock()
		return "already_member"
	}
	if target.isHost && target.sessionID != "" {
		r.mu.Unlock()
		return "target_hosting"
	}

	var prevLeftKeys map[string]struct{}
	var prevLeftCross bool
	var prevLeftSessionID string
	if target.sessionID != "" {
		prevLeftSessionID = target.sessionID
		if old := r.sessions[target.sessionID]; old != nil {
			r.removeMemberLocked(old, deviceKey(scope, targetID), target.UserID)
			prevLeftKeys = copyStringSet(old.MemberKeys)
			prevLeftCross = old.CrossUser
		}
		target.sessionID = ""
		target.isHost = false
		target.info.SessionID = ""
		target.info.IsHost = false
	}

	key := deviceKey(scope, targetID)
	r.addMemberLocked(session, key, target.UserID)
	target.sessionID = session.ID
	target.isHost = false
	target.info.SessionID = session.ID
	target.info.IsHost = false
	hostID := session.HostID
	sessionID := session.ID
	snap := r.hostPlaybackLocked(session)
	partyKeys := copyStringSet(session.MemberKeys)
	partyCross := session.CrossUser
	name, username := r.memberLabelsLocked(scope, targetID)
	r.mu.Unlock()

	if prevLeftKeys != nil {
		r.broadcastSessionTargets(scope, prevLeftKeys, prevLeftCross, Event{
			Type: EventSessionUpdated,
			Payload: map[string]any{
				"sessionId": prevLeftSessionID,
				"action":    "left",
				"reason":    "left",
				"deviceId":  targetID,
				"name":      name,
				"username":  username,
			},
		})
		r.broadcastDevicesTargets(scope, prevLeftKeys, prevLeftCross)
	}

	r.broadcastDevicesTargets(scope, partyKeys, partyCross)
	r.broadcastSessionTargets(scope, partyKeys, partyCross, Event{
		Type: EventSessionUpdated,
		Payload: map[string]any{
			"sessionId": sessionID,
			"hostId":    hostID,
			"action":    "joined",
			"deviceId":  targetID,
			"name":      name,
			"username":  username,
			"state":     snap,
		},
	})
	return ""
}

// EnsureInviteToken marks a hosted session as cross-user and returns its invite token.
func (r *DeviceRegistry) EnsureInviteToken(scope, deviceID string) (sessionID, token string, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	dev := r.devices[deviceKey(scope, deviceID)]
	if dev == nil || !dev.isHost || dev.sessionID == "" {
		return "", "", false
	}
	session := r.sessions[dev.sessionID]
	if session == nil {
		return "", "", false
	}
	session.CrossUser = true
	if session.InviteToken == "" {
		token, err := newPartyInviteToken()
		if err != nil {
			return "", "", false
		}
		session.InviteToken = token
	}
	return session.ID, session.InviteToken, true
}

// SessionByInviteToken returns the session and current host playback for an
// invite token without joining. Used for link previews and invite metadata.
func (r *DeviceRegistry) SessionByInviteToken(token string) (ListenSession, *PlaybackSnapshot, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return ListenSession{}, nil, false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.sessions {
		if s.InviteToken != "" && s.InviteToken == token && s.CrossUser {
			return cloneSession(*s), r.hostPlaybackLocked(s), true
		}
	}
	return ListenSession{}, nil, false
}

// JoinSessionByToken joins a listen session via invite token across account scopes.
func (r *DeviceRegistry) JoinSessionByToken(scope, deviceID, userID, token string) (sessionID string, snap *PlaybackSnapshot, ok bool) {
	token = strings.TrimSpace(token)
	if token == "" || deviceID == "" {
		return "", nil, false
	}
	r.mu.Lock()
	var session *ListenSession
	for _, s := range r.sessions {
		if s.InviteToken != "" && s.InviteToken == token {
			session = s
			break
		}
	}
	if session == nil || !session.CrossUser {
		r.mu.Unlock()
		return "", nil, false
	}
	dev := r.devices[deviceKey(scope, deviceID)]
	if dev == nil {
		r.mu.Unlock()
		return "", nil, false
	}
	key := deviceKey(scope, deviceID)
	r.addMemberLocked(session, key, userID)
	dev.sessionID = session.ID
	dev.isHost = false
	dev.info.SessionID = session.ID
	dev.info.IsHost = false
	hostID := session.HostID
	hostScope := session.Scope
	snap = r.hostPlaybackLocked(session)
	partyKeys := copyStringSet(session.MemberKeys)
	sessionID = session.ID
	name, username := r.memberLabelsLocked(scope, deviceID)
	r.mu.Unlock()

	r.broadcastDevicesTargets(hostScope, partyKeys, true)
	r.broadcastSessionTargets(hostScope, partyKeys, true, Event{
		Type: EventSessionUpdated,
		Payload: map[string]any{
			"sessionId": sessionID,
			"hostId":    hostID,
			"action":    "joined",
			"deviceId":  deviceID,
			"name":      name,
			"username":  username,
			"state":     snap,
			"crossUser": true,
		},
	})
	return sessionID, snap, true
}

func (r *DeviceRegistry) LeaveSession(client *WSClient) {
	r.mu.Lock()
	dev := r.devices[deviceKey(client.ScopeKey(), client.DeviceID())]
	if dev == nil || dev.client != client || dev.sessionID == "" {
		r.mu.Unlock()
		return
	}
	sessionID := dev.sessionID
	wasHost := dev.isHost
	scope := client.ScopeKey()
	deviceID := client.DeviceID()
	key := deviceKey(scope, deviceID)
	dev.sessionID = ""
	dev.isHost = false
	dev.info.SessionID = ""
	dev.info.IsHost = false

	name, username := r.memberLabelsLocked(scope, deviceID)
	var endedKeys map[string]struct{}
	var endedCross bool
	var leftKeys map[string]struct{}
	var leftCross bool
	if wasHost {
		endedKeys, endedCross = r.endSessionLocked(sessionID)
	} else if session := r.sessions[sessionID]; session != nil {
		r.removeMemberLocked(session, key, client.UserID)
		leftKeys = copyStringSet(session.MemberKeys)
		leftCross = session.CrossUser
	}
	r.mu.Unlock()

	if wasHost {
		r.broadcastSessionTargets(scope, endedKeys, endedCross, Event{
			Type: EventSessionUpdated,
			Payload: map[string]any{
				"sessionId": sessionID,
				"action":    "ended",
				"reason":    "host_left",
				"deviceId":  deviceID,
				"name":      name,
				"username":  username,
			},
		})
		r.broadcastDevicesTargets(scope, endedKeys, endedCross)
		return
	}
	r.broadcastSessionTargets(scope, leftKeys, leftCross, Event{
		Type: EventSessionUpdated,
		Payload: map[string]any{
			"sessionId": sessionID,
			"action":    "left",
			"reason":    "left",
			"deviceId":  deviceID,
			"name":      name,
			"username":  username,
		},
	})
	r.broadcastDevicesTargets(scope, leftKeys, leftCross)
}
func (r *DeviceRegistry) SessionStatus(sessionID string) (PartyStatus, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	session := r.sessions[sessionID]
	if session == nil {
		return PartyStatus{}, false
	}
	members := make([]PartyMemberInfo, 0, len(session.MemberKeys))
	for key := range session.MemberKeys {
		dev := r.devices[key]
		if dev == nil {
			continue
		}
		members = append(members, PartyMemberInfo{
			DeviceID: dev.info.DeviceID,
			UserID:   dev.UserID,
			Username: dev.info.Username,
			Name:     dev.info.Name,
			IsHost:   dev.isHost,
		})
	}
	return PartyStatus{
		SessionID:  session.ID,
		HostID:     session.HostID,
		HostUserID: session.HostUserID,
		CrossUser:  session.CrossUser,
		Members:    members,
	}, true
}

// sessionTrackAllowlistCap bounds the rolling set of track ids a party
// session remembers. When it fills the set is reset rather than
// evicting one by one, keeping long sessions cheap.
const sessionTrackAllowlistCap = 1024

// rememberTracks records ids the host reported as current or queued so
// members may stream exactly what the party is sharing. Cover art ids
// ride along because members resolve artwork through the same gate.
func (s *ListenSession) rememberTracks(trackID, coverArt string, queueIDs []string) {
	if s.TrackIDs == nil {
		s.TrackIDs = make(map[string]struct{})
	}
	if len(s.TrackIDs) >= sessionTrackAllowlistCap {
		s.TrackIDs = make(map[string]struct{}, sessionTrackAllowlistCap)
	}
	if trackID != "" {
		s.TrackIDs[trackID] = struct{}{}
	}
	if coverArt != "" {
		s.TrackIDs[coverArt] = struct{}{}
	}
	for _, id := range queueIDs {
		if id != "" {
			s.TrackIDs[id] = struct{}{}
		}
	}
}

// SessionAllowsTrack reports whether a party member may fetch trackID
// through the session. The allowlist is the ids the host has reported
// playing or queued, plus whatever the host snapshot holds right now so
// a track queued before this code existed still resolves.
func (r *DeviceRegistry) SessionAllowsTrack(sessionID, trackID string) bool {
	if trackID == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	session := r.sessions[sessionID]
	if session == nil {
		return false
	}
	if _, ok := session.TrackIDs[trackID]; ok {
		return true
	}
	if snap := r.hostPlaybackLocked(session); snap != nil {
		if snap.TrackID == trackID || snap.CoverArt == trackID {
			return true
		}
		for _, id := range snap.QueueIDs {
			if id == trackID {
				return true
			}
		}
	}
	return false
}

func (r *DeviceRegistry) IsSessionMember(sessionID, userID, scope, deviceID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	session := r.sessions[sessionID]
	if session == nil {
		return false
	}
	if deviceID != "" {
		if _, ok := session.MemberKeys[deviceKey(scope, deviceID)]; ok {
			return true
		}
	}
	if userID != "" {
		_, ok := session.MemberUserIDs[userID]
		return ok
	}
	return false
}

func (r *DeviceRegistry) HostedSessionID(scope, deviceID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	dev := r.devices[deviceKey(scope, deviceID)]
	if dev == nil || !dev.isHost {
		return ""
	}
	return dev.sessionID
}

func (r *DeviceRegistry) memberLabelsLocked(scope, deviceID string) (name, username string) {
	dev := r.devices[deviceKey(scope, deviceID)]
	if dev == nil {
		return "", ""
	}
	name = strings.TrimSpace(dev.info.Name)
	username = strings.TrimSpace(dev.info.Username)
	return name, username
}
func (r *DeviceRegistry) addMemberLocked(session *ListenSession, key, userID string) {
	if session.MemberKeys == nil {
		session.MemberKeys = make(map[string]struct{})
	}
	if session.MemberUserIDs == nil {
		session.MemberUserIDs = make(map[string]struct{})
	}
	session.MemberKeys[key] = struct{}{}
	if userID != "" {
		session.MemberUserIDs[userID] = struct{}{}
	}
}

func (r *DeviceRegistry) removeMemberLocked(session *ListenSession, key, userID string) {
	delete(session.MemberKeys, key)
	if userID == "" {
		return
	}
	stillPresent := false
	for memberKey := range session.MemberKeys {
		if dev := r.devices[memberKey]; dev != nil && dev.UserID == userID {
			stillPresent = true
			break
		}
	}
	if !stillPresent {
		delete(session.MemberUserIDs, userID)
	}
}

// endSessionLocked clears a session and returns the former member keys for fan-out.
func (r *DeviceRegistry) endSessionLocked(sessionID string) (memberKeys map[string]struct{}, crossUser bool) {
	session := r.sessions[sessionID]
	if session == nil {
		return nil, false
	}
	memberKeys = copyStringSet(session.MemberKeys)
	crossUser = session.CrossUser
	delete(r.sessions, sessionID)
	for _, dev := range r.devices {
		if dev.sessionID == sessionID {
			dev.sessionID = ""
			dev.isHost = false
			dev.info.SessionID = ""
			dev.info.IsHost = false
		}
	}
	return memberKeys, crossUser
}
