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

const (
	lrclibBaseURL       = "https://lrclib.net/api/get"
	lrclibSearchBaseURL = "https://lrclib.net/api/search"
)

type LRCLIBProvider struct {
	BaseURL       string
	SearchBaseURL string
}

func (p LRCLIBProvider) ID() string { return "lrclib" }

func (p LRCLIBProvider) Name() string { return "LRCLIB" }

func (p LRCLIBProvider) baseURL() string {
	if strings.TrimSpace(p.BaseURL) != "" {
		return strings.TrimRight(p.BaseURL, "/")
	}
	return lrclibBaseURL
}

func (p LRCLIBProvider) searchBaseURL() string {
	if strings.TrimSpace(p.SearchBaseURL) != "" {
		return strings.TrimRight(p.SearchBaseURL, "/")
	}
	base := p.baseURL()
	if base != lrclibBaseURL {
		if before, ok := strings.CutSuffix(base, "/get"); ok {
			return before + "/search"
		}
		return strings.TrimRight(base, "/") + "/search"
	}
	return lrclibSearchBaseURL
}

func (p LRCLIBProvider) Fetch(ctx context.Context, client *http.Client, in FetchInput) (*Document, error) {
	if strings.TrimSpace(in.Artist) == "" || strings.TrimSpace(in.Title) == "" {
		return nil, fmt.Errorf("lrclib: artist and title required")
	}

	doc, metaErr := p.fetchByMetadata(ctx, client, in)
	if metaErr == nil && doc != nil && doc.Synced {
		return doc, nil
	}
	// /api/get often returns a plain-only hit for popular tracks even when
	// synced copies exist in /api/search. Keep searching in that case.
	if metaErr != nil && !isNotFoundError(metaErr) {
		return nil, metaErr
	}

	searchDoc, searchErr := p.fetchBySearch(ctx, client, in)
	if searchErr == nil {
		if searchDoc.Synced || doc == nil {
			return searchDoc, nil
		}
	}
	if doc != nil {
		return doc, nil
	}
	if searchErr != nil {
		return nil, searchErr
	}
	if metaErr != nil {
		return nil, metaErr
	}
	return nil, fmt.Errorf("lrclib: lyrics not found")
}

func (p LRCLIBProvider) fetchByMetadata(ctx context.Context, client *http.Client, in FetchInput) (*Document, error) {
	query := url.Values{}
	query.Set("artist_name", in.Artist)
	query.Set("track_name", in.Title)
	if in.Album != "" {
		query.Set("album_name", in.Album)
	}
	if in.DurationSec > 0 {
		query.Set("duration", fmt.Sprintf("%d", in.DurationSec))
	}
	endpoint := p.baseURL() + "?" + query.Encode()
	payload, err := p.requestJSON(ctx, client, endpoint)
	if err != nil {
		return nil, err
	}
	return p.documentFromPayload(payload, in)
}

func (p LRCLIBProvider) fetchBySearch(ctx context.Context, client *http.Client, in FetchInput) (*Document, error) {
	query := url.Values{}
	query.Set("q", strings.TrimSpace(in.Artist+" "+in.Title))
	endpoint := p.searchBaseURL() + "?" + query.Encode()

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
		return nil, fmt.Errorf("lrclib: lyrics not found")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("lrclib: search status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var results []lrclibPayload
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("lrclib: decode search response: %w", err)
	}
	match := pickLRCLIBMatch(results, in)
	if match == nil {
		return nil, fmt.Errorf("lrclib: lyrics not found")
	}
	return p.documentFromPayload(match, in)
}

type lrclibPayload struct {
	PlainLyrics  string  `json:"plainLyrics"`
	SyncedLyrics string  `json:"syncedLyrics"`
	ArtistName   string  `json:"artistName"`
	TrackName    string  `json:"trackName"`
	Instrumental bool    `json:"instrumental"`
	Duration     float64 `json:"duration"`
}

func (p LRCLIBProvider) requestJSON(ctx context.Context, client *http.Client, endpoint string) (*lrclibPayload, error) {
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
		return nil, fmt.Errorf("lrclib: lyrics not found")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("lrclib: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload lrclibPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("lrclib: decode response: %w", err)
	}
	return &payload, nil
}

func (p LRCLIBProvider) documentFromPayload(payload *lrclibPayload, in FetchInput) (*Document, error) {
	if payload == nil {
		return nil, fmt.Errorf("lrclib: empty response")
	}
	if payload.Instrumental {
		return nil, fmt.Errorf("lrclib: instrumental track")
	}

	raw := strings.TrimSpace(payload.SyncedLyrics)
	if raw == "" {
		raw = strings.TrimSpace(payload.PlainLyrics)
	}
	if raw == "" {
		return nil, fmt.Errorf("lrclib: empty lyrics")
	}

	doc := ParseText(raw, in)
	doc.Source = p.ID()
	if payload.ArtistName != "" {
		doc.Artist = payload.ArtistName
	}
	if payload.TrackName != "" {
		doc.Title = payload.TrackName
	}
	return NormalizeDocument(doc)
}

func pickLRCLIBMatch(results []lrclibPayload, in FetchInput) *lrclibPayload {
	wantArtist := normalizeMatchText(in.Artist)
	wantTitle := normalizeMatchText(in.Title)
	var best *lrclibPayload
	bestScore := -1

	for i := range results {
		candidate := &results[i]
		if candidate.Instrumental {
			continue
		}
		if strings.TrimSpace(candidate.PlainLyrics) == "" && strings.TrimSpace(candidate.SyncedLyrics) == "" {
			continue
		}
		score := 0
		if normalizeMatchText(candidate.ArtistName) == wantArtist {
			score += 3
		} else if strings.Contains(normalizeMatchText(candidate.ArtistName), wantArtist) || strings.Contains(wantArtist, normalizeMatchText(candidate.ArtistName)) {
			score += 1
		}
		if normalizeMatchText(candidate.TrackName) == wantTitle {
			score += 3
		} else if strings.Contains(normalizeMatchText(candidate.TrackName), wantTitle) || strings.Contains(wantTitle, normalizeMatchText(candidate.TrackName)) {
			score += 1
		}
		if strings.TrimSpace(candidate.SyncedLyrics) != "" {
			score += 1
		}
		if score > bestScore {
			best = candidate
			bestScore = score
		}
	}
	if bestScore < 2 {
		return nil
	}
	return best
}

func normalizeMatchText(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isNotFoundError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "not found")
}
