// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"melovian/internal/httputil"
	"melovian/internal/store"
)

func (s *Server) handleDeviceClientMessage(client *wsClient, raw []byte) {
	var msg clientMessage
	if err := json.Unmarshal(raw, &msg); err != nil || msg.Type == "" {
		return
	}
	switch msg.Type {
	case CmdDeviceRegister:
		var p struct {
			DeviceID  string `json:"deviceId"`
			Name      string `json:"name"`
			UserAgent string `json:"userAgent"`
		}
		_ = json.Unmarshal(msg.Payload, &p)
		if p.DeviceID != "" {
			client.setDeviceID(p.DeviceID)
		}
		if client.getDeviceID() == "" {
			return
		}
		authEnabled := s.auth != nil && s.auth.Enabled()
		client.setScopeKey(ResolveDeviceScope(client.userID, client.getDeviceID(), authEnabled))
		info := s.devices.Register(client, p.Name, p.UserAgent)
		s.events.SendToClient(client, Event{
			Type:    EventDevicesUpdated,
			Payload: map[string]any{"devices": s.devices.List(client.getScopeKey()), "self": info},
		})
		s.devices.broadcastDevices(client.getScopeKey())

	case CmdDeviceRename:
		var p struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(msg.Payload, &p)
		s.devices.Rename(client, p.Name)

	case CmdPlaybackState:
		var snap PlaybackSnapshot
		_ = json.Unmarshal(msg.Payload, &snap)
		s.devices.UpdatePlayback(client, snap)

	case CmdPlaybackTakeOver:
		if code := s.devices.TakeOver(client); code != "" {
			s.replyDeviceError(client, code, deviceErrorMessage(code))
		}

	case CmdPlaybackHandoffTo:
		var p struct {
			TargetID string `json:"targetId"`
		}
		_ = json.Unmarshal(msg.Payload, &p)
		if code := s.devices.HandoffTo(client, p.TargetID); code != "" {
			s.replyDeviceError(client, code, deviceErrorMessage(code))
		}

	case CmdPlaybackCommand:
		var p struct {
			Action   string         `json:"action"`
			TargetID string         `json:"targetId"`
			Extra    map[string]any `json:"extra"`
		}
		_ = json.Unmarshal(msg.Payload, &p)
		extra := p.Extra
		if extra == nil {
			var all map[string]any
			_ = json.Unmarshal(msg.Payload, &all)
			extra = map[string]any{}
			for k, v := range all {
				if k == "action" || k == "targetId" {
					continue
				}
				extra[k] = v
			}
		}
		s.devices.Command(client, p.Action, p.TargetID, extra)

	case CmdSessionCreate:
		if id := s.devices.CreateSession(client); id == "" {
			s.replyDeviceError(client, "not_registered", deviceErrorMessage("not_registered"))
		}

	case CmdSessionJoin:
		var p struct {
			SessionID string `json:"sessionId"`
		}
		_ = json.Unmarshal(msg.Payload, &p)
		if p.SessionID == "" {
			s.replyDeviceError(client, "session_missing", deviceErrorMessage("session_missing"))
			return
		}
		if !s.devices.JoinSession(client, p.SessionID) {
			s.replyDeviceError(client, "session_unavailable", deviceErrorMessage("session_unavailable"))
		}

	case CmdSessionLeave:
		s.leaveListenSession(client)

	case CmdSessionInvite:
		var p struct {
			TargetID string `json:"targetId"`
		}
		_ = json.Unmarshal(msg.Payload, &p)
		if code := s.devices.InviteToSession(client, p.TargetID); code != "" {
			s.replyDeviceError(client, code, deviceErrorMessage(code))
		}
	}
}

func (s *Server) leaveListenSession(client *wsClient) {
	if client == nil || s.devices == nil {
		return
	}
	notice := s.devices.PeekPartyLeaveNotice(client.getScopeKey(), client.getDeviceID())
	if notice != nil && notice.Kind == "ended" {
		notice.Reason = "host_left"
	}
	if notice != nil && notice.Kind == "left" {
		notice.Reason = "left"
	}
	s.devices.LeaveSession(client)
	s.dispatchPartyLeaveNotice(notice)
}

func (s *Server) dispatchPartyLeaveNotice(notice *PartyLeaveNotice) {
	if notice == nil || len(notice.NotifyIDs) == 0 {
		return
	}
	href := ""
	if notice.InviteToken != "" {
		href = "/listen/" + notice.InviteToken
	}
	actor := strings.TrimSpace(notice.ActorUser)
	if actor == "" {
		actor = strings.TrimSpace(notice.ActorName)
	}
	if actor == "" {
		actor = "Someone"
	}
	payload, _ := json.Marshal(map[string]any{
		"sessionId": notice.SessionID,
		"reason":    notice.Reason,
		"name":      notice.ActorName,
		"username":  notice.ActorUser,
	})
	switch notice.Kind {
	case "ended":
		body := "The host left the session."
		if notice.Reason == "host_disconnected" {
			body = "The host disconnected."
		}
		for _, uid := range notice.NotifyIDs {
			_, _ = s.notifyUser(store.CreateNotificationInput{
				UserID:  uid,
				Kind:    store.NotificationPartyEnded,
				Title:   "Listen together ended",
				Body:    body,
				Href:    href,
				Payload: string(payload),
			})
		}
	case "left":
		title := actor + " left the session"
		body := ""
		if notice.Reason == "disconnected" {
			title = actor + " disconnected"
			body = "They dropped out of listen together."
		}
		for _, uid := range notice.NotifyIDs {
			_, _ = s.notifyUser(store.CreateNotificationInput{
				UserID:  uid,
				Kind:    store.NotificationPartyLeft,
				Title:   title,
				Body:    body,
				Href:    href,
				Payload: string(payload),
			})
		}
	}
}

func (s *Server) replyDeviceError(client *wsClient, code, message string) {
	if s.events == nil || client == nil {
		return
	}
	s.events.SendToClient(client, Event{
		Type: EventDeviceError,
		Payload: map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	deviceID := DeviceIDFromContext(r.Context())
	authEnabled := s.auth != nil && s.auth.Enabled()
	if authEnabled && userID == "" {
		token := sessionTokenFromRequest(r)
		resolved, err := s.auth.UserIDFromToken(token)
		if err == nil {
			userID = resolved
		}
	}
	scope := ResolveDeviceScope(userID, deviceID, authEnabled)
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"devices": s.devices.List(scope),
	})
}
