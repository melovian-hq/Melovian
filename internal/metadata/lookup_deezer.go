// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var deezerSearchURL = "https://api.deezer.com/search"

// SetDeezerSearchURL overrides the Deezer lookup endpoint. Tests only.
func SetDeezerSearchURL(url string) {
	deezerSearchURL = url
}

type deezerResult struct {
	Data []struct {
		ID     int64  `json:"id"`
		Title  string `json:"title"`
		Rank   int    `json:"rank"`
		Artist struct {
			Name string `json:"name"`
		} `json:"artist"`
		Album struct {
			Title   string `json:"title"`
			CoverXL string `json:"cover_xl"`
			Cover   string `json:"cover"`
		} `json:"album"`
	} `json:"data"`
}

// LookupDeezer searches the Deezer catalog for metadata matches. Deezer does
// not return track numbers, years, or genres in search results, so matches
// carry title, artist, album, and artwork.
func LookupDeezer(ctx context.Context, query string, limit int) ([]LookupMatch, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 25 {
		limit = 25
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("limit", strconv.Itoa(limit))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, deezerSearchURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: lookupTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("deezer lookup: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("deezer lookup: status %d", resp.StatusCode)
	}

	var payload deezerResult
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("deezer decode: %w", err)
	}

	matches := make([]LookupMatch, 0, len(payload.Data))
	for _, item := range payload.Data {
		artwork := item.Album.CoverXL
		if artwork == "" {
			artwork = item.Album.Cover
		}
		artist := strings.TrimSpace(item.Artist.Name)
		matches = append(matches, LookupMatch{
			ID:          fmt.Sprintf("deezer:%d", item.ID),
			Source:      "deezer",
			Title:       strings.TrimSpace(item.Title),
			Artist:      artist,
			Album:       strings.TrimSpace(item.Album.Title),
			AlbumArtist: artist,
			ArtworkURL:  artwork,
		})
	}
	return matches, nil
}
