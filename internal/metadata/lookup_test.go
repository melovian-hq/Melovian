// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLookupITunes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("term") != "slowdive alison" {
			t.Fatalf("unexpected term: %q", r.URL.Query().Get("term"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"trackId":          42,
					"trackName":        "Alison",
					"artistName":       "Slowdive",
					"collectionName":   "Souvlaki",
					"trackNumber":      3,
					"primaryGenreName": "Shoegaze",
					"releaseDate":      "1993-05-17T07:00:00Z",
					"artworkUrl100":    "https://example.com/100x100bb.jpg",
				},
			},
		})
	}))
	defer server.Close()

	SetITunesSearchURL(server.URL)
	t.Cleanup(func() { SetITunesSearchURL("https://itunes.apple.com/search") })

	matches, err := LookupITunes(context.Background(), "slowdive alison", 5)
	if err != nil {
		t.Fatalf("LookupITunes: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	match := matches[0]
	if match.Title != "Alison" || match.Artist != "Slowdive" || match.Album != "Souvlaki" || match.Year != 1993 {
		t.Fatalf("unexpected match: %+v", match)
	}
	if match.ArtworkURL == "" {
		t.Fatal("expected artwork url")
	}
}
