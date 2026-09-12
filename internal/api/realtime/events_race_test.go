// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package realtime

import (
	"sync"
	"testing"
	"time"
)

func TestEventHubConcurrentBroadcastDoesNotPanic(t *testing.T) {
	hub := NewEventHub()
	clients := make([]*WSClient, 0, 32)
	for range 32 {
		c := &WSClient{UserID: "u", Send: make(chan []byte, 4)}
		hub.Register(c)
		clients = append(clients, c)
	}

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := range 200 {
			hub.Broadcast(Event{Type: EventScanProgress, Payload: ScanProgressPayload{LibraryID: "lib", Processed: i}})
		}
	}()
	go func() {
		defer wg.Done()
		for _, c := range clients {
			hub.Unregister(c)
			hub.Register(c)
		}
	}()
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("event hub concurrent broadcast timed out")
	}

	if hub.ClientCount() < 0 {
		t.Fatal("client count should never be negative")
	}
}

func TestEventHubCrashSafeNilPayload(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()
	hub := NewEventHub()
	hub.Broadcast(Event{Type: "noop", Payload: nil})
	hub.Broadcast(Event{Type: "bad", Payload: make(chan int)})
}
