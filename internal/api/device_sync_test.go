// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestResolveDeviceScope(t *testing.T) {
	if got := ResolveDeviceScope("u1", "d1", true); got != "user:u1" {
		t.Fatalf("auth scope = %q", got)
	}
	if got := ResolveDeviceScope("", "d1", false); got != "server:local" {
		t.Fatalf("no-auth scope = %q", got)
	}
	if got := ResolveDeviceScope("", "", true); got != "server:local" {
		t.Fatalf("auth without user falls back = %q", got)
	}
}

func TestGuessDeviceName(t *testing.T) {
	cases := map[string]string{
		"Mozilla/5.0 (iPhone; CPU iPhone OS)": "iPhone",
		"Mozilla/5.0 (iPad; CPU OS)":          "iPad",
		"Mozilla/5.0 (Linux; Android 14)":     "Android",
		"Mozilla/5.0 (Macintosh; Intel Mac)":  "Mac",
		"Mozilla/5.0 (Windows NT 10.0)":       "Windows",
		"Mozilla/5.0 (X11; Linux x86_64)":     "Linux",
		"UnknownClient/1.0":                   "Browser",
	}
	for ua, want := range cases {
		if got := guessDeviceName(ua); got != want {
			t.Fatalf("guessDeviceName(%q)=%q want %q", ua, got, want)
		}
	}
}

func mockWSClient(userID, deviceID, scope string) *wsClient {
	c := &wsClient{
		userID: userID,
		send:   make(chan []byte, 16),
	}
	c.setDeviceID(deviceID)
	c.setScopeKey(scope)
	return c
}

func TestDeviceRegistryRegisterAndList(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	scope := "server:local"

	a := mockWSClient("", "dev-a", scope)
	b := mockWSClient("", "dev-b", scope)
	reg.Register(a, "Phone", "iPhone")
	reg.Register(b, "Laptop", "Macintosh")

	list := reg.List(scope)
	if len(list) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(list))
	}
}

func TestDeviceRegistryScopeIsolationAdversarial(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)

	alice := mockWSClient("alice", "d1", "user:alice")
	bob := mockWSClient("bob", "d2", "user:bob")
	reg.Register(alice, "Alice Phone", "iPhone")
	reg.Register(bob, "Bob Laptop", "Windows")

	aliceDevices := reg.List("user:alice")
	bobDevices := reg.List("user:bob")
	if len(aliceDevices) != 1 || aliceDevices[0].DeviceID != "d1" {
		t.Fatalf("alice leaked peers: %+v", aliceDevices)
	}
	if len(bobDevices) != 1 || bobDevices[0].DeviceID != "d2" {
		t.Fatalf("bob leaked peers: %+v", bobDevices)
	}
}

func TestDeviceRegistryTakeOverHandoff(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	scope := "server:local"

	active := mockWSClient("", "active", scope)
	idle := mockWSClient("", "idle", scope)
	hub.register(active)
	hub.register(idle)
	reg.Register(active, "Active", "Linux")
	reg.Register(idle, "Idle", "Windows")

	reg.UpdatePlayback(active, PlaybackSnapshot{
		TrackID:    "t1",
		PositionMs: 42_000,
		Paused:     false,
		QueueIDs:   []string{"t1", "t2"},
		QueueIndex: 0,
	})

	reg.TakeOver(idle)

	list := reg.List(scope)
	var idleInfo, activeInfo DeviceInfo
	for _, d := range list {
		if d.DeviceID == "idle" {
			idleInfo = d
		}
		if d.DeviceID == "active" {
			activeInfo = d
		}
	}
	if !idleInfo.IsActivePlayer {
		t.Fatal("idle device should become active after takeOver")
	}
	if activeInfo.IsActivePlayer {
		t.Fatal("previous active should release ownership")
	}

	drainCommands := func(c *wsClient) []string {
		var actions []string
		for {
			select {
			case msg := <-c.send:
				var ev Event
				_ = json.Unmarshal(msg, &ev)
				if ev.Type != EventPlaybackCommand {
					continue
				}
				payload, _ := json.Marshal(ev.Payload)
				var p map[string]any
				_ = json.Unmarshal(payload, &p)
				if action, ok := p["action"].(string); ok {
					actions = append(actions, action)
				}
			default:
				return actions
			}
		}
	}

	activeActions := drainCommands(active)
	idleActions := drainCommands(idle)
	foundRelease := false
	for _, a := range activeActions {
		if a == "pause_release" {
			foundRelease = true
		}
	}
	if !foundRelease {
		t.Fatalf("expected pause_release on previous active, got %v", activeActions)
	}
	foundTakeOver := false
	for _, a := range idleActions {
		if a == "take_over" {
			foundTakeOver = true
		}
	}
	if !foundTakeOver {
		t.Fatalf("expected take_over on requester, got %v", idleActions)
	}
}

