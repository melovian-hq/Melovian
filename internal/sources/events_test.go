// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"melovian/internal/api/realtime"
	"melovian/internal/store"
)

// streamSource is a fake EventStreamer that emits queued events then blocks.
type streamSource struct {
	fakeSource
	events   chan struct{ name string }
	started  atomic.Int32
	stoppedC chan struct{}
}

func (s *streamSource) StreamEvents(ctx context.Context, inst store.SourceInstance, emit EventEmit) error {
	s.started.Add(1)
	defer func() {
		if s.stoppedC != nil {
			close(s.stoppedC)
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-s.events:
			if !ok {
				return errors.New("stream closed")
			}
			if err := emit(ev.name, json.RawMessage(`{}`)); err != nil {
				return err
			}
		}
	}
}

func newBridgeFixture(t *testing.T, enabled map[string]bool) (*EventBridge, *realtime.EventHub, *store.InstanceStore, *Registry) {
	t.Helper()
	db := store.OpenTestDB(t)
	instances := store.NewInstanceStore(db)
	hub := realtime.NewEventHub()
	reg := New(func(id string) bool { return enabled[id] })
	bridge := NewEventBridge(reg, hub, instances)
	return bridge, hub, instances, reg
}

func wsClientFor(hub *realtime.EventHub, userID string) *realtime.WSClient {
	c := realtime.NewWSClient(userID, 16)
	hub.Register(c)
	return c
}

func awaitMessage(t *testing.T, c *realtime.WSClient) []byte {
	t.Helper()
	select {
	case msg := <-c.Send:
		return msg
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for ws message")
		return nil
	}
}

func TestEventBridgeStreamsToUser(t *testing.T) {
	enabled := map[string]bool{"navidrome": true}
	bridge, hub, instances, reg := newBridgeFixture(t, enabled)
	defer bridge.StopAll()

	src := &streamSource{
		fakeSource: fakeSource{id: "navidrome", caps: Capabilities{Events: true}},
		events:     make(chan struct{ name string }, 4),
	}
	reg.Register(src)

	inst, err := instances.CreateForUser("user-1", store.CreateInstanceInput{
		Name: "Navi", ServerURL: "http://x", Username: "u", Password: "p", SourceID: "navidrome",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := instances.SetActiveForUser("user-1", inst.ID); err != nil {
		t.Fatal(err)
	}
	client := wsClientFor(hub, "user-1")

	bridge.SyncUser("user-1")
	if !waitFor(func() bool { return src.started.Load() > 0 }, 3*time.Second) {
		t.Fatal("stream did not start")
	}

	src.events <- struct{ name string }{name: "scanStatus"}
	msg := awaitMessage(t, client)
	if !contains(string(msg), "source.navidrome.scanStatus") {
		t.Fatalf("unexpected event payload: %s", msg)
	}
}

func TestEventBridgeStopsOnDisable(t *testing.T) {
	enabled := map[string]bool{"navidrome": true}
	bridge, hub, instances, reg := newBridgeFixture(t, enabled)
	defer bridge.StopAll()

	src := &streamSource{
		fakeSource: fakeSource{id: "navidrome", caps: Capabilities{Events: true}},
		events:     make(chan struct{ name string }, 4),
	}
	reg.Register(src)

	inst, _ := instances.CreateForUser("user-1", store.CreateInstanceInput{
		Name: "Navi", ServerURL: "http://x", Username: "u", Password: "p", SourceID: "navidrome",
	})
	_ = instances.SetActiveForUser("user-1", inst.ID)
	_ = hub

	bridge.SyncUser("user-1")
	if !waitFor(func() bool { return src.started.Load() > 0 }, 3*time.Second) {
		t.Fatal("stream did not start")
	}

	// Disabling the extension must end the stream, not just skip events.
	enabled["navidrome"] = false
	src.events <- struct{ name string }{name: "scanStatus"}
	if !waitFor(func() bool {
		bridge.mu.Lock()
		defer bridge.mu.Unlock()
		return len(bridge.running) == 0
	}, 3*time.Second) {
		t.Fatal("stream still running after disable")
	}
}

func TestEventBridgeSkipsNonEventSources(t *testing.T) {
	enabled := map[string]bool{"subsonic": true}
	bridge, _, instances, reg := newBridgeFixture(t, enabled)
	defer bridge.StopAll()
	reg.Register(&fakeSource{id: "subsonic"})

	inst, _ := instances.CreateForUser("user-1", store.CreateInstanceInput{
		Name: "Sub", ServerURL: "http://x", Username: "u", Password: "p", SourceID: "subsonic",
	})
	_ = instances.SetActiveForUser("user-1", inst.ID)

	bridge.SyncUser("user-1")
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	if len(bridge.running) != 0 {
		t.Fatal("no stream should run for a non-event source")
	}
}

func TestEventBridgeKeepsAliveFiltered(t *testing.T) {
	enabled := map[string]bool{"navidrome": true}
	bridge, hub, instances, reg := newBridgeFixture(t, enabled)
	defer bridge.StopAll()

	src := &streamSource{
		fakeSource: fakeSource{id: "navidrome", caps: Capabilities{Events: true}},
		events:     make(chan struct{ name string }, 4),
	}
	reg.Register(src)

	inst, _ := instances.CreateForUser("user-1", store.CreateInstanceInput{
		Name: "Navi", ServerURL: "http://x", Username: "u", Password: "p", SourceID: "navidrome",
	})
	_ = instances.SetActiveForUser("user-1", inst.ID)
	client := wsClientFor(hub, "user-1")

	bridge.SyncUser("user-1")
	if !waitFor(func() bool { return src.started.Load() > 0 }, 3*time.Second) {
		t.Fatal("stream did not start")
	}
	src.events <- struct{ name string }{name: "keepAlive"}
	src.events <- struct{ name string }{name: "serverStart"}
	msg := awaitMessage(t, client)
	if contains(string(msg), "keepAlive") {
		t.Fatalf("keepAlive should be filtered, got %s", msg)
	}
}

func waitFor(cond func() bool, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return cond()
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
