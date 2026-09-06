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
	"time"

	"melovian/internal/httputil"
)

const maxSearchResults = 20
const maxResponseBytes = 2 << 20

type SearchHit struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Author        string `json:"author,omitempty"`
	LengthSeconds int    `json:"lengthSeconds,omitempty"`
	Thumbnail     string `json:"thumbnail,omitempty"`
}

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: httputil.APITransport(),
		},
	}
}

func NormalizeInstanceURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("instance url is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid instance url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("instance url must be http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("instance url host is required")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func EmbedURL(instanceBase, videoID string) (string, error) {
	base, err := NormalizeInstanceURL(instanceBase)
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(videoID)
	if id == "" {
		return "", fmt.Errorf("video id is required")
	}
	return base + "/embed/" + url.PathEscape(id), nil
}

// GetInvidiousVideo loads title and author for a video id from an Invidious instance.
func (c *Client) GetInvidiousVideo(ctx context.Context, instanceBase, videoID string) (SearchHit, error) {
	base, err := NormalizeInstanceURL(instanceBase)
	if err != nil {
		return SearchHit{}, err
	}
	id := strings.TrimSpace(videoID)
	if id == "" {
		return SearchHit{}, fmt.Errorf("video id is required")
	}

	endpoint := base + "/api/v1/videos/" + url.PathEscape(id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
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
		return SearchHit{}, fmt.Errorf("invidious video lookup failed: %s", resp.Status)
	}

	limited := io.LimitReader(resp.Body, maxResponseBytes)
	var payload struct {
		VideoID       string `json:"videoId"`
		Title         string `json:"title"`
		Author        string `json:"author"`
		LengthSeconds int    `json:"lengthSeconds"`
		Thumbnails    []struct {
			URL string `json:"url"`
		} `json:"videoThumbnails"`
	}
	if err := json.NewDecoder(limited).Decode(&payload); err != nil {
		return SearchHit{}, fmt.Errorf("decode invidious video: %w", err)
	}

	resolvedID := strings.TrimSpace(payload.VideoID)
	if resolvedID == "" {
		resolvedID = id
	}
	title := strings.TrimSpace(payload.Title)
	if title == "" {
		return SearchHit{}, fmt.Errorf("invidious video has no title")
	}
	hit := SearchHit{
		ID:            resolvedID,
		Title:         title,
		Author:        strings.TrimSpace(payload.Author),
		LengthSeconds: payload.LengthSeconds,
	}
	for _, thumb := range payload.Thumbnails {
		if urlStr := strings.TrimSpace(thumb.URL); urlStr != "" {
			hit.Thumbnail = urlStr
			break
		}
	}
	return hit, nil
}

func (c *Client) SearchInvidious(ctx context.Context, instanceBase, query string) ([]SearchHit, error) {
	base, err := NormalizeInstanceURL(instanceBase)
	if err != nil {
		return nil, err
	}
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, fmt.Errorf("search query is required")
	}

	endpoint, err := url.Parse(base + "/api/v1/search")
	if err != nil {
		return nil, err
	}
	values := endpoint.Query()
	values.Set("q", q)
	values.Set("type", "video")
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
		return nil, fmt.Errorf("invidious search failed: %s", resp.Status)
	}

	limited := io.LimitReader(resp.Body, maxResponseBytes)
	var raw []map[string]any
	if err := json.NewDecoder(limited).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode invidious search: %w", err)
	}

	hits := make([]SearchHit, 0, len(raw))
	for _, item := range raw {
		if typ, _ := item["type"].(string); typ != "" && typ != "video" {
			continue
		}
		id, _ := item["videoId"].(string)
		title, _ := item["title"].(string)
		if strings.TrimSpace(id) == "" || strings.TrimSpace(title) == "" {
			continue
		}
		hit := SearchHit{
			ID:     id,
			Title:  title,
			Author: stringField(item, "author"),
		}
		if length, ok := item["lengthSeconds"].(float64); ok {
			hit.LengthSeconds = int(length)
		}
		hit.Thumbnail = firstThumbnail(item)
		hits = append(hits, hit)
		if len(hits) >= maxSearchResults {
			break
		}
	}
	return hits, nil
}

func stringField(item map[string]any, key string) string {
	value, _ := item[key].(string)
	return strings.TrimSpace(value)
}

func firstThumbnail(item map[string]any) string {
	thumbs, ok := item["videoThumbnails"].([]any)
	if !ok || len(thumbs) == 0 {
		return ""
	}
	for _, thumb := range thumbs {
		obj, ok := thumb.(map[string]any)
		if !ok {
			continue
		}
		if urlStr, _ := obj["url"].(string); strings.TrimSpace(urlStr) != "" {
			return urlStr
		}
	}
	return ""
}
