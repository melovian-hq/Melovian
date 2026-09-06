// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"maps"
	"slices"
	"time"
)

func (r *DeviceRegistry) Register(client *wsClient, name, userAgent string) DeviceInfo {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UnixMilli()
	key := deviceKey(client.getScopeKey(), client.getDeviceID())
	if existing, ok := r.devices[key]; ok && existing.client != client {
		existing.client.setDeviceID("")
	}

	displayName := name
	if displayName == "" {
		displayName = guessDeviceName(userAgent)
	}

	username := ""
	if r.lookupUsername != nil && client.userID != "" {
		username = r.lookupUsername(client.userID)
	}

	dev := &registeredDevice{
		client: client,
		info: DeviceInfo{
			DeviceID:    client.getDeviceID(),
			Name:        displayName,
			Username:    username,
			UserAgent:   userAgent,
			ConnectedAt: now,
			LastSeen:    now,
		},
		scopeKey: client.getScopeKey(),
		userID:   client.userID,
	}
	r.devices[key] = dev
	client.setRegistered(true)
	return r.publicInfo(dev)
}

func (r *DeviceRegistry) Unregister(client *wsClient) {
	if client == nil || client.getDeviceID() == "" {
		return
	}
	r.mu.Lock()
	key := deviceKey(client.getScopeKey(), client.getDeviceID())
	dev, ok := r.devices[key]
	if !ok || dev.client != client {
		r.mu.Unlock()
		return
	}
	sessionID := dev.sessionID
	wasHost := dev.isHost
	deviceID := client.getDeviceID()
	scope := client.getScopeKey()
	name, username := r.memberLabelsLocked(scope, deviceID)
	delete(r.devices, key)

	var endedKeys map[string]struct{}
	var endedCross bool
	var leftKeys map[string]struct{}
	var leftCross bool
	if sessionID != "" {
		session := r.sessions[sessionID]
		if wasHost {
			endedKeys, endedCross = r.endSessionLocked(sessionID)
		} else if session != nil {
			r.removeMemberLocked(session, key, client.userID)
			leftKeys = copyStringSet(session.MemberKeys)
			leftCross = session.CrossUser
		}
	}
	r.mu.Unlock()

	if endedKeys != nil {
		r.broadcastSessionTargets(scope, endedKeys, endedCross, Event{
			Type: EventSessionUpdated,
			Payload: map[string]any{
				"sessionId": sessionID,
				"action":    "ended",
				"reason":    "host_disconnected",
				"deviceId":  deviceID,
				"name":      name,
				"username":  username,
			},
		})
		r.broadcastDevicesTargets(scope, endedKeys, endedCross)
		return
	}
	if leftKeys != nil {
		r.broadcastSessionTargets(scope, leftKeys, leftCross, Event{
			Type: EventSessionUpdated,
			Payload: map[string]any{
				"sessionId": sessionID,
				"action":    "left",
				"reason":    "disconnected",
				"deviceId":  deviceID,
				"name":      name,
				"username":  username,
			},
		})
		r.broadcastDevicesTargets(scope, leftKeys, leftCross)
		return
	}
	r.broadcastDevices(scope)
}

func (r *DeviceRegistry) Rename(client *wsClient, name string) {
	if name == "" {
		return
	}
	r.mu.Lock()
	dev := r.devices[deviceKey(client.getScopeKey(), client.getDeviceID())]
	if dev == nil || dev.client != client {
		r.mu.Unlock()
		return
	}
	dev.info.Name = name
	scope := client.getScopeKey()
	r.mu.Unlock()
	r.broadcastDevices(scope)
}

