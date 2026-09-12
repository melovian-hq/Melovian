// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lastfm

import (
	"context"
	"crypto/md5" //#nosec G501 -- Last.fm API signatures require MD5 per protocol spec
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"melovian/internal/httputil"
)

const DefaultBaseURL = "https://ws.audioscrobbler.com/2.0/"

// Track holds the metadata needed for a Last.fm scrobble.
type Track struct {
	ID              string
	Title           string
	Artist          string
	Album           string
	DurationSeconds int
}

// Client submits listens to the Last.fm Audioscrobbler 2.0 API.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// NewClient returns a Last.fm client using the supplied HTTP client.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		BaseURL: DefaultBaseURL,
		HTTP:    httpClient,
	}
}

func (c *Client) base() string {
	base := strings.TrimSpace(c.BaseURL)
	if base == "" {
		return DefaultBaseURL
	}
	return strings.TrimRight(base, "/") + "/"
}

func sign(params url.Values, secret string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		for _, v := range params[k] {
			sb.WriteString(k)
			sb.WriteString(v)
		}
	}
	sb.WriteString(secret)
	sum := md5.Sum([]byte(sb.String())) //#nosec G401 -- Last.fm API mandated signature = md5(sorted params + secret)
	return hex.EncodeToString(sum[:])
}

type lastFMResponse struct {
	Error   int    `json:"error"`
	Message string `json:"message"`
	User    *struct {
		Name string `json:"name"`
	} `json:"user"`
}

func (c *Client) do(ctx context.Context, params url.Values, apiSecret string) error {
	signed := url.Values{}
	maps.Copy(signed, params)
	signed.Set("api_sig", sign(signed, apiSecret))
	// The signed URL carries api_key, sk, and api_sig. Never put it in an
	// error string; these errors reach API responses and logs.
	u := c.base() + "?" + signed.Encode() + "&format=json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return httputil.SanitizeErrorURL(err)
	}
	defer resp.Body.Close()
	var body lastFMResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&body); err != nil {
		return fmt.Errorf("last.fm request failed: status %d", resp.StatusCode)
	}
	if body.Error != 0 {
		return fmt.Errorf("last.fm error %d: %s", body.Error, body.Message)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("last.fm request failed: status %d", resp.StatusCode)
	}
	return nil
}

// ValidateToken checks whether the configured session key is active.
func (c *Client) ValidateToken(ctx context.Context, apiKey, apiSecret, sessionKey string) (string, error) {
	params := url.Values{
		"method":  []string{"user.getInfo"},
		"api_key": []string{apiKey},
		"sk":      []string{sessionKey},
	}
	signed := url.Values{}
	maps.Copy(signed, params)
	signed.Set("api_sig", sign(signed, apiSecret))
	u := c.base() + "?" + signed.Encode() + "&format=json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", httputil.SanitizeErrorURL(err)
	}
	defer resp.Body.Close()
	var body lastFMResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&body); err != nil {
		return "", fmt.Errorf("last.fm validation failed")
	}
	if resp.StatusCode >= 400 && body.Error == 0 {
		return "", fmt.Errorf("last.fm validation returned status %d", resp.StatusCode)
	}
	if body.Error != 0 {
		return "", fmt.Errorf("last.fm error %d: %s", body.Error, body.Message)
	}
	if body.User == nil || body.User.Name == "" {
		return "", fmt.Errorf("last.fm validation returned no user")
	}
	return body.User.Name, nil
}

// UpdateNowPlaying reports the currently playing track.
func (c *Client) UpdateNowPlaying(ctx context.Context, apiKey, apiSecret, sessionKey string, track Track) error {
	params := url.Values{
		"method":   []string{"track.updateNowPlaying"},
		"api_key":  []string{apiKey},
		"sk":       []string{sessionKey},
		"artist":   []string{track.Artist},
		"track":    []string{track.Title},
		"album":    []string{track.Album},
		"duration": []string{fmt.Sprintf("%d", track.DurationSeconds)},
	}
	return c.do(ctx, params, apiSecret)
}

// Scrobble reports a completed track listen.
func (c *Client) Scrobble(ctx context.Context, apiKey, apiSecret, sessionKey string, track Track, listenedAt time.Time) error {
	params := url.Values{
		"method":    []string{"track.scrobble"},
		"api_key":   []string{apiKey},
		"sk":        []string{sessionKey},
		"artist":    []string{track.Artist},
		"track":     []string{track.Title},
		"album":     []string{track.Album},
		"timestamp": []string{fmt.Sprintf("%d", listenedAt.Unix())},
		"duration":  []string{fmt.Sprintf("%d", track.DurationSeconds)},
	}
	return c.do(ctx, params, apiSecret)
}
