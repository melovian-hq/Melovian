// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"sync"
	"sync/atomic"
)

const (
	EventScanProgress   = "scan.progress"
	EventScanComplete   = "scan.complete"
	EventScanError      = "scan.error"
	EventLibraryUpdated = "library.updated"
)

type Event struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type ScanProgressPayload struct {
	LibraryID string `json:"libraryId"`
	Name      string `json:"name,omitempty"`
	Processed int    `json:"processed"`
	Phase     string `json:"phase"`
}

type ScanCompletePayload struct {
	LibraryID  string `json:"libraryId"`
	Name       string `json:"name,omitempty"`
	TrackCount int    `json:"trackCount"`
}

type ScanErrorPayload struct {
	LibraryID string `json:"libraryId"`
	Name      string `json:"name,omitempty"`
	Error     string `json:"error"`
}

type LibraryUpdatedPayload struct {
	LibraryID      string `json:"libraryId"`
	Name           string `json:"name,omitempty"`
	TrackCount     int    `json:"trackCount"`
	MissingCount   int    `json:"missingCount"`
	DuplicateCount int    `json:"duplicateCount"`
}

type wsClient struct {
	userID string
	send   chan []byte

	// Identity fields are written on the connection read loop and read by
	// EventHub broadcasts on other goroutines. Use atomics to avoid races.
	deviceID   atomic.Value // string
	scopeKey   atomic.Value // string
	registered atomic.Bool
}

func (c *wsClient) getDeviceID() string {
	v, _ := c.deviceID.Load().(string)
	return v
}

func (c *wsClient) setDeviceID(id string) {
	c.deviceID.Store(id)
}

func (c *wsClient) getScopeKey() string {
	v, _ := c.scopeKey.Load().(string)
	return v
}

func (c *wsClient) setScopeKey(scope string) {
	c.scopeKey.Store(scope)
}

func (c *wsClient) setRegistered(v bool) {
	c.registered.Store(v)
}

type EventHub struct {
	mu      sync.Mutex
	clients map[*wsClient]struct{}
}

func NewEventHub() *EventHub {
	return &EventHub{clients: make(map[*wsClient]struct{})}
}

func (h *EventHub) register(c *wsClient) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *EventHub) unregister(c *wsClient) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

func (h *EventHub) ClientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

func (h *EventHub) Broadcast(event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.Lock()
	clients := make([]*wsClient, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()

	for _, c := range clients {
		select {
		case c.send <- payload:
		default:
		}
	}
}

func (h *EventHub) BroadcastToUser(userID string, event Event) {
	if userID == "" {
		h.Broadcast(event)
		return
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.Lock()
	clients := make([]*wsClient, 0)
	for c := range h.clients {
		if c.userID == userID {
			clients = append(clients, c)
		}
	}
	h.mu.Unlock()

	for _, c := range clients {
		select {
		case c.send <- payload:
		default:
		}
	}
}

func (h *EventHub) BroadcastToScope(scope string, event Event) {
	if scope == "" {
		h.Broadcast(event)
		return
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.Lock()
	clients := make([]*wsClient, 0)
	for c := range h.clients {
		if c.getScopeKey() == scope {
			clients = append(clients, c)
		}
	}
	h.mu.Unlock()

	for _, c := range clients {
		select {
		case c.send <- payload:
		default:
		}
	}
}

func (h *EventHub) SendToClient(c *wsClient, event Event) {
	if c == nil {
		return
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	select {
	case c.send <- payload:
	default:
	}
}
