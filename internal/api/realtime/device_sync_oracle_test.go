// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package realtime

import (
	"encoding/json"
	"testing"
)

func TestTakeOverTransfersExactSnapshotOracle(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	scope := "server:local"

	active := mockWSClient("", "active", scope)
	idle := mockWSClient("", "idle", scope)
	hub.Register(active)
	hub.Register(idle)
	reg.Register(active, "Active", "Linux")
	reg.Register(idle, "Idle", "Windows")

	reg.UpdatePlayback(active, PlaybackSnapshot{
		TrackID:    "song-9",
		PositionMs: 77_777,
		Paused:     false,
		QueueIDs:   []string{"song-9", "song-10"},
		QueueIndex: 0,
	})

	// Drain noise so we can assert the handoff payload.
	drain := func(c *WSClient) {
		for {
			select {
			case <-c.Send:
			default:
				return
			}
		}
	}
	drain(active)
	drain(idle)

	reg.TakeOver(idle)

	found := false
	for range 8 {
		select {
		case msg := <-idle.Send:
			var ev Event
			if err := jsonUnmarshal(msg, &ev); err != nil {
				continue
			}
			if ev.Type != EventPlaybackCommand {
				continue
			}
			payload := asMap(ev.Payload)
			if payload["action"] != "take_over" {
				continue
			}
			state := asMap(payload["state"])
			if state["trackId"] != "song-9" {
				t.Fatalf("handoff trackId=%v want song-9", state["trackId"])
			}
			pos, ok := asInt64(state["positionMs"])
			if !ok || pos != 77_777 {
				t.Fatalf("handoff positionMs=%v want 77777", state["positionMs"])
			}
			found = true
		default:
		}
	}
	if !found {
		t.Fatal("idle device never received take_over with playback snapshot")
	}
}

func TestUpdatePlaybackDoesNotBroadcastDevicesOnPositionTicksOracle(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	scope := "server:local"

	player := mockWSClient("", "player", scope)
	peer := mockWSClient("", "peer", scope)
	hub.Register(player)
	hub.Register(peer)
	reg.Register(player, "Player", "Linux")
	reg.Register(peer, "Peer", "Windows")

	drain := func(c *WSClient) {
		for {
			select {
			case <-c.Send:
			default:
				return
			}
		}
	}
	drain(player)
	drain(peer)

	reg.UpdatePlayback(player, PlaybackSnapshot{
		TrackID:    "t1",
		PositionMs: 1000,
		Paused:     false,
	})
	// First playing claim may broadcast devices (became active / track set).
	drain(peer)

	reg.UpdatePlayback(player, PlaybackSnapshot{
		TrackID:    "t1",
		PositionMs: 5000,
		Paused:     false,
	})
	reg.UpdatePlayback(player, PlaybackSnapshot{
		TrackID:    "t1",
		PositionMs: 9000,
		Paused:     false,
	})

	for range 8 {
		select {
		case msg := <-peer.Send:
			var ev Event
			_ = json.Unmarshal(msg, &ev)
			if ev.Type == EventDevicesUpdated {
				t.Fatal("position-only playback updates must not spam devices.updated")
			}
		default:
			return
		}
	}
}

func TestFollowerEchoCannotStealActivePlayerOracle(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	scope := "server:local"

	host := mockWSClient("", "host", scope)
	follower := mockWSClient("", "follower", scope)
	hub.Register(host)
	hub.Register(follower)
	reg.Register(host, "Host", "Mac")
	reg.Register(follower, "Follower", "Linux")

	reg.UpdatePlayback(host, PlaybackSnapshot{
		TrackID:    "t1",
		PositionMs: 1000,
		Paused:     false,
	})
	reg.UpdatePlayback(follower, PlaybackSnapshot{
		TrackID:    "t1",
		PositionMs: 1000,
		Paused:     false,
	})

	list := reg.List(scope)
	var hostInfo, followerInfo DeviceInfo
	for _, d := range list {
		if d.DeviceID == "host" {
			hostInfo = d
		}
		if d.DeviceID == "follower" {
			followerInfo = d
		}
	}
	if !hostInfo.IsActivePlayer {
		t.Fatal("host should remain active")
	}
	if followerInfo.IsActivePlayer {
		t.Fatal("follower echo must not steal active player")
	}
}