func TestDeviceRegistryListenTogetherSession(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	scope := "server:local"

	host := mockWSClient("", "host", scope)
	guest := mockWSClient("", "guest", scope)
	hub.register(host)
	hub.register(guest)
	reg.Register(host, "Host", "Mac")
	reg.Register(guest, "Guest", "Linux")

	sessionID := reg.CreateSession(host)
	if sessionID == "" {
		t.Fatal("expected session id")
	}
	if !reg.JoinSession(guest, sessionID) {
		t.Fatal("guest should join host session")
	}

	list := reg.List(scope)
	for _, d := range list {
		if d.SessionID != sessionID {
			t.Fatalf("device %s missing session", d.DeviceID)
		}
	}

	reg.LeaveSession(host)
	list = reg.List(scope)
	for _, d := range list {
		if d.SessionID != "" {
			t.Fatalf("session should end when host leaves, still on %s", d.DeviceID)
		}
	}
}

func TestAdvancePlaybackSnapshot(t *testing.T) {
	paused := advancePlaybackSnapshot(PlaybackSnapshot{
		PositionMs: 10_000,
		Paused:     true,
		ServerTime: 1_000,
	}, 4_000)
	if paused.PositionMs != 10_000 {
		t.Fatalf("paused snapshot should stay put, got %d", paused.PositionMs)
	}
	if paused.ServerTime != 4_000 {
		t.Fatalf("paused snapshot should refresh server time, got %d", paused.ServerTime)
	}

	playing := advancePlaybackSnapshot(PlaybackSnapshot{
		PositionMs: 10_000,
		DurationMs: 12_000,
		Paused:     false,
		ServerTime: 1_000,
	}, 4_000)
	if playing.PositionMs != 12_000 {
		t.Fatalf("playing snapshot should clamp to duration, got %d", playing.PositionMs)
	}

	mid := advancePlaybackSnapshot(PlaybackSnapshot{
		PositionMs: 8_000,
		DurationMs: 60_000,
		Paused:     false,
		ServerTime: 1_000,
	}, 4_000)
	if mid.PositionMs != 11_000 {
		t.Fatalf("playing snapshot should add elapsed, got %d", mid.PositionMs)
	}
}

func TestJoinSessionAdvancesHostPosition(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	scope := "server:local"
	host := mockWSClient("", "host", scope)
	guest := mockWSClient("", "guest", scope)
	hub.register(host)
	hub.register(guest)
	reg.Register(host, "Host", "Mac")
	reg.Register(guest, "Guest", "Linux")

	reg.UpdatePlayback(host, PlaybackSnapshot{
		TrackID:    "t1",
		PositionMs: 10_000,
		DurationMs: 180_000,
		Paused:     false,
	})
	reg.mu.Lock()
	hostDev := reg.devices[deviceKey(scope, "host")]
	hostDev.info.Playback.PositionMs = 10_000
	hostDev.info.Playback.ServerTime = time.Now().UnixMilli() - 2500
	reg.mu.Unlock()
	id := reg.CreateSession(host)
	drain := func(c *wsClient) {
		for {
			select {
			case <-c.send:
			default:
				return
			}
		}
	}
	drain(host)
	drain(guest)

	if !reg.JoinSession(guest, id) {
		t.Fatal("guest should join")
	}

	deadline := time.After(2 * time.Second)
	for {
		select {
		case msg := <-guest.send:
			var ev Event
			if err := json.Unmarshal(msg, &ev); err != nil {
				continue
			}
			if ev.Type != EventSessionUpdated {
				continue
			}
			payload, _ := json.Marshal(ev.Payload)
			var p map[string]any
			_ = json.Unmarshal(payload, &p)
			if p["action"] != "joined" {
				continue
			}
			state, _ := p["state"].(map[string]any)
			if state == nil {
				t.Fatal("join should include host playback state")
			}
			pos, _ := state["positionMs"].(float64)
			if pos < 12_000 {
				t.Fatalf("join position should include elapsed time, got %v", pos)
			}
			return
		case <-deadline:
			t.Fatal("guest did not receive join snapshot")
		}
	}
}

