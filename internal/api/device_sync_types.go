// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"sync"
	"time"
)

const (
	EventDevicesUpdated  = "devices.updated"
	EventPlaybackState   = "playback.state"
	EventPlaybackCommand = "playback.command"
	EventSessionUpdated  = "session.updated"
	EventDeviceError     = "device.error"

	CmdDeviceRegister    = "device.register"
	CmdDeviceRename      = "device.rename"
	CmdPlaybackState     = "playback.state"
	CmdPlaybackCommand   = "playback.command"
	CmdPlaybackTakeOver  = "playback.takeOver"
	CmdPlaybackHandoffTo = "playback.handoffTo"
	CmdSessionCreate     = "session.create"
	CmdSessionJoin       = "session.join"
	CmdSessionLeave      = "session.leave"
	CmdSessionInvite     = "session.invite"
)

type PlaybackSnapshot struct {
	TrackID    string   `json:"trackId,omitempty"`
	TrackTitle string   `json:"trackTitle,omitempty"`
	ArtistName string   `json:"artistName,omitempty"`
	CoverArt   string   `json:"coverArt,omitempty"`
	PositionMs int64    `json:"positionMs"`
	DurationMs int64    `json:"durationMs,omitempty"`
	Paused     bool     `json:"paused"`
	QueueIDs   []string `json:"queueIds,omitempty"`
	QueueIndex int      `json:"queueIndex,omitempty"`
	ServerTime int64    `json:"serverTime"`
}

type DeviceInfo struct {
	DeviceID       string            `json:"deviceId"`
	Name           string            `json:"name"`
	Username       string            `json:"username,omitempty"`
	UserAgent      string            `json:"userAgent,omitempty"`
	ConnectedAt    int64             `json:"connectedAt"`
	LastSeen       int64             `json:"lastSeen"`
	IsActivePlayer bool              `json:"isActivePlayer"`
	Playback       *PlaybackSnapshot `json:"playback,omitempty"`
	SessionID      string            `json:"sessionId,omitempty"`
	IsHost         bool              `json:"isHost,omitempty"`
}

type ListenSession struct {
	ID            string
	HostID        string
	Scope         string
	HostUserID    string
	InviteToken   string
	CrossUser     bool
	MemberUserIDs map[string]struct{}
	MemberKeys    map[string]struct{}
}

type PartyMemberInfo struct {
	DeviceID string `json:"deviceId"`
	UserID   string `json:"userId,omitempty"`
	Username string `json:"username,omitempty"`
	Name     string `json:"name"`
	IsHost   bool   `json:"isHost"`
}

type PartyStatus struct {
	SessionID  string            `json:"sessionId"`
	HostID     string            `json:"hostId"`
	HostUserID string            `json:"hostUserId,omitempty"`
	CrossUser  bool              `json:"crossUser"`
	Members    []PartyMemberInfo `json:"members"`
}

type registeredDevice struct {
	client         *wsClient
	info           DeviceInfo
	scopeKey       string
	userID         string
	isActivePlayer bool
	sessionID      string
	isHost         bool
}

type DeviceRegistry struct {
	mu             sync.Mutex
	hub            *EventHub
	devices        map[string]*registeredDevice
	sessions       map[string]*ListenSession
	lookupUsername func(userID string) string
}

func NewDeviceRegistry(hub *EventHub) *DeviceRegistry {
	return &DeviceRegistry{
		hub:      hub,
		devices:  make(map[string]*registeredDevice),
		sessions: make(map[string]*ListenSession),
	}
}

func newPartyInviteToken() string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("pty-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func newMemberMaps() (map[string]struct{}, map[string]struct{}) {
	return make(map[string]struct{}), make(map[string]struct{})
}

func copyStringSet(src map[string]struct{}) map[string]struct{} {
	if len(src) == 0 {
		return map[string]struct{}{}
	}
	out := make(map[string]struct{}, len(src))
	maps.Copy(out, src)
	return out
}

func deviceKey(scope, deviceID string) string {
	return scope + "\x00" + deviceID
}

func ResolveDeviceScope(userID, deviceID string, authEnabled bool) string {
	if authEnabled && userID != "" {
		return "user:" + userID
	}
	return "server:local"
}

func guessDeviceName(ua string) string {
	switch {
	case strings.Contains(ua, "iPhone"):
		return "iPhone"
	case strings.Contains(ua, "iPad"):
		return "iPad"
	case strings.Contains(ua, "Android"):
		return "Android"
	case strings.Contains(ua, "Macintosh"), strings.Contains(ua, "Mac OS"):
		return "Mac"
	case strings.Contains(ua, "Windows"):
		return "Windows"
	case strings.Contains(ua, "Linux"):
		return "Linux"
	default:
		return "Browser"
	}
}
func copyPlayback(src *PlaybackSnapshot) *PlaybackSnapshot {
	if src == nil {
		return nil
	}
	copySnap := *src
	if src.QueueIDs != nil {
		copySnap.QueueIDs = append([]string(nil), src.QueueIDs...)
	}
	return &copySnap
}

func advancePlaybackSnapshot(snap PlaybackSnapshot, now int64) PlaybackSnapshot {
	if snap.Paused || snap.ServerTime <= 0 {
		snap.ServerTime = now
		return snap
	}
	elapsed := now - snap.ServerTime
	if elapsed > 0 {
		snap.PositionMs += elapsed
		if snap.DurationMs > 0 && snap.PositionMs > snap.DurationMs {
			snap.PositionMs = snap.DurationMs
		}
	}
	snap.ServerTime = now
	return snap
}
func cloneSession(s ListenSession) ListenSession {
	out := s
	out.MemberUserIDs = copyStringSet(s.MemberUserIDs)
	out.MemberKeys = copyStringSet(s.MemberKeys)
	return out
}

type clientMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func deviceErrorMessage(code string) string {
	switch code {
	case "not_registered":
		return "This device is not connected yet. Wait a moment and try again."
	case "target_offline":
		return "That device is not connected."
	case "nothing_playing":
		return "Nothing is playing to transfer."
	case "missing_target":
		return "Pick a device."
	case "session_missing":
		return "That listen together session is missing."
	case "session_unavailable":
		return "Could not join. The session ended or is on another account."
	case "not_hosting":
		return "Start listen together before inviting a device."
	case "already_member":
		return "That device is already listening together."
	case "target_hosting":
		return "That device is hosting its own session. Ask them to leave first."
	default:
		return "The device action failed."
	}
}
