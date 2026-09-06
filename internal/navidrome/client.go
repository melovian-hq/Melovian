// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package navidrome

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"melovian/internal/httputil"
)

const authHeader = "X-ND-Authorization"

var (
	ErrNotNavidrome     = errors.New("smart playlists require Navidrome")
	ErrNotAuthenticated = errors.New("navidrome authentication failed")
)

type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	mu         sync.Mutex
	token      string
}

type CreateSmartPlaylistRequest struct {
	Name    string         `json:"name"`
	Comment string         `json:"comment,omitempty"`
	Public  bool           `json:"public,omitempty"`
	Rules   map[string]any `json:"rules"`
}

type Playlist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func NewClient(serverURL, username, password string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(serverURL, "/"),
		username:   username,
		password:   password,
		httpClient: httputil.NewRetryHTTPClient(),
	}
}

func IsNavidromeServer(serverName, version string) bool {
	haystack := strings.ToLower(strings.TrimSpace(serverName) + " " + strings.TrimSpace(version))
	if strings.Contains(haystack, "navidrome") {
		return true
	}
	// OpenSubsonic type field is often just "navidrome" with a 0.x server version.
	name := strings.ToLower(strings.TrimSpace(serverName))
	if name == "nd" || strings.HasPrefix(name, "navidrome/") {
		return true
	}
	return false
}

func (c *Client) Login(ctx context.Context) error {
	body, err := json.Marshal(map[string]string{
		"username": c.username,
		"password": c.password,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/auth/login",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	payload, err := httputil.ReadLimited(resp.Body, 4<<20)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return ErrNotAuthenticated
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("navidrome login failed: %s", strings.TrimSpace(string(payload)))
	}

	var parsed struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return fmt.Errorf("parse navidrome login response: %w", err)
	}
	if strings.TrimSpace(parsed.Token) == "" {
		return fmt.Errorf("navidrome login returned no token")
	}
	c.mu.Lock()
	c.token = parsed.Token
	c.mu.Unlock()
	return nil
}

func (c *Client) CreateSmartPlaylist(
	ctx context.Context,
	req CreateSmartPlaylistRequest,
) (Playlist, error) {
	c.mu.Lock()
	token := strings.TrimSpace(c.token)
	c.mu.Unlock()
	if token == "" {
		if err := c.Login(ctx); err != nil {
			return Playlist{}, err
		}
		c.mu.Lock()
		token = strings.TrimSpace(c.token)
		c.mu.Unlock()
	}

	body, err := json.Marshal(req)
	if err != nil {
		return Playlist{}, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/playlist",
		bytes.NewReader(body),
	)
	if err != nil {
		return Playlist{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set(authHeader, "Bearer "+token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return Playlist{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	payload, err := httputil.ReadLimited(resp.Body, 4<<20)
	if err != nil {
		return Playlist{}, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		c.mu.Lock()
		c.token = ""
		c.mu.Unlock()
		return Playlist{}, ErrNotAuthenticated
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Playlist{}, fmt.Errorf(
			"navidrome create playlist failed: %s",
			strings.TrimSpace(string(payload)),
		)
	}

	var playlist Playlist
	if err := json.Unmarshal(payload, &playlist); err != nil {
		return Playlist{}, fmt.Errorf("parse navidrome playlist response: %w", err)
	}
	if strings.TrimSpace(playlist.ID) == "" {
		return Playlist{}, fmt.Errorf("navidrome create playlist returned no id")
	}
	return playlist, nil
}

func (c *Client) PingNavidrome(ctx context.Context) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	if err := c.Login(ctx); err != nil {
		if errors.Is(err, ErrNotAuthenticated) {
			return false, err
		}
		return false, nil
	}
	return true, nil
}
