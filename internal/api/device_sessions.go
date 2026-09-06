// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"strings"
)

func (r *DeviceRegistry) CreateSession(client *wsClient) string {
	r.mu.Lock()
	dev := r.devices[deviceKey(client.getScopeKey(), client.getDeviceID())]
	if dev == nil || dev.client != client {
		r.mu.Unlock()
		return ""
	}
	if dev.sessionID != "" {
		id := dev.sessionID
		r.mu.Unlock()
		return id
	}
	id := "lt-" + client.getDeviceID()
	key := deviceKey(client.getScopeKey(), client.getDeviceID())
	memberUsers, memberKeys := newMemberMaps()
	memberKeys[key] = struct{}{}
	if client.userID != "" {
		memberUsers[client.userID] = struct{}{}
	}
	r.sessions[id] = &ListenSession{
		ID:            id,
		HostID:        client.getDeviceID(),
		Scope:         client.getScopeKey(),
		HostUserID:    client.userID,
		MemberUserIDs: memberUsers,
		MemberKeys:    memberKeys,
	}
	dev.sessionID = id
	dev.isHost = true
	dev.info.SessionID = id
	dev.info.IsHost = true
	r.setActiveLocked(client.getScopeKey(), client.getDeviceID())
	scope := client.getScopeKey()
	r.mu.Unlock()
	r.broadcastDevices(scope)
	r.broadcastToScope(scope, Event{
		Type: EventSessionUpdated,
		Payload: map[string]any{
			"sessionId": id,
			"hostId":    client.getDeviceID(),
			"action":    "created",
		},
	})
	return id
}

func (r *DeviceRegistry) JoinSession(client *wsClient, sessionID string) bool {
	r.mu.Lock()
	session := r.sessions[sessionID]
	if session == nil || session.Scope != client.getScopeKey() {
		r.mu.Unlock()
		return false
	}
	dev := r.devices[deviceKey(client.getScopeKey(), client.getDeviceID())]
	if dev == nil || dev.client != client {
		r.mu.Unlock()
		return false
	}
	key := deviceKey(client.getScopeKey(), client.getDeviceID())
	r.addMemberLocked(session, key, client.userID)
	dev.sessionID = sessionID
	dev.isHost = false
	dev.info.SessionID = sessionID
	dev.info.IsHost = false
	hostID := session.HostID
	snap := r.hostPlaybackLocked(session)
	scope := client.getScopeKey()
	partyKeys := copyStringSet(session.MemberKeys)
	partyCross := session.CrossUser
	name, username := r.memberLabelsLocked(scope, client.getDeviceID())
	r.mu.Unlock()
	r.broadcastDevicesTargets(scope, partyKeys, partyCross)
	r.broadcastSessionTargets(scope, partyKeys, partyCross, Event{
		Type: EventSessionUpdated,
		Payload: map[string]any{
			"sessionId": sessionID,
			"hostId":    hostID,
			"action":    "joined",
			"deviceId":  client.getDeviceID(),
			"name":      name,
			"username":  username,
			"state":     snap,
		},
	})
	return true
}

// InviteToSession adds a same-scope device to the caller's hosted listen session.
func (r *DeviceRegistry) InviteToSession(client *wsClient, targetID string) string {
	targetID = strings.TrimSpace(targetID)
	if targetID == "" {
		return "missing_target"
	}
	r.mu.Lock()
	scope := client.getScopeKey()
	host := r.devices[deviceKey(scope, client.getDeviceID())]
	if host == nil || host.client != client {
		r.mu.Unlock()
		return "not_registered"
	}
	if targetID == client.getDeviceID() {
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
			r.removeMemberLocked(old, deviceKey(scope, targetID), target.userID)
			prevLeftKeys = copyStringSet(old.MemberKeys)
			prevLeftCross = old.CrossUser
		}
		target.sessionID = ""
		target.isHost = false
		target.info.SessionID = ""
		target.info.IsHost = false
	}

	key := deviceKey(scope, targetID)
	r.addMemberLocked(session, key, target.userID)
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
		session.InviteToken = newPartyInviteToken()
	}
	return session.ID, session.InviteToken, true
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

func (r *DeviceRegistry) LeaveSession(client *wsClient) {
	r.mu.Lock()
	dev := r.devices[deviceKey(client.getScopeKey(), client.getDeviceID())]
	if dev == nil || dev.client != client || dev.sessionID == "" {
		r.mu.Unlock()
		return
	}
	sessionID := dev.sessionID
	wasHost := dev.isHost
	scope := client.getScopeKey()
	deviceID := client.getDeviceID()
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
		r.removeMemberLocked(session, key, client.userID)
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
			UserID:   dev.userID,
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

// PartyLeaveNotice describes persistent notifications for cross-user party leave or end.
type PartyLeaveNotice struct {
	Kind        string // "ended" or "left"
	Reason      string
	SessionID   string
	InviteToken string
	ActorName   string
	ActorUser   string
	NotifyIDs   []string
}

// PeekPartyLeaveNotice returns notification targets before LeaveSession or Unregister.
func (r *DeviceRegistry) PeekPartyLeaveNotice(scope, deviceID string) *PartyLeaveNotice {
	r.mu.Lock()
	defer r.mu.Unlock()
	dev := r.devices[deviceKey(scope, deviceID)]
	if dev == nil || dev.sessionID == "" {
		return nil
	}
	session := r.sessions[dev.sessionID]
	if session == nil || !session.CrossUser {
		return nil
	}
	name, username := r.memberLabelsLocked(scope, deviceID)
	if username == "" {
		username = name
	}
	if name == "" {
		name = "Someone"
	}
	if dev.isHost {
		members := make([]string, 0, len(session.MemberUserIDs))
		for uid := range session.MemberUserIDs {
			if uid != "" && uid != session.HostUserID {
				members = append(members, uid)
			}
		}
		if len(members) == 0 {
			return nil
		}
		return &PartyLeaveNotice{
			Kind:        "ended",
			Reason:      "host_left",
			SessionID:   session.ID,
			InviteToken: session.InviteToken,
			ActorName:   name,
			ActorUser:   username,
			NotifyIDs:   members,
		}
	}
	if session.HostUserID == "" || session.HostUserID == dev.userID {
		return nil
	}
	return &PartyLeaveNotice{
		Kind:        "left",
		Reason:      "left",
		SessionID:   session.ID,
		InviteToken: session.InviteToken,
		ActorName:   name,
		ActorUser:   username,
		NotifyIDs:   []string{session.HostUserID},
	}
}

// CrossUserLeaveNotify returns member user IDs to notify when the host ends a cross-user party.
func (r *DeviceRegistry) CrossUserLeaveNotify(scope, deviceID string) (sessionID, inviteToken, hostUserID string, memberUserIDs []string) {
	notice := r.PeekPartyLeaveNotice(scope, deviceID)
	if notice == nil || notice.Kind != "ended" {
		return "", "", "", nil
	}
	return notice.SessionID, notice.InviteToken, "", notice.NotifyIDs
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
		if dev := r.devices[memberKey]; dev != nil && dev.userID == userID {
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
