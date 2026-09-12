// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"melovian/internal/httputil"
)

func (s *Server) registerWSRoutes() {
	s.mux.HandleFunc("GET /api/ws", s.handleWebSocket)
	s.mux.HandleFunc("GET /api/devices", s.handleListDevices)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if s.auth != nil && s.auth.Enabled() {
		token := sessionTokenFromRequest(r)
		resolved, err := s.auth.UserIDFromToken(token)
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
		OriginPatterns: wsOriginPatterns(),
	})
	if err != nil {
		return
	}

	authEnabled := s.auth != nil && s.auth.Enabled()
	client := &wsClient{
		userID: userID,
		send:   make(chan []byte, 64),
	}
	client.setDeviceID(deviceID)
	client.setScopeKey(ResolveDeviceScope(userID, deviceID, authEnabled))
	s.events.register(client)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer func() {
		notice := s.devices.PeekPartyLeaveNotice(client.getScopeKey(), client.getDeviceID())
		if notice != nil && notice.Kind == "ended" {
			notice.Reason = "host_disconnected"
		}
		if notice != nil && notice.Kind == "left" {
			notice.Reason = "disconnected"
		}
		s.devices.Unregister(client)
		s.dispatchPartyLeaveNotice(notice)
		s.events.unregister(client)
		_ = conn.Close(websocket.StatusNormalClosure, "closed")
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-client.send:
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
			s.handleDeviceClientMessage(client, data)
		}
	}
}