func (r *DeviceRegistry) UpdatePlayback(client *wsClient, snap PlaybackSnapshot) {
	r.mu.Lock()
	dev := r.devices[deviceKey(client.getScopeKey(), client.getDeviceID())]
	if dev == nil || dev.client != client {
		r.mu.Unlock()
		return
	}
	snap.ServerTime = time.Now().UnixMilli()
	copySnap := snap
	prevTrack := ""
	if dev.info.Playback != nil {
		prevTrack = dev.info.Playback.TrackID
	}
	wasActive := dev.isActivePlayer
	dev.info.Playback = &copySnap
	dev.info.LastSeen = snap.ServerTime
	// Only claim active ownership from a playing report when nobody else
	// is active. Followers echoing host state must not steal the throne.
	if !dev.isActivePlayer && !snap.Paused && snap.TrackID != "" {
		hasActive := false
		for _, other := range r.devices {
			if other.scopeKey == client.getScopeKey() && other.isActivePlayer {
				hasActive = true
				break
			}
		}
		if !hasActive {
			r.setActiveLocked(client.getScopeKey(), client.getDeviceID())
		}
	}
	scope := client.getScopeKey()
	sessionID := dev.sessionID
	isHost := dev.isHost
	active := dev.isActivePlayer
	becameActive := !wasActive && active
	trackChanged := prevTrack != snap.TrackID

	var partyKeys map[string]struct{}
	partyCross := false
	if sessionID != "" {
		if session := r.sessions[sessionID]; session != nil {
			partyKeys = copyStringSet(session.MemberKeys)
			partyCross = session.CrossUser
		}
	}
	r.mu.Unlock()

	if becameActive || trackChanged {
		r.broadcastDevicesTargets(scope, partyKeys, partyCross && sessionID != "")
	}
	if active {
		r.broadcastToScope(scope, Event{
			Type:    EventPlaybackState,
			Payload: map[string]any{"deviceId": client.getDeviceID(), "state": snap},
		})
	}
	if sessionID != "" && isHost {
		r.broadcastSessionTargets(scope, partyKeys, partyCross, Event{
			Type: EventPlaybackState,
			Payload: map[string]any{
				"deviceId":  client.getDeviceID(),
				"sessionId": sessionID,
				"state":     snap,
				"together":  true,
			},
		})
	}
}

func (r *DeviceRegistry) sendToDevice(scope, deviceID string, event Event) {
	r.mu.Lock()
	dev := r.devices[deviceKey(scope, deviceID)]
	var client *wsClient
	if dev != nil {
		client = dev.client
	}
	r.mu.Unlock()
	if client == nil || r.hub == nil {
		return
	}
	r.hub.SendToClient(client, event)
}

func (r *DeviceRegistry) TakeOver(client *wsClient) string {
	return r.grantPlayback(client, client.getDeviceID())
}

func (r *DeviceRegistry) HandoffTo(client *wsClient, targetID string) string {
	if targetID == "" {
		return "missing_target"
	}
	return r.grantPlayback(client, targetID)
}

func (r *DeviceRegistry) grantPlayback(client *wsClient, targetID string) string {
	r.mu.Lock()
	scope := client.getScopeKey()
	source := r.devices[deviceKey(scope, client.getDeviceID())]
	if source == nil || source.client != client {
		r.mu.Unlock()
		return "not_registered"
	}
	target := r.devices[deviceKey(scope, targetID)]
	if target == nil {
		r.mu.Unlock()
		return "target_offline"
	}

	var active *registeredDevice
	for _, other := range r.devices {
		if other.scopeKey == scope && other.isActivePlayer {
			active = other
			break
		}
	}

	snapshot := copyPlayback(source.info.Playback)
	if snapshot == nil || snapshot.TrackID == "" {
		if active != nil {
			snapshot = copyPlayback(active.info.Playback)
		}
	}
	if snapshot == nil || snapshot.TrackID == "" {
		for _, other := range r.devices {
			if other.scopeKey != scope {
				continue
			}
			if other.info.Playback != nil && other.info.Playback.TrackID != "" {
				snapshot = copyPlayback(other.info.Playback)
				break
			}
		}
	}
	if snapshot == nil || snapshot.TrackID == "" {
		r.mu.Unlock()
		return "nothing_playing"
	}

	releaseIDs := make([]string, 0, 2)
	if active != nil && active.info.DeviceID != targetID {
		active.isActivePlayer = false
		active.info.IsActivePlayer = false
		releaseIDs = append(releaseIDs, active.info.DeviceID)
	}
	if source.info.DeviceID != targetID && source.info.DeviceID != "" {
		already := slices.Contains(releaseIDs, source.info.DeviceID)
		if !already {
			releaseIDs = append(releaseIDs, source.info.DeviceID)
		}
	}

	r.setActiveLocked(scope, targetID)
	fromID := client.getDeviceID()
	r.mu.Unlock()

	for _, id := range releaseIDs {
		r.sendToDevice(scope, id, Event{
			Type: EventPlaybackCommand,
			Payload: map[string]any{
				"action":   "pause_release",
				"targetId": id,
				"fromId":   fromID,
				"reason":   "handoff",
			},
		})
	}
	r.sendToDevice(scope, targetID, Event{
		Type: EventPlaybackCommand,
		Payload: map[string]any{
			"action":   "take_over",
			"targetId": targetID,
			"fromId":   fromID,
			"state":    snapshot,
		},
	})
	r.broadcastDevices(scope)
	return ""
}
func (r *DeviceRegistry) Command(client *wsClient, action, targetID string, extra map[string]any) {
	r.mu.Lock()
	scope := client.getScopeKey()
	if targetID == "" {
		for _, other := range r.devices {
			if other.scopeKey == scope && other.isActivePlayer {
				targetID = other.info.DeviceID
				break
			}
		}
	}
	r.mu.Unlock()
	if targetID == "" {
		return
	}
	payload := map[string]any{
		"action":   action,
		"targetId": targetID,
		"fromId":   client.getDeviceID(),
	}
	maps.Copy(payload, extra)
	r.sendToDevice(scope, targetID, Event{
		Type:    EventPlaybackCommand,
		Payload: payload,
	})
}
func (r *DeviceRegistry) List(scope string) []DeviceInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]DeviceInfo, 0)
	for _, dev := range r.devices {
		if dev.scopeKey != scope {
			continue
		}
		out = append(out, r.publicInfo(dev))
	}
	return out
}

