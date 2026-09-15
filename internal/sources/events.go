// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"melovian/internal/api/realtime"
	"melovian/internal/store"
)

// EventEmit receives one upstream event: a name and the raw payload.
type EventEmit func(name string, data json.RawMessage) error

// EventStreamer is a source that can hold a long-lived event connection to
// its server (Navidrome SSE). The bridge owns reconnects and shutdown.
type EventStreamer interface {
	Source
	StreamEvents(ctx context.Context, inst store.SourceInstance, emit EventEmit) error
}

// EventBridge mirrors upstream source events onto the app websocket hub.
// One stream runs per active (user, instance) pair. Streams restart on
// failure with backoff and stop when the instance is deactivated, the
// source is disabled, or the source does not stream events.
type EventBridge struct {
	reg       *Registry
	hub       *realtime.EventHub
	instances *store.InstanceStore

	mu      sync.Mutex
	running map[string]*eventStream
}

type eventStream struct {
	key    string
	cancel context.CancelFunc
}

// NewEventBridge wires a bridge over the registry and hub.
func NewEventBridge(reg *Registry, hub *realtime.EventHub, instances *store.InstanceStore) *EventBridge {
	return &EventBridge{
		reg:       reg,
		hub:       hub,
		instances: instances,
		running:   map[string]*eventStream{},
	}
}

// SyncUser reconciles the event stream for one user against their active
// instance. Safe to call after any activation, delete, or disable change.
func (b *EventBridge) SyncUser(userID string) {
	inst, err := b.instances.GetActiveForUser(userID)
	if err != nil {
		b.stopUser(userID)
		return
	}
	b.syncInstance(userID, inst)
}

// SyncAll reconciles streams for every user with an active instance plus
// the legacy global selection. Called once at server start.
func (b *EventBridge) SyncAll() {
	pairs, err := b.instances.ListActivePairs()
	if err != nil {
		slog.Warn("source events: list active pairs failed", "err", err)
		return
	}
	for userID := range pairs {
		b.SyncUser(userID)
	}
}

// StopAll cancels every running stream. Called at server shutdown.
func (b *EventBridge) StopAll() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for key, stream := range b.running {
		stream.cancel()
		delete(b.running, key)
	}
}

// syncInstance starts or replaces the user's stream when the active
// instance changed, or stops it when the source cannot stream.
func (b *EventBridge) syncInstance(userID string, inst store.SourceInstance) {
	src, err := b.reg.Available(inst)
	if err != nil {
		b.stopUser(userID)
		return
	}
	streamer, ok := src.(EventStreamer)
	if !ok || !src.Caps().Events {
		b.stopUser(userID)
		return
	}

	key := userID + "|" + inst.ID
	b.mu.Lock()
	if _, ok := b.running[key]; ok {
		b.mu.Unlock()
		return
	}
	// Same user pointing at a different instance gets a fresh stream.
	for k, stream := range b.running {
		if streamUser(k) == userID {
			stream.cancel()
			delete(b.running, k)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	b.running[key] = &eventStream{key: key, cancel: cancel}
	b.mu.Unlock()

	go b.run(ctx, key, userID, streamer, inst)
}

// run loops the stream with exponential backoff until cancelled or the
// source is disabled.
func (b *EventBridge) run(ctx context.Context, key, userID string, streamer EventStreamer, inst store.SourceInstance) {
	defer func() {
		b.mu.Lock()
		if current, ok := b.running[key]; ok && current.key == key {
			delete(b.running, key)
		}
		b.mu.Unlock()
	}()

	backoff := 1 * time.Second
	for {
		if ctx.Err() != nil || !b.reg.Enabled(streamer.ID()) {
			return
		}
		emit := func(name string, data json.RawMessage) error {
			if !b.reg.Enabled(streamer.ID()) {
				return ErrSourceDisabled
			}
			if name == "" || name == "keepAlive" {
				return nil
			}
			b.hub.BroadcastToUser(userID, realtime.Event{
				Type:    "source." + streamer.ID() + "." + name,
				Payload: data,
			})
			return nil
		}
		err := streamer.StreamEvents(ctx, inst, emit)
		if ctx.Err() != nil || errors.Is(err, ErrSourceDisabled) {
			return
		}
		if errors.Is(err, io.EOF) {
			// Clean stream end, reconnect soon.
			backoff = 1 * time.Second
		}
		slog.Warn("source events stream ended",
			"source", streamer.ID(),
			"instance", inst.ID,
			"err", err,
		)
		b.reg.ReportResult(inst.ID, err)

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

// stopUser cancels whatever stream is running for a user.
func (b *EventBridge) stopUser(userID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for k, stream := range b.running {
		if streamUser(k) == userID {
			stream.cancel()
			delete(b.running, k)
		}
	}
}

func streamUser(key string) string {
	user, _, _ := strings.Cut(key, "|")
	return user
}
