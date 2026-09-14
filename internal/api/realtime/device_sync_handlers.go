// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package realtime

import (
	"encoding/json"
	"net/http"

	"melovian/internal/api/apishared"
	"melovian/internal/httputil"
)

func (h *Handler) handleDeviceClientMessage(client *WSClient, raw []byte) {
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
			client.SetDeviceID(p.DeviceID)
		}
		if client.DeviceID() == "" {
			return
		}
		authEnabled := h.auth != nil && h.auth.Enabled()
		client.SetScopeKey(ResolveDeviceScope(client.UserID, client.DeviceID(), authEnabled))
		info := h.devices.Register(client, p.Name, p.UserAgent)
		h.events.SendToClient(client, Event{
			Type:    EventDevicesUpdated,
			Payload: map[string]any{"devices": h.devices.List(client.ScopeKey()), "self": info},
		})
		h.devices.broadcastDevices(client.ScopeKey())

	case CmdDeviceRename:
		var p struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(msg.Payload, &p)
		h.devices.Rename(client, p.Name)

	case CmdPlaybackState:
		var snap PlaybackSnapshot
		_ = json.Unmarshal(msg.Payload, &snap)
		h.devices.UpdatePlayback(client, snap)

	case CmdPlaybackTakeOver:
		if code := h.devices.TakeOver(client); code != "" {
			h.replyDeviceError(client, code, deviceErrorMessage(code))
		}

	case CmdPlaybackHandoffTo:
		var p struct {
			TargetID string `json:"targetId"`
		}
		_ = json.Unmarshal(msg.Payload, &p)
		if code := h.devices.HandoffTo(client, p.TargetID); code != "" {
			h.replyDeviceError(client, code, deviceErrorMessage(code))
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
		h.devices.Command(client, p.Action, p.TargetID, extra)

	case CmdSessionCreate:
		if id := h.devices.CreateSession(client); id == "" {
			h.replyDeviceError(client, "not_registered", deviceErrorMessage("not_registered"))
		}

	case CmdSessionJoin:
		var p struct {
			SessionID string `json:"sessionId"`
		}
		_ = json.Unmarshal(msg.Payload, &p)
		if p.SessionID == "" {
			h.replyDeviceError(client, "session_missing", deviceErrorMessage("session_missing"))
			return
		}
		if !h.devices.JoinSession(client, p.SessionID) {
			h.replyDeviceError(client, "session_unavailable", deviceErrorMessage("session_unavailable"))
		}

	case CmdSessionLeave:
		h.devices.LeaveSession(client)

	case CmdSessionInvite:
		var p struct {
			TargetID string `json:"targetId"`
		}
		_ = json.Unmarshal(msg.Payload, &p)
		if code := h.devices.InviteToSession(client, p.TargetID); code != "" {
			h.replyDeviceError(client, code, deviceErrorMessage(code))
		}
	}
}

func (h *Handler) replyDeviceError(client *WSClient, code, message string) {
	if h.events == nil || client == nil {
		return
	}
	h.events.SendToClient(client, Event{
		Type: EventDeviceError,
		Payload: map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
func (h *Handler) handleListDevices(w http.ResponseWriter, r *http.Request) {
	userID := apishared.UserIDFromContext(r.Context())
	deviceID := apishared.DeviceIDFromContext(r.Context())
	authEnabled := h.auth != nil && h.auth.Enabled()
	if authEnabled && userID == "" {
		token := apishared.SessionTokenFromRequest(r)
		resolved, err := h.auth.UserIDFromToken(token)
		if err == nil {
			userID = resolved
		}
	}
	scope := ResolveDeviceScope(userID, deviceID, authEnabled)
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"devices": h.devices.List(scope),
	})
}