func TestDeviceRegistryRemoteCommandTargetsActive(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	scope := "server:local"

	active := mockWSClient("", "active", scope)
	remote := mockWSClient("", "remote", scope)
	hub.register(active)
	hub.register(remote)
	reg.Register(active, "Active", "Linux")
	reg.Register(remote, "Remote", "Windows")
	reg.UpdatePlayback(active, PlaybackSnapshot{TrackID: "t1", Paused: false})

	drain := func(c *wsClient) {
		for {
			select {
			case <-c.send:
			default:
				return
			}
		}
	}
	drain(active)
	drain(remote)

	reg.Command(remote, "pause", "", nil)

	deadline := time.After(2 * time.Second)
	for {
		select {
		case msg := <-active.send:
			var ev Event
			if err := json.Unmarshal(msg, &ev); err != nil {
				continue
			}
			if ev.Type != EventPlaybackCommand {
				continue
			}
			payload, _ := json.Marshal(ev.Payload)
			var p map[string]any
			_ = json.Unmarshal(payload, &p)
			if p["action"] == "pause" && p["targetId"] == "active" {
				return
			}
		case <-deadline:
			t.Fatal("active did not receive pause command")
		}
	}
}

func TestDeviceRegistryGuestUnregisterKeepsSession(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	scope := "server:local"
	host := mockWSClient("", "host", scope)
	guest := mockWSClient("", "guest", scope)
	reg.Register(host, "Host", "Mac")
	reg.Register(guest, "Guest", "Linux")
	id := reg.CreateSession(host)
	if !reg.JoinSession(guest, id) {
		t.Fatal("guest should join")
	}

	reg.Unregister(guest)
	list := reg.List(scope)
	if len(list) != 1 {
		t.Fatalf("expected host remaining, got %d", len(list))
	}
	if list[0].DeviceID != "host" {
		t.Fatalf("expected host, got %s", list[0].DeviceID)
	}
	if list[0].SessionID != id {
		t.Fatal("host session must survive guest disconnect")
	}
	if !list[0].IsHost {
		t.Fatal("host should remain host")
	}
}

func TestDeviceRegistryHandoffToPushesPlayback(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	scope := "server:local"

	source := mockWSClient("", "source", scope)
	target := mockWSClient("", "target", scope)
	hub.register(source)
	hub.register(target)
	reg.Register(source, "Laptop", "Linux")
	reg.Register(target, "Phone", "iPhone")

	reg.UpdatePlayback(source, PlaybackSnapshot{
		TrackID:    "t1",
		PositionMs: 12_000,
		Paused:     false,
		QueueIDs:   []string{"t1", "t2"},
		QueueIndex: 0,
	})

	if code := reg.HandoffTo(source, "target"); code != "" {
		t.Fatalf("handoff failed: %s", code)
	}

	list := reg.List(scope)
	var sourceInfo, targetInfo DeviceInfo
	for _, d := range list {
		if d.DeviceID == "source" {
			sourceInfo = d
		}
		if d.DeviceID == "target" {
			targetInfo = d
		}
	}
	if sourceInfo.IsActivePlayer {
		t.Fatal("source should release after handoff")
	}
	if !targetInfo.IsActivePlayer {
		t.Fatal("target should become active after handoff")
	}

	drainCommands := func(c *wsClient) []string {
		var actions []string
		for {
			select {
			case msg := <-c.send:
				var ev Event
				_ = json.Unmarshal(msg, &ev)
				if ev.Type != EventPlaybackCommand {
					continue
				}
				payload, _ := json.Marshal(ev.Payload)
				var p map[string]any
				_ = json.Unmarshal(payload, &p)
				if action, ok := p["action"].(string); ok {
					actions = append(actions, action)
				}
			default:
				return actions
			}
		}
	}

	sourceActions := drainCommands(source)
	targetActions := drainCommands(target)
	foundRelease := false
	for _, a := range sourceActions {
		if a == "pause_release" {
			foundRelease = true
		}
	}
	if !foundRelease {
		t.Fatalf("expected pause_release on source, got %v", sourceActions)
	}
	foundTakeOver := false
	for _, a := range targetActions {
		if a == "take_over" {
			foundTakeOver = true
		}
	}
	if !foundTakeOver {
		t.Fatalf("expected take_over on target, got %v", targetActions)
	}
}

func TestDeviceRegistryHandoffToMissingTarget(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	scope := "server:local"
	source := mockWSClient("", "source", scope)
	reg.Register(source, "Laptop", "Linux")
	reg.UpdatePlayback(source, PlaybackSnapshot{TrackID: "t1", Paused: false})
	if code := reg.HandoffTo(source, "ghost"); code != "target_offline" {
		t.Fatalf("code = %q want target_offline", code)
	}
}

