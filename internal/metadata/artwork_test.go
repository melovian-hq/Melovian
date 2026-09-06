// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLookupAlbumArtworkURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"artistName":"Bruno Mars","collectionName":"24K Magic","artworkUrl100":"https://example.com/100x100bb.jpg"}]}`))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { SetITunesSearchURL("https://itunes.apple.com/search") })
	SetITunesSearchURL(srv.URL)

	url, err := LookupAlbumArtworkURL(context.Background(), "Bruno Mars", "24K Magic")
	if err != nil {
		t.Fatalf("LookupAlbumArtworkURL: %v", err)
	}
	if !strings.Contains(url, "600x600bb") {
		t.Fatalf("expected upscaled art url, got %q", url)
	}
}

func TestNamesLooselyMatch(t *testing.T) {
	if !namesLooselyMatch("Bruno Mars", "Bruno Mars") {
		t.Fatal("exact match")
	}
	if !namesLooselyMatch("24K Magic", "24K Magic (Deluxe)") {
		t.Fatal("album deluxe should match")
	}
	if namesLooselyMatch("Taylor Swift", "Bruno Mars") {
		t.Fatal("different artists should not match")
	}
}
