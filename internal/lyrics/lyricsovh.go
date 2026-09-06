// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lyrics

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const lyricsOvhBaseURL = "https://api.lyrics.ovh/v1"

type LyricsOvhProvider struct {
	BaseURL string
}

func (p LyricsOvhProvider) ID() string { return "lyrics-ovh" }

func (p LyricsOvhProvider) Name() string { return "Lyrics.ovh" }

func (p LyricsOvhProvider) baseURL() string {
	if strings.TrimSpace(p.BaseURL) != "" {
		return strings.TrimRight(p.BaseURL, "/")
	}
	return lyricsOvhBaseURL
}

func (p LyricsOvhProvider) Fetch(ctx context.Context, client *http.Client, in FetchInput) (*Document, error) {
	if strings.TrimSpace(in.Artist) == "" || strings.TrimSpace(in.Title) == "" {
		return nil, fmt.Errorf("lyrics-ovh: artist and title required")
	}

	endpoint := fmt.Sprintf(
		"%s/%s/%s",
		p.baseURL(),
		url.PathEscape(strings.TrimSpace(in.Artist)),
		url.PathEscape(strings.TrimSpace(in.Title)),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Melovian/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("lyrics-ovh: lyrics not found")
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("lyrics-ovh: upstream unavailable (%d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("lyrics-ovh: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Lyrics string `json:"lyrics"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("lyrics-ovh: decode response: %w", err)
	}
	raw := strings.TrimSpace(payload.Lyrics)
	if raw == "" {
		return nil, fmt.Errorf("lyrics-ovh: empty lyrics")
	}

	doc := ParseText(raw, in)
	doc.Source = p.ID()
	return NormalizeDocument(doc)
}
