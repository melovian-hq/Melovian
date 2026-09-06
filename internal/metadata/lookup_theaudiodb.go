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

// theAudioDBKey is the free public v1 API key TheAudioDB publishes for
// evaluation and personal projects.
var theAudioDBBaseURL = "https://theaudiodb.com/api/v1/json/2"

// SetTheAudioDBBaseURL overrides the TheAudioDB API base URL. Tests only.
func SetTheAudioDBBaseURL(url string) {
	theAudioDBBaseURL = strings.TrimRight(url, "/")
}

type theAudioDBResult struct {
	Track []struct {
		IDTrack     string `json:"idTrack"`
		Title       string `json:"strTrack"`
		Artist      string `json:"strArtist"`
		Album       string `json:"strAlbum"`
		Genre       string `json:"strGenre"`
		TrackNumber string `json:"intTrackNumber"`
		Year        string `json:"intYearReleased"`
		Thumb       string `json:"strTrackThumb"`
	} `json:"track"`
}

// LookupTheAudioDB searches TheAudioDB for a track by artist and title. It
// needs both fields; a freeform query without an artist returns no matches.
func LookupTheAudioDB(ctx context.Context, q LookupQuery, limit int) ([]LookupMatch, error) {
	artist := strings.TrimSpace(q.Artist)
	title := strings.TrimSpace(q.Title)
	if artist == "" || title == "" {
		if splitArtist, splitTitle, ok := splitArtistTitle(q.Query); ok {
			artist = splitArtist
			title = splitTitle
		}
	}
	if artist == "" || title == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	params := url.Values{}
	params.Set("s", artist)
	params.Set("t", title)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, theAudioDBBaseURL+"/searchtrack.php?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: lookupTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("theaudiodb lookup: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("theaudiodb lookup: status %d", resp.StatusCode)
	}

	var payload theAudioDBResult
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("theaudiodb decode: %w", err)
	}

	matches := make([]LookupMatch, 0, len(payload.Track))
	for i, item := range payload.Track {
		if limit > 0 && i >= limit {
			break
		}
		trackNum, _ := strconv.Atoi(strings.TrimSpace(item.TrackNumber))
		year, _ := strconv.Atoi(strings.TrimSpace(item.Year))
		matches = append(matches, LookupMatch{
			ID:          fmt.Sprintf("theaudiodb:%s", item.IDTrack),
			Source:      "theaudiodb",
			Title:       strings.TrimSpace(item.Title),
			Artist:      strings.TrimSpace(item.Artist),
			Album:       strings.TrimSpace(item.Album),
			AlbumArtist: strings.TrimSpace(item.Artist),
			TrackNum:    trackNum,
			Year:        year,
			Genre:       strings.TrimSpace(item.Genre),
			ArtworkURL:  strings.TrimSpace(item.Thumb),
		})
	}
	return matches, nil
}
