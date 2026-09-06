// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package video

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func YouTubeEmbedURL(videoID string) (string, error) {
	id := strings.TrimSpace(videoID)
	if id == "" {
		return "", fmt.Errorf("video id is required")
	}
	return "https://www.youtube.com/embed/" + url.PathEscape(id), nil
}

// GetYouTubeVideo loads title and channel for a video id via the Data API.
func (c *Client) GetYouTubeVideo(ctx context.Context, apiKey, videoID string) (SearchHit, error) {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return SearchHit{}, fmt.Errorf("youtube api key is required")
	}
	id := strings.TrimSpace(videoID)
	if id == "" {
		return SearchHit{}, fmt.Errorf("video id is required")
	}

	endpoint, err := url.Parse("https://www.googleapis.com/youtube/v3/videos")
	if err != nil {
		return SearchHit{}, err
	}
	values := endpoint.Query()
	values.Set("part", "snippet")
	values.Set("id", id)
	values.Set("key", key)
	endpoint.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return SearchHit{}, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SearchHit{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return SearchHit{}, fmt.Errorf("youtube video lookup failed: %s", msg)
	}

	limited := io.LimitReader(resp.Body, maxResponseBytes)
	var payload struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title        string `json:"title"`
				ChannelTitle string `json:"channelTitle"`
				Thumbnails   map[string]struct {
					URL string `json:"url"`
				} `json:"thumbnails"`
			} `json:"snippet"`
		} `json:"items"`
	}
	if err := json.NewDecoder(limited).Decode(&payload); err != nil {
		return SearchHit{}, fmt.Errorf("decode youtube video: %w", err)
	}
	if len(payload.Items) == 0 {
		return SearchHit{}, fmt.Errorf("youtube video not found")
	}
	item := payload.Items[0]
	title := strings.TrimSpace(item.Snippet.Title)
	if title == "" {
		return SearchHit{}, fmt.Errorf("youtube video has no title")
	}
	hit := SearchHit{
		ID:     strings.TrimSpace(item.ID),
		Title:  title,
		Author: strings.TrimSpace(item.Snippet.ChannelTitle),
	}
	if hit.ID == "" {
		hit.ID = id
	}
	for _, key := range []string{"medium", "high", "default"} {
		if thumb, ok := item.Snippet.Thumbnails[key]; ok && strings.TrimSpace(thumb.URL) != "" {
			hit.Thumbnail = thumb.URL
			break
		}
	}
	return hit, nil
}

func (c *Client) SearchYouTube(ctx context.Context, apiKey, query string) ([]SearchHit, error) {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return nil, fmt.Errorf("youtube api key is required")
	}
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, fmt.Errorf("search query is required")
	}

	endpoint, err := url.Parse("https://www.googleapis.com/youtube/v3/search")
	if err != nil {
		return nil, err
	}
	values := endpoint.Query()
	values.Set("part", "snippet")
	values.Set("type", "video")
	values.Set("maxResults", fmt.Sprintf("%d", maxSearchResults))
	values.Set("q", q)
	values.Set("key", key)
	endpoint.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("youtube search failed: %s", msg)
	}

	limited := io.LimitReader(resp.Body, maxResponseBytes)
	var payload struct {
		Items []struct {
			ID struct {
				VideoID string `json:"videoId"`
			} `json:"id"`
			Snippet struct {
				Title        string `json:"title"`
				ChannelTitle string `json:"channelTitle"`
				Thumbnails   map[string]struct {
					URL string `json:"url"`
				} `json:"thumbnails"`
			} `json:"snippet"`
		} `json:"items"`
	}
	if err := json.NewDecoder(limited).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode youtube search: %w", err)
	}

	hits := make([]SearchHit, 0, len(payload.Items))
	for _, item := range payload.Items {
		id := strings.TrimSpace(item.ID.VideoID)
		title := strings.TrimSpace(item.Snippet.Title)
		if id == "" || title == "" {
			continue
		}
		hit := SearchHit{
			ID:     id,
			Title:  title,
			Author: strings.TrimSpace(item.Snippet.ChannelTitle),
		}
		for _, key := range []string{"medium", "high", "default"} {
			if thumb, ok := item.Snippet.Thumbnails[key]; ok && strings.TrimSpace(thumb.URL) != "" {
				hit.Thumbnail = thumb.URL
				break
			}
		}
		hits = append(hits, hit)
		if len(hits) >= maxSearchResults {
			break
		}
	}
	return hits, nil
}