func TestDeviceRegistryTakeOverNothingPlaying(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	scope := "server:local"
	idle := mockWSClient("", "idle", scope)
	reg.Register(idle, "Idle", "Linux")
	if code := reg.TakeOver(idle); code != "nothing_playing" {
		t.Fatalf("code = %q want nothing_playing", code)
	}
}

func TestDeviceRegistryHostLeaveBroadcastsEnded(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	scope := "server:local"
	host := mockWSClient("", "host", scope)
	guest := mockWSClient("", "guest", scope)
	hub.register(host)
	hub.register(guest)
	reg.Register(host, "Host", "Mac")
	reg.Register(guest, "Guest", "Linux")
	id := reg.CreateSession(host)
	if !reg.JoinSession(guest, id) {
		t.Fatal("join failed")
	}

	drain := func(c *wsClient) {
		for {
			select {
			case <-c.send:
			default:
				return
			}
		}
	}
	drain(host)
	drain(guest)

	reg.LeaveSession(host)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case msg := <-guest.send:
			var ev Event
			if err := json.Unmarshal(msg, &ev); err != nil {
				continue
			}
			if ev.Type != EventSessionUpdated {
				continue
			}
			payload, _ := json.Marshal(ev.Payload)
			var p map[string]any
			_ = json.Unmarshal(payload, &p)
			if p["action"] == "ended" && p["sessionId"] == id {
				return
			}
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	t.Fatal("guest did not receive session ended")
}

func TestDeviceErrorMessageHasNoSecrets(t *testing.T) {
	codes := []string{
		"not_registered",
		"target_offline",
		"nothing_playing",
		"missing_target",
		"session_missing",
		"session_unavailable",
		"unknown",
	}
	for _, code := range codes {
		msg := deviceErrorMessage(code)
		if msg == "" {
			t.Fatalf("empty message for %s", code)
		}
		lower := strings.ToLower(msg)
		for _, banned := range []string{"token", "password", "http://", "127.0.0.1", "user:"} {
			if strings.Contains(lower, banned) {
				t.Fatalf("message for %s leaked %q: %s", code, banned, msg)
			}
		}
	}
}

func TestDeviceRegistryPlaybackCoverArtRoundtrip(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	scope := "server:local"

	player := mockWSClient("", "player", scope)
	reg.Register(player, "Player", "Linux")
	reg.UpdatePlayback(player, PlaybackSnapshot{
		TrackID:    "t1",
		TrackTitle: "Song",
		ArtistName: "Artist",
		CoverArt:   "cover-42",
		PositionMs: 1000,
		Paused:     false,
	})

	list := reg.List(scope)
	if len(list) != 1 {
		t.Fatalf("expected 1 device, got %d", len(list))
	}
	if list[0].Playback == nil {
		t.Fatal("expected playback snapshot")
	}
	if list[0].Playback.CoverArt != "cover-42" {
		t.Fatalf("coverArt = %q want cover-42", list[0].Playback.CoverArt)
	}
}

func TestJoinSessionByTokenCrossScope(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	reg.lookupUsername = func(userID string) string {
		switch userID {
		case "alice":
			return "alice"
		case "bob":
			return "bob"
		default:
			return ""
		}
	}

	alice := mockWSClient("alice", "host", "user:alice")
	bob := mockWSClient("bob", "guest", "user:bob")
	hub.register(alice)
	hub.register(bob)
	reg.Register(alice, "Alice Phone", "iPhone")
	reg.Register(bob, "Bob Laptop", "Windows")

	sessionID := reg.CreateSession(alice)
	if sessionID == "" {
		t.Fatal("expected session id")
	}
	if reg.JoinSession(bob, sessionID) {
		t.Fatal("same-account JoinSession must reject cross-scope guests")
	}

	gotSessionID, token, ok := reg.EnsureInviteToken("user:alice", "host")
	if !ok || token == "" || gotSessionID != sessionID {
		t.Fatalf("EnsureInviteToken failed: ok=%v token=%q id=%q", ok, token, gotSessionID)
	}

	joinedID, snap, ok := reg.JoinSessionByToken("user:bob", "guest", "bob", token)
	if !ok || joinedID != sessionID {
		t.Fatalf("JoinSessionByToken failed: ok=%v id=%q", ok, joinedID)
	}
	_ = snap

	status, found := reg.SessionStatus(sessionID)
	if !found {
		t.Fatal("expected party status")
	}
	if !status.CrossUser {
		t.Fatal("expected crossUser session")
	}
	if len(status.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(status.Members))
	}
	if !reg.IsSessionMember(sessionID, "bob", "user:bob", "guest") {
		t.Fatal("bob should be a session member")
	}
	if !reg.IsSessionMember(sessionID, "alice", "user:alice", "host") {
		t.Fatal("alice should be a session member")
	}

	// Drain bob's inbox then confirm host playback fans out across scopes.
	for {
		select {
		case <-bob.send:
		default:
			goto drained
		}
	}
