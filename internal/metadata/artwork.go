// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"melovian/internal/httputil"
)

// LookupAlbumArtworkURL finds an iTunes artwork URL for an album.
func LookupAlbumArtworkURL(ctx context.Context, artist, album string) (string, error) {
	artist = strings.TrimSpace(artist)
	album = strings.TrimSpace(album)
	if album == "" {
		return "", nil
	}
	term := strings.TrimSpace(artist + " " + album)
	results, err := searchITunes(ctx, term, "album", 12)
	if err != nil {
		return "", err
	}
	for _, item := range results {
		if item.ArtworkURL == "" {
			continue
		}
		if artist != "" && !namesLooselyMatch(artist, item.Artist) {
			continue
		}
		if !namesLooselyMatch(album, item.Album) {
			continue
		}
		return upscaleArtworkURL(item.ArtworkURL), nil
	}
	// Fall back to first album result with art when names are close enough via term search.
	for _, item := range results {
		if item.ArtworkURL == "" {
			continue
		}
		if artist == "" || namesLooselyMatch(artist, item.Artist) {
			return upscaleArtworkURL(item.ArtworkURL), nil
		}
	}
	return "", nil
}

// LookupSongArtworkURL finds artwork via an iTunes song search.
func LookupSongArtworkURL(ctx context.Context, artist, title, album string) (string, error) {
	artist = strings.TrimSpace(artist)
	title = strings.TrimSpace(title)
	album = strings.TrimSpace(album)
	if title == "" {
		return "", nil
	}
	term := strings.TrimSpace(strings.Join([]string{artist, title}, " "))
	results, err := searchITunes(ctx, term, "song", 12)
	if err != nil {
		return "", err
	}
	for _, item := range results {
		if item.ArtworkURL == "" {
			continue
		}
		if artist != "" && !namesLooselyMatch(artist, item.Artist) {
			continue
		}
		if !namesLooselyMatch(title, item.Title) {
			continue
		}
		if album != "" && item.Album != "" && !namesLooselyMatch(album, item.Album) {
			continue
		}
		return upscaleArtworkURL(item.ArtworkURL), nil
	}
	for _, item := range results {
		if item.ArtworkURL == "" {
			continue
		}
		if artist == "" || namesLooselyMatch(artist, item.Artist) {
			return upscaleArtworkURL(item.ArtworkURL), nil
		}
	}
	return "", nil
}

// LookupArtistArtworkURL finds an iTunes artwork URL for an artist (via album art).
func LookupArtistArtworkURL(ctx context.Context, artist string) (string, error) {
	artist = strings.TrimSpace(artist)
	if artist == "" {
		return "", nil
	}
	results, err := searchITunes(ctx, artist, "album", 12)
	if err != nil {
		return "", err
	}
	for _, item := range results {
		if item.ArtworkURL == "" {
			continue
		}
		if namesLooselyMatch(artist, item.Artist) {
			return upscaleArtworkURL(item.ArtworkURL), nil
		}
	}
	return "", nil
}

// FetchArtwork downloads an artwork URL and returns body plus content type.
// The URL comes from third party metadata APIs, so the scheme is restricted
// and redirects are bounded to keep a hostile response from aiming the fetch
// at internal or non HTTP targets.
func FetchArtwork(ctx context.Context, artworkURL string) ([]byte, string, error) {
	artworkURL = strings.TrimSpace(artworkURL)
	if artworkURL == "" {
		return nil, "", fmt.Errorf("empty artwork url")
	}
	if _, err := httputil.ParseHTTPURL(artworkURL); err != nil {
		return nil, "", fmt.Errorf("invalid artwork url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, artworkURL, nil)
	if err != nil {
		return nil, "", err
	}
	client := &http.Client{
		Timeout: lookupTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			if req.URL.Scheme != "https" && req.URL.Scheme != "http" {
				return fmt.Errorf("redirect to disallowed scheme")
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", httputil.SanitizeErrorURL(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("artwork fetch status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, "", err
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/jpeg"
	}
	return body, ct, nil
}

func searchITunes(ctx context.Context, term, entity string, limit int) ([]LookupMatch, error) {
	term = strings.TrimSpace(term)
	if term == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 25 {
		limit = 25
	}
	if entity == "" {
		entity = "song"
	}

	params := url.Values{}
	params.Set("term", term)
	params.Set("media", "music")
	params.Set("entity", entity)
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
		id := item.TrackID
		title := strings.TrimSpace(item.TrackName)
		if id == 0 {
			// Album results use collectionId in the full API. TrackID may be 0.
			id = int64(len(matches) + 1)
		}
		matches = append(matches, LookupMatch{
			ID:          fmt.Sprintf("itunes:%d", id),
			Source:      "itunes",
			Title:       title,
			Artist:      strings.TrimSpace(item.ArtistName),
			Album:       strings.TrimSpace(item.CollectionName),
			AlbumArtist: strings.TrimSpace(item.ArtistName),
			TrackNum:    item.TrackNumber,
			Year:        parseReleaseYear(item.ReleaseDate),
			Genre:       strings.TrimSpace(item.PrimaryGenre),
			ArtworkURL:  strings.TrimSpace(item.ArtworkURL),
		})
	}
	return matches, nil
}

func upscaleArtworkURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	return strings.Replace(raw, "100x100bb", "600x600bb", 1)
}

func namesLooselyMatch(expected, candidate string) bool {
	left := normalizeName(expected)
	right := normalizeName(candidate)
	if left == "" || right == "" {
		return false
	}
	if left == right {
		return true
	}
	if strings.Contains(right, left) || strings.Contains(left, right) {
		shorter := left
		longer := right
		if len(left) > len(right) {
			shorter, longer = right, left
		}
		return float64(len(shorter))/float64(len(longer)) >= 0.7
	}
	return false
}

func normalizeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "the ")
	// Drop edition suffixes so "24K Magic (Deluxe)" matches "24K Magic".
	value = strings.Split(value, "(")[0]
	value = strings.Split(value, "[")[0]
	value = strings.TrimSpace(value)
	var b strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' {
			b.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
