// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package navidrome

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// EventsPath is the Navidrome server-sent events endpoint. Authentication
// rides in the jwt query parameter because EventSource cannot set headers.
const EventsPath = "/api/events"

// Token returns a valid JWT for native API calls, logging in when needed.
func (c *Client) Token(ctx context.Context) (string, error) {
	c.mu.Lock()
	token := strings.TrimSpace(c.token)
	c.mu.Unlock()
	if token != "" {
		return token, nil
	}
	if err := c.Login(ctx); err != nil {
		return "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return strings.TrimSpace(c.token), nil
}

// StreamEvents connects to the Navidrome SSE endpoint and invokes emit for
// each event until the stream ends or ctx is cancelled. A nil error with
// ctx.Err() set means caller-driven shutdown.
func (c *Client) StreamEvents(ctx context.Context, emit func(name string, data json.RawMessage) error) error {
	token, err := c.Token(ctx)
	if err != nil {
		return err
	}

	endpoint := c.baseURL + EventsPath + "?jwt=" + url.QueryEscape(token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	// No client timeout on the request itself; SSE is long-lived and ctx
	// cancellation drives shutdown.
	streamClient := &http.Client{}
	resp, err := streamClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		c.mu.Lock()
		c.token = ""
		c.mu.Unlock()
		return ErrNotAuthenticated
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("navidrome events stream failed: status %d", resp.StatusCode)
	}

	return readSSE(resp.Body, emit)
}

// readSSE parses a text/event-stream body. The event name comes from the
// SSE event field, or from a name member inside a JSON data payload, which
// is how Navidrome frames its broker events.
func readSSE(body io.Reader, emit func(name string, data json.RawMessage) error) error {
	scanner := bufio.NewScanner(body)
	// Navidrome payloads stay small, but refreshResource can embed a list
	// of ids, so allow a generous ceiling.
	scanner.Buffer(make([]byte, 0, 64<<10), 4<<20)

	var eventName string
	var dataLines []string
	dispatch := func() error {
		if len(dataLines) == 0 {
			eventName = ""
			return nil
		}
		data := strings.Join(dataLines, "\n")
		dataLines = nil
		name := eventName
		eventName = ""
		raw := json.RawMessage(data)
		if name == "" {
			var framed struct {
				Name string          `json:"name"`
				Data json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(raw, &framed); err == nil {
				name = framed.Name
				if len(framed.Data) > 0 {
					raw = framed.Data
				}
			}
		}
		return emit(name, raw)
	}

	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		switch {
		case line == "":
			if err := dispatch(); err != nil {
				return err
			}
		case strings.HasPrefix(line, ":"):
			// comment or keepalive line
		case strings.HasPrefix(line, "event:"):
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return io.EOF
}

// KeepAliveName is the Navidrome heartbeat event, sent roughly every 15s.
const KeepAliveName = "keepAlive"

// ReconnectBounds for the events stream retry loop.
const (
	EventsMinBackoff = 1 * time.Second
	EventsMaxBackoff = 30 * time.Second
)
