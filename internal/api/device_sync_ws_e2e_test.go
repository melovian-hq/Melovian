// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"melovian/internal/api/realtime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestDevicesRESTSmoke(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/devices", nil)
	req.Header.Set("X-Device-Id", "rest-device-1")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Devices []realtime.DeviceInfo `json:"devices"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Devices == nil {
		t.Fatal("devices array should be present")
	}
}

func TestDeviceSyncWebSocketE2E(t *testing.T) {
	srv, _ := newTestServer(t)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	dial := func(deviceID string) *websocket.Conn {
		t.Helper()
		wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/ws?deviceId=" + deviceID
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)
		conn, _, err := websocket.Dial(ctx, wsURL, nil)
		if err != nil {
			t.Fatalf("dial %s: %v", deviceID, err)
		}
		t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "done") })
		return conn
	}

	send := func(conn *websocket.Conn, typ string, payload any) {
		t.Helper()
		msg, _ := json.Marshal(map[string]any{"type": typ, "payload": payload})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := conn.Write(ctx, websocket.MessageText, msg); err != nil {
			t.Fatalf("write %s: %v", typ, err)
		}
	}

	readEvent := func(conn *websocket.Conn, wantType string, timeout time.Duration) realtime.Event {
		t.Helper()
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Until(deadline))
			_, data, err := conn.Read(ctx)
			cancel()
			if err != nil {
				continue
			}
			var ev realtime.Event
			if err := json.Unmarshal(data, &ev); err != nil {
				continue
			}
			if ev.Type == wantType {
				return ev
			}
		}
		t.Fatalf("timed out waiting for %s", wantType)
		return realtime.Event{}
	}

	a := dial("e2e-a")
	b := dial("e2e-b")

	send(a, realtime.CmdDeviceRegister, map[string]any{
		"deviceId":  "e2e-a",
		"name":      "Device A",
		"userAgent": "Linux",
	})
	_ = readEvent(a, realtime.EventDevicesUpdated, 3*time.Second)

	send(b, realtime.CmdDeviceRegister, map[string]any{
		"deviceId":  "e2e-b",
		"name":      "Device B",
		"userAgent": "Windows",
	})
	_ = readEvent(b, realtime.EventDevicesUpdated, 3*time.Second)

	send(a, realtime.CmdPlaybackState, realtime.PlaybackSnapshot{
		TrackID:    "song-1",
		PositionMs: 15000,
		Paused:     false,
		QueueIDs:   []string{"song-1"},
	})

	deadline := time.Now().Add(3 * time.Second)
	for {
		devices := srv.devices.List("server:local")
		ready := false
		for _, d := range devices {
			if d.DeviceID == "e2e-a" && d.IsActivePlayer && d.Playback != nil && d.Playback.TrackID == "song-1" {
				ready = true
				break
			}
		}
		if ready {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for device A playback ownership, devices=%d", len(devices))
		}
		time.Sleep(5 * time.Millisecond)
	}

	send(b, realtime.CmdPlaybackTakeOver, map[string]any{})
	deadline = time.Now().Add(3 * time.Second)
	gotTakeOver := false
	for time.Now().Before(deadline) {
		ev := readEvent(b, realtime.EventPlaybackCommand, time.Until(deadline))
		payload, _ := json.Marshal(ev.Payload)
		var cmd map[string]any
		_ = json.Unmarshal(payload, &cmd)
		if cmd["action"] == "take_over" {
			gotTakeOver = true
			break
		}
	}
	if !gotTakeOver {
		t.Fatal("expected take_over for B")
	}

	send(a, realtime.CmdSessionCreate, map[string]any{})
	sess := readEvent(a, realtime.EventSessionUpdated, 3*time.Second)
	sessPayload, _ := json.Marshal(sess.Payload)
	var session map[string]any
	_ = json.Unmarshal(sessPayload, &session)
	sessionID, _ := session["sessionId"].(string)
	if sessionID == "" {
		t.Fatal("expected sessionId from create")
	}

	send(b, realtime.CmdSessionJoin, map[string]any{"sessionId": sessionID})
	join := readEvent(b, realtime.EventSessionUpdated, 3*time.Second)
	joinPayload, _ := json.Marshal(join.Payload)
	var joined map[string]any
	_ = json.Unmarshal(joinPayload, &joined)
	if joined["action"] != "joined" && joined["sessionId"] != sessionID {
		if joined["sessionId"] != sessionID {
			t.Fatalf("unexpected join payload: %v", joined)
		}
	}
}
