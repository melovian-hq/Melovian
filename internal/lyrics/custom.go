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

func (p CustomProvider) Fetch(ctx context.Context, client *http.Client, in FetchInput) (*Document, error) {
	template := strings.TrimSpace(p.URL)
	if template == "" {
		return nil, fmt.Errorf("custom provider %q: url required", p.ID())
	}

	endpoint, err := expandURLTemplate(template, in)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Melovian/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("custom provider %q: lyrics not found", p.ID())
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("custom provider %q: upstream unavailable (%d)", p.ID(), resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("custom provider %q: status %d: %s", p.ID(), resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	raw := extractLyricsBody(body)
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("custom provider %q: empty lyrics", p.ID())
	}

	doc := ParseText(raw, in)
	doc.Source = p.ID()
	return NormalizeDocument(doc)
}

func expandURLTemplate(template string, in FetchInput) (string, error) {
	if strings.Contains(template, "{") {
		replaced := template
		replaced = strings.ReplaceAll(replaced, "{artist}", url.QueryEscape(in.Artist))
		replaced = strings.ReplaceAll(replaced, "{title}", url.QueryEscape(in.Title))
		replaced = strings.ReplaceAll(replaced, "{album}", url.QueryEscape(in.Album))
		replaced = strings.ReplaceAll(replaced, "{trackId}", url.QueryEscape(in.TrackID))
		replaced = strings.ReplaceAll(replaced, "{duration}", url.QueryEscape(fmt.Sprintf("%d", in.DurationSec)))
		return replaced, nil
	}
	parsed, err := url.Parse(template)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	if query.Get("artist") == "" {
		query.Set("artist", in.Artist)
	}
	if query.Get("title") == "" {
		query.Set("title", in.Title)
	}
	if query.Get("album") == "" && in.Album != "" {
		query.Set("album", in.Album)
	}
	if query.Get("trackId") == "" && in.TrackID != "" {
		query.Set("trackId", in.TrackID)
	}
	if query.Get("duration") == "" && in.DurationSec > 0 {
		query.Set("duration", fmt.Sprintf("%d", in.DurationSec))
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func extractLyricsBody(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return ""
	}
	if trimmed[0] != '{' {
		return trimmed
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return trimmed
	}
	for _, key := range []string{"syncedLyrics", "plainLyrics", "lyrics", "value", "text"} {
		if raw, ok := payload[key].(string); ok && strings.TrimSpace(raw) != "" {
			return raw
		}
	}
	return trimmed
}
