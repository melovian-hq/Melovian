// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package realtime

import (
	"context"
	"net/http"
	"time"

	"melovian/internal/api/apishared"
	"melovian/internal/httputil"
	"melovian/internal/store"

	"github.com/coder/websocket"
)

// Handler serves the websocket endpoint and device-sync HTTP routes.
type Handler struct {
	auth    *store.AuthStore
	events  *EventHub
	devices *DeviceRegistry
	notify  func(store.CreateNotificationInput) (store.Notification, error)
}

func New(auth *store.AuthStore, events *EventHub, devices *DeviceRegistry, notify func(store.CreateNotificationInput) (store.Notification, error)) *Handler {
	return &Handler{auth: auth, events: events, devices: devices, notify: notify}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/ws", h.handleWebSocket)
	mux.HandleFunc("GET /api/devices", h.handleListDevices)
}

func (h *Handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := apishared.UserIDFromContext(r.Context())
	if h.auth != nil && h.auth.Enabled() {
		token := apishared.SessionTokenFromRequest(r)
		resolved, err := h.auth.UserIDFromToken(token)
		if err != nil {
			httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
			return
		}
		userID = resolved
	}

	deviceID := r.Header.Get("X-Device-Id")
	if deviceID == "" {
		deviceID = r.URL.Query().Get("deviceId")
	}

	// Same-origin upgrades are always allowed by Accept. Cross-origin upgrades
	// are limited to the configured CORS allowlist so a random web page cannot
	// open a control socket to a desktop or LAN instance.
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: apishared.WSOriginPatterns(),
	})
	if err != nil {
		return
	}

	authEnabled := h.auth != nil && h.auth.Enabled()
	client := NewWSClient(userID, 64)
	client.SetDeviceID(deviceID)
	client.SetScopeKey(ResolveDeviceScope(userID, deviceID, authEnabled))
	h.events.Register(client)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer func() {
		notice := h.devices.PeekPartyLeaveNotice(client.ScopeKey(), client.DeviceID())
		if notice != nil && notice.Kind == "ended" {
			notice.Reason = "host_disconnected"
		}
		if notice != nil && notice.Kind == "left" {
			notice.Reason = "disconnected"
		}
		h.devices.Unregister(client)
		h.dispatchPartyLeaveNotice(notice)
		h.events.Unregister(client)
		_ = conn.Close(websocket.StatusNormalClosure, "closed")
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-client.Send:
				if !ok {
					return
				}
				writeCtx, writeCancel := context.WithTimeout(ctx, 5*time.Second)
				err := conn.Write(writeCtx, websocket.MessageText, msg)
				writeCancel()
				if err != nil {
					cancel()
					return
				}
			}
		}
	}()

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if len(data) > 0 {
			h.handleDeviceClientMessage(client, data)
		}
	}
}