func (r *DeviceRegistry) GetSession(sessionID string) (ListenSession, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	session := r.sessions[sessionID]
	if session == nil {
		return ListenSession{}, false
	}
	return cloneSession(*session), true
}
func (r *DeviceRegistry) hostPlaybackLocked(session *ListenSession) *PlaybackSnapshot {
	if session == nil {
		return nil
	}
	host := r.devices[deviceKey(session.Scope, session.HostID)]
	if host == nil || host.info.Playback == nil {
		return nil
	}
	copied := copyPlayback(host.info.Playback)
	if copied == nil {
		return nil
	}
	advanced := advancePlaybackSnapshot(*copied, time.Now().UnixMilli())
	return &advanced
}
func (r *DeviceRegistry) setActiveLocked(scope, deviceID string) {
	for _, dev := range r.devices {
		if dev.scopeKey != scope {
			continue
		}
		active := dev.info.DeviceID == deviceID
		dev.isActivePlayer = active
		dev.info.IsActivePlayer = active
	}
}

func (r *DeviceRegistry) publicInfo(dev *registeredDevice) DeviceInfo {
	info := dev.info
	info.IsActivePlayer = dev.isActivePlayer
	info.SessionID = dev.sessionID
	info.IsHost = dev.isHost
	if dev.info.Playback != nil {
		copySnap := *dev.info.Playback
		info.Playback = &copySnap
	}
	return info
}

func (r *DeviceRegistry) listByKeysLocked(keys map[string]struct{}) []DeviceInfo {
	out := make([]DeviceInfo, 0, len(keys))
	for key := range keys {
		if dev := r.devices[key]; dev != nil {
			out = append(out, r.publicInfo(dev))
		}
	}
	return out
}

func (r *DeviceRegistry) broadcastDevices(scope string) {
	devices := r.List(scope)
	r.broadcastToScope(scope, Event{
		Type:    EventDevicesUpdated,
		Payload: map[string]any{"devices": devices},
	})
}

func (r *DeviceRegistry) broadcastDevicesTargets(scope string, memberKeys map[string]struct{}, crossUser bool) {
	if crossUser {
		r.mu.Lock()
		devices := r.listByKeysLocked(memberKeys)
		r.mu.Unlock()
		r.broadcastToMemberKeys(memberKeys, Event{
			Type:    EventDevicesUpdated,
			Payload: map[string]any{"devices": devices, "party": true},
		})
		return
	}
	r.broadcastDevices(scope)
}

func (r *DeviceRegistry) broadcastSessionTargets(scope string, memberKeys map[string]struct{}, crossUser bool, event Event) {
	if crossUser {
		r.broadcastToMemberKeys(memberKeys, event)
		return
	}
	r.broadcastToScope(scope, event)
}

func (r *DeviceRegistry) broadcastToMemberKeys(keys map[string]struct{}, event Event) {
	if r.hub == nil || len(keys) == 0 {
		return
	}
	r.mu.Lock()
	clients := make([]*wsClient, 0, len(keys))
	for key := range keys {
		if dev := r.devices[key]; dev != nil && dev.client != nil {
			clients = append(clients, dev.client)
		}
	}
	r.mu.Unlock()
	for _, c := range clients {
		r.hub.SendToClient(c, event)
	}
}

func (r *DeviceRegistry) broadcastToScope(scope string, event Event) {
	if r.hub == nil {
		return
	}
	r.hub.BroadcastToScope(scope, event)
}
