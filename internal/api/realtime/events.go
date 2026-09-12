// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package realtime

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

// NewWSClient returns a websocket client record with a buffered send queue.
func NewWSClient(userID string, sendBuf int) *WSClient {
	return &WSClient{
		UserID: userID,
		Send:   make(chan []byte, sendBuf),
	}
}

type WSClient struct {
	UserID string
	Send   chan []byte

	// Identity fields are written on the connection read loop and read by
	// EventHub broadcasts on other goroutines. Use atomics to avoid races.
	deviceID   atomic.Value // string
	scopeKey   atomic.Value // string
	registered atomic.Bool
}

func (c *WSClient) DeviceID() string {
	v, _ := c.deviceID.Load().(string)
	return v
}

func (c *WSClient) SetDeviceID(id string) {
	c.deviceID.Store(id)
}

func (c *WSClient) ScopeKey() string {
	v, _ := c.scopeKey.Load().(string)
	return v
}

func (c *WSClient) SetScopeKey(scope string) {
	c.scopeKey.Store(scope)
}

func (c *WSClient) SetRegistered(v bool) {
	c.registered.Store(v)
}

type EventHub struct {
	mu      sync.Mutex
	clients map[*WSClient]struct{}
}

func NewEventHub() *EventHub {
	return &EventHub{clients: make(map[*WSClient]struct{})}
}

func (h *EventHub) Register(c *WSClient) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *EventHub) Unregister(c *WSClient) {
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
	clients := make([]*WSClient, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()

	for _, c := range clients {
		select {
		case c.Send <- payload:
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
	clients := make([]*WSClient, 0)
	for c := range h.clients {
		if c.UserID == userID {
			clients = append(clients, c)
		}
	}
	h.mu.Unlock()

	for _, c := range clients {
		select {
		case c.Send <- payload:
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
	clients := make([]*WSClient, 0)
	for c := range h.clients {
		if c.ScopeKey() == scope {
			clients = append(clients, c)
		}
	}
	h.mu.Unlock()

	for _, c := range clients {
		select {
		case c.Send <- payload:
		default:
		}
	}
}

func (h *EventHub) SendToClient(c *WSClient, event Event) {
	if c == nil {
		return
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	select {
	case c.Send <- payload:
	default:
	}
}