func TestJoinSessionRejectsCrossScopeAdversarial(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)

	host := mockWSClient("alice", "host", "user:alice")
	intruder := mockWSClient("bob", "bob", "user:bob")
	reg.Register(host, "Host", "Mac")
	reg.Register(intruder, "Intruder", "Linux")

	sessionID := reg.CreateSession(host)
	if reg.JoinSession(intruder, sessionID) {
		t.Fatal("cross-user join must be rejected")
	}
}

func TestPlaybackCommandsAreTargetedNotBroadcastOracle(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	scope := "server:local"

	active := mockWSClient("", "active", scope)
	idle := mockWSClient("", "idle", scope)
	bystander := mockWSClient("", "bystander", scope)
	hub.Register(active)
	hub.Register(idle)
	hub.Register(bystander)
	reg.Register(active, "Active", "Linux")
	reg.Register(idle, "Idle", "Windows")
	reg.Register(bystander, "Bystander", "Mac")
	reg.UpdatePlayback(active, PlaybackSnapshot{TrackID: "t1", Paused: false})

	drain := func(c *WSClient) {
		for {
			select {
			case <-c.Send:
			default:
				return
			}
		}
	}
	drain(active)
	drain(idle)
	drain(bystander)

	reg.TakeOver(idle)

	assertNoCommand := func(name string, c *WSClient) {
		t.Helper()
		for range 8 {
			select {
			case msg := <-c.Send:
				var ev Event
				_ = json.Unmarshal(msg, &ev)
				if ev.Type == EventPlaybackCommand {
					t.Fatalf("%s received playback.command: %s", name, string(msg))
				}
			default:
				return
			}
		}
	}
	assertNoCommand("bystander", bystander)

	gotTakeOver := false
	for range 8 {
		select {
		case msg := <-idle.Send:
			var ev Event
			_ = json.Unmarshal(msg, &ev)
			if ev.Type != EventPlaybackCommand {
				continue
			}
			payload := asMap(ev.Payload)
			if payload["action"] == "take_over" {
				gotTakeOver = true
			}
		default:
		}
	}
	if !gotTakeOver {
		t.Fatal("idle missing take_over")
	}

	gotRelease := false
	for range 8 {
		select {
		case msg := <-active.Send:
			var ev Event
			_ = json.Unmarshal(msg, &ev)
			if ev.Type != EventPlaybackCommand {
				continue
			}
			payload := asMap(ev.Payload)
			if payload["action"] == "pause_release" {
				gotRelease = true
			}
		default:
		}
	}
	if !gotRelease {
		t.Fatal("active missing pause_release")
	}
}

func TestUnregisterStaleClientDoesNotWipeReplacement(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	scope := "server:local"

	oldConn := mockWSClient("", "phone", scope)
	newConn := mockWSClient("", "phone", scope)
	reg.Register(oldConn, "Phone", "iPhone")
	reg.Register(newConn, "Phone", "iPhone")

	// Old socket closes after replacement.
	reg.Unregister(oldConn)

	list := reg.List(scope)
	if len(list) != 1 {
		t.Fatalf("replacement device should remain, got %d", len(list))
	}
	if list[0].DeviceID != "phone" {
		t.Fatalf("unexpected device %q", list[0].DeviceID)
	}
}

func jsonUnmarshal(data []byte, ev *Event) error {
	return json.Unmarshal(data, ev)
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	raw, _ := json.Marshal(v)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}

func asInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	default:
		return 0, false
	}
}