drained:

	reg.UpdatePlayback(alice, PlaybackSnapshot{
		TrackID:    "track-1",
		TrackTitle: "Song",
		PositionMs: 5_000,
		Paused:     false,
	})

	gotTogether := false
	deadline := time.After(2 * time.Second)
	for !gotTogether {
		select {
		case msg := <-bob.send:
			var ev Event
			if err := json.Unmarshal(msg, &ev); err != nil {
				continue
			}
			if ev.Type != EventPlaybackState {
				continue
			}
			payload, _ := json.Marshal(ev.Payload)
			var p map[string]any
			_ = json.Unmarshal(payload, &p)
			if p["together"] == true && p["sessionId"] == sessionID {
				gotTogether = true
			}
		case <-deadline:
			t.Fatal("bob did not receive cross-user playback.state")
		}
	}

	reg.LeaveSession(alice)
	if _, still := reg.GetSession(sessionID); still {
		t.Fatal("session should end when host leaves")
	}
}

func TestInviteToSessionPullsSameScopeDevice(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	scope := "user:alice"

	host := mockWSClient("alice", "host", scope)
	guest := mockWSClient("alice", "phone", scope)
	hub.register(host)
	hub.register(guest)
	reg.Register(host, "Laptop", "Linux")
	reg.Register(guest, "Phone", "iPhone")

	sessionID := reg.CreateSession(host)
	if sessionID == "" {
		t.Fatal("expected session")
	}
	if code := reg.InviteToSession(host, "phone"); code != "" {
		t.Fatalf("InviteToSession: %s", code)
	}
	list := reg.List(scope)
	var phone DeviceInfo
	for _, d := range list {
		if d.DeviceID == "phone" {
			phone = d
		}
	}
	if phone.SessionID != sessionID || phone.IsHost {
		t.Fatalf("phone session=%q host=%v", phone.SessionID, phone.IsHost)
	}
	if code := reg.InviteToSession(host, "phone"); code != "already_member" {
		t.Fatalf("expected already_member, got %q", code)
	}
	if code := reg.InviteToSession(guest, "host"); code != "not_hosting" {
		t.Fatalf("expected not_hosting from guest, got %q", code)
	}
}

func TestPeekPartyLeaveNoticeGuestAndHost(t *testing.T) {
	hub := NewEventHub()
	reg := NewDeviceRegistry(hub)
	reg.lookupUsername = func(userID string) string {
		if userID == "alice" {
			return "alice"
		}
		if userID == "bob" {
			return "bob"
		}
		return ""
	}

	alice := mockWSClient("alice", "host", "user:alice")
	bob := mockWSClient("bob", "guest", "user:bob")
	hub.register(alice)
	hub.register(bob)
	reg.Register(alice, "Alice Phone", "iPhone")
	reg.Register(bob, "Bob Laptop", "Windows")

	sessionID := reg.CreateSession(alice)
	_, token, ok := reg.EnsureInviteToken("user:alice", "host")
	if !ok || token == "" {
		t.Fatal("expected invite token")
	}
	if _, _, ok := reg.JoinSessionByToken("user:bob", "guest", "bob", token); !ok {
		t.Fatal("bob should join")
	}

	guestNotice := reg.PeekPartyLeaveNotice("user:bob", "guest")
	if guestNotice == nil || guestNotice.Kind != "left" {
		t.Fatalf("expected guest left notice, got %#v", guestNotice)
	}
	if len(guestNotice.NotifyIDs) != 1 || guestNotice.NotifyIDs[0] != "alice" {
		t.Fatalf("guest leave should notify host, got %v", guestNotice.NotifyIDs)
	}
	if guestNotice.ActorUser != "bob" && guestNotice.ActorName == "" {
		t.Fatal("expected actor labels on guest leave")
	}

	hostNotice := reg.PeekPartyLeaveNotice("user:alice", "host")
	if hostNotice == nil || hostNotice.Kind != "ended" {
		t.Fatalf("expected host ended notice, got %#v", hostNotice)
	}
	if len(hostNotice.NotifyIDs) != 1 || hostNotice.NotifyIDs[0] != "bob" {
		t.Fatalf("host end should notify guest, got %v", hostNotice.NotifyIDs)
	}
	if hostNotice.SessionID != sessionID {
		t.Fatalf("session id = %q want %q", hostNotice.SessionID, sessionID)
	}
}
