// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package rocksky

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"melovian/internal/brand"
)

const DefaultBaseURL = "https://audioscrobbler.rocksky.app"

// Track holds the metadata needed for a Rocksky scrobble.
type Track struct {
	ID              string
	Title           string
	Artist          string
	Album           string
	DurationSeconds int
}

// Client submits listens to Rocksky's ListenBrainz-compatible endpoint.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// NewClient returns a Rocksky client using the supplied HTTP client.
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
	return strings.TrimRight(base, "/")
}

func (c *Client) doJSON(ctx context.Context, method, path, token string, body, dest any) error {
	u := c.base() + path
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("rocksky %s %s: %d %s", method, path, resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	if dest != nil && resp.StatusCode != http.StatusNoContent {
		return json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(dest)
	}
	return nil
}

type submitListensRequest struct {
	ListenType string          `json:"listen_type"`
	Payload    []listenPayload `json:"payload"`
}

type listenPayload struct {
	ListenedAt    int64         `json:"listened_at,omitempty"`
	TrackMetadata trackMetadata `json:"track_metadata"`
}

type trackMetadata struct {
	ArtistName     string         `json:"artist_name"`
	TrackName      string         `json:"track_name"`
	ReleaseName    string         `json:"release_name,omitempty"`
	AdditionalInfo additionalInfo `json:"additional_info"`
}

type additionalInfo struct {
	DurationMs       int    `json:"duration_ms"`
	MediaPlayer      string `json:"media_player"`
	SubmissionClient string `json:"submission_client"`
}

func (c *Client) buildSubmission(track Track, listenType string, listenedAt int64) submitListensRequest {
	return submitListensRequest{
		ListenType: listenType,
		Payload: []listenPayload{
			{
				ListenedAt: listenedAt,
				TrackMetadata: trackMetadata{
					ArtistName:  track.Artist,
					TrackName:   track.Title,
					ReleaseName: track.Album,
					AdditionalInfo: additionalInfo{
						DurationMs:       track.DurationSeconds * 1000,
						MediaPlayer:      brand.Name,
						SubmissionClient: brand.Name,
					},
				},
			},
		},
	}
}

// SubmitNowPlaying reports the currently playing track.
func (c *Client) SubmitNowPlaying(ctx context.Context, token string, track Track) error {
	body := c.buildSubmission(track, "playing_now", 0)
	return c.doJSON(ctx, http.MethodPost, "/1/submit-listens", token, body, nil)
}

// SubmitScrobble reports a completed track listen.
func (c *Client) SubmitScrobble(ctx context.Context, token string, track Track, listenedAt time.Time) error {
	body := c.buildSubmission(track, "single", listenedAt.Unix())
	return c.doJSON(ctx, http.MethodPost, "/1/submit-listens", token, body, nil)
}

type validateTokenResponse struct {
	Valid    bool   `json:"valid"`
	UserName string `json:"user_name"`
	Message  string `json:"message"`
}

// ValidateToken checks whether a Rocksky API token is active.
func (c *Client) ValidateToken(ctx context.Context, token string) (string, error) {
	var resp validateTokenResponse
	if err := c.doJSON(ctx, http.MethodGet, "/1/validate-token", token, nil, &resp); err != nil {
		return "", err
	}
	if !resp.Valid {
		return "", fmt.Errorf("rocksky token invalid: %s", resp.Message)
	}
	return resp.UserName, nil
}
