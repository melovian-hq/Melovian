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
	"time"
)

const lookupTimeout = 12 * time.Second

var itunesSearchURL = "https://itunes.apple.com/search"

// SetITunesSearchURL overrides the iTunes lookup endpoint. Tests only.
func SetITunesSearchURL(url string) {
	itunesSearchURL = url
}

// LookupMatch is a metadata suggestion from an external catalog.
type LookupMatch struct {
	ID          string `json:"id"`
	Source      string `json:"source"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	AlbumArtist string `json:"albumArtist"`
	TrackNum    int    `json:"trackNum"`
	Year        int    `json:"year"`
	Genre       string `json:"genre"`
	ArtworkURL  string `json:"artworkUrl,omitempty"`
}

type itunesResult struct {
	Results []struct {
		TrackID        int64  `json:"trackId"`
		TrackName      string `json:"trackName"`
		ArtistName     string `json:"artistName"`
		CollectionName string `json:"collectionName"`
		TrackNumber    int    `json:"trackNumber"`
		PrimaryGenre   string `json:"primaryGenreName"`
		ReleaseDate    string `json:"releaseDate"`
		ArtworkURL     string `json:"artworkUrl100"`
	} `json:"results"`
}

// LookupITunes searches the iTunes catalog for metadata matches.
func LookupITunes(ctx context.Context, query string, limit int) ([]LookupMatch, error) {
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
	params.Set("term", query)
	params.Set("media", "music")
	params.Set("entity", "song")
	params.Set("limit", strconv.Itoa(limit))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, itunesSearchURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: lookupTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("itunes lookup: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("itunes lookup: status %d", resp.StatusCode)
	}

	var payload itunesResult
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("itunes decode: %w", err)
	}

	matches := make([]LookupMatch, 0, len(payload.Results))
	for _, item := range payload.Results {
		year := parseReleaseYear(item.ReleaseDate)
		artwork := strings.Replace(item.ArtworkURL, "100x100bb", "600x600bb", 1)
		matches = append(matches, LookupMatch{
			ID:          fmt.Sprintf("itunes:%d", item.TrackID),
			Source:      "itunes",
			Title:       strings.TrimSpace(item.TrackName),
			Artist:      strings.TrimSpace(item.ArtistName),
			Album:       strings.TrimSpace(item.CollectionName),
			AlbumArtist: strings.TrimSpace(item.ArtistName),
			TrackNum:    item.TrackNumber,
			Year:        year,
			Genre:       strings.TrimSpace(item.PrimaryGenre),
			ArtworkURL:  artwork,
		})
	}
	return matches, nil
}

func parseReleaseYear(value string) int {
	value = strings.TrimSpace(value)
	if len(value) < 4 {
		return 0
	}
	year, err := strconv.Atoi(value[:4])
	if err != nil {
		return 0
	}
	return year
}
