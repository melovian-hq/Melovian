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

var musicBrainzSearchURL = "https://musicbrainz.org/ws/2/recording/"

// SetMusicBrainzSearchURL overrides the MusicBrainz lookup endpoint. Tests only.
func SetMusicBrainzSearchURL(url string) {
	musicBrainzSearchURL = url
}

type musicBrainzRecording struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Date    string `json:"first-release-date"`
	Score   int    `json:"score"`
	Acredit []struct {
		Name   string `json:"name"`
		Artist struct {
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"artist-credit"`
	Releases []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Date  string `json:"date"`
	} `json:"releases"`
	Tags []struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	} `json:"tags"`
}

type musicBrainzResult struct {
	Recordings []musicBrainzRecording `json:"recordings"`
}

func musicBrainzQuery(q LookupQuery) string {
	var parts []string
	if strings.TrimSpace(q.Title) != "" {
		parts = append(parts, fmt.Sprintf("recording:%q", q.Title))
	}
	if strings.TrimSpace(q.Artist) != "" {
		parts = append(parts, fmt.Sprintf("artist:%q", q.Artist))
	}
	if strings.TrimSpace(q.Album) != "" {
		parts = append(parts, fmt.Sprintf("release:%q", q.Album))
	}
	if len(parts) == 0 {
		term := q.term()
		if term == "" {
			return ""
		}
		return fmt.Sprintf("recording:%q OR artist:%q", term, term)
	}
	return strings.Join(parts, " AND ")
}

// LookupMusicBrainz searches the MusicBrainz recording catalog for matches.
func LookupMusicBrainz(ctx context.Context, q LookupQuery, limit int) ([]LookupMatch, error) {
	query := musicBrainzQuery(q)
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
	params.Set("query", query)
	params.Set("fmt", "json")
	params.Set("limit", strconv.Itoa(limit))
	params.Set("inc", "tags+releases")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, musicBrainzSearchURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	// MusicBrainz requires a descriptive User-Agent with contact info.
	req.Header.Set("User-Agent", "Melovian/1.0 (https://github.com/Quad4-Software/melovian)")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: lookupTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("musicbrainz lookup: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("musicbrainz lookup: status %d", resp.StatusCode)
	}

	var payload musicBrainzResult
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("musicbrainz decode: %w", err)
	}

	matches := make([]LookupMatch, 0, len(payload.Recordings))
	for _, rec := range payload.Recordings {
		match := LookupMatch{
			ID:     fmt.Sprintf("musicbrainz:%s", rec.ID),
			Source: "musicbrainz",
			Title:  strings.TrimSpace(rec.Title),
		}
		if len(rec.Acredit) > 0 {
			name := strings.TrimSpace(rec.Acredit[0].Artist.Name)
			if name == "" {
				name = strings.TrimSpace(rec.Acredit[0].Name)
			}
			match.Artist = name
			match.AlbumArtist = name
		}
		if len(rec.Releases) > 0 {
			match.Album = strings.TrimSpace(rec.Releases[0].Title)
			if rec.Releases[0].ID != "" {
				match.ArtworkURL = "https://coverartarchive.org/release/" + rec.Releases[0].ID + "/front-500"
			}
		}
		date := rec.Date
		if date == "" && len(rec.Releases) > 0 {
			date = rec.Releases[0].Date
		}
		match.Year = parseReleaseYear(date)
		if len(rec.Tags) > 0 {
			best := rec.Tags[0]
			for _, tag := range rec.Tags[1:] {
				if tag.Count > best.Count {
					best = tag
				}
			}
			match.Genre = strings.TrimSpace(best.Name)
		}
		matches = append(matches, match)
	}
	return matches, nil
}
