// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"net/http"
	"testing"

	"melovian/internal/lyrics"
)

type stubLyricsProvider struct {
	id string
}

func (p stubLyricsProvider) ID() string   { return p.id }
func (p stubLyricsProvider) Name() string { return p.id }
func (p stubLyricsProvider) Fetch(
	_ context.Context,
	_ *http.Client,
	_ lyrics.FetchInput,
) (*lyrics.Document, error) {
	return &lyrics.Document{
		Source: p.id,
		Lines:  []lyrics.Line{{Text: p.id}},
	}, nil
}

func TestPreferOnlineLyricsProvidersOrdersSubsonicLast(t *testing.T) {
	providers := []lyrics.Provider{
		stubLyricsProvider{id: "subsonic"},
		stubLyricsProvider{id: "lrclib"},
		stubLyricsProvider{id: "lyrics-ovh"},
	}
	ordered := preferOnlineLyricsProviders(providers)
	if len(ordered) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(ordered))
	}
	if ordered[0].ID() == "subsonic" {
		t.Fatal("force refresh must not try subsonic first")
	}
	if ordered[len(ordered)-1].ID() != "subsonic" {
		t.Fatal("subsonic must remain last fallback")
	}
}

func TestExcludeProvidersDropsMatchingID(t *testing.T) {
	providers := []lyrics.Provider{
		stubLyricsProvider{id: "subsonic"},
		stubLyricsProvider{id: "lrclib"},
	}
	online := excludeProviders(providers, "subsonic")
	if len(online) != 1 || online[0].ID() != "lrclib" {
		t.Fatalf("unexpected providers: %+v", online)
	}
}

type stubDocLyricsProvider struct {
	id  string
	doc *lyrics.Document
}

func (p stubDocLyricsProvider) ID() string   { return p.id }
func (p stubDocLyricsProvider) Name() string { return p.id }
func (p stubDocLyricsProvider) Fetch(
	_ context.Context,
	_ *http.Client,
	_ lyrics.FetchInput,
) (*lyrics.Document, error) {
	return p.doc, nil
}

func TestForceRefreshOrderLetsSyncedOnlineBeatPlainSubsonic(t *testing.T) {
	start := 1000
	providers := preferOnlineLyricsProviders([]lyrics.Provider{
		stubDocLyricsProvider{
			id: "subsonic",
			doc: &lyrics.Document{
				Source:   "subsonic",
				RawValue: "Plain from server",
				Lines:    []lyrics.Line{{Text: "Plain from server"}},
			},
		},
		stubDocLyricsProvider{
			id: "lrclib",
			doc: &lyrics.Document{
				Source: "lrclib",
				Synced: true,
				Lines: []lyrics.Line{
					{Text: "Synced", StartMs: &start},
				},
			},
		},
	})
	fetcher := &lyrics.Fetcher{Client: http.DefaultClient}
	doc, err := fetcher.FetchFromProviders(context.Background(), providers, lyrics.FetchInput{
		Artist: "A",
		Title:  "B",
	})
	if err != nil {
		t.Fatalf("FetchFromProviders: %v", err)
	}
	if !doc.Synced || doc.Source != "lrclib" {
		t.Fatalf("force refresh must prefer synced online lyrics, got source=%q synced=%v", doc.Source, doc.Synced)
	}
}

func TestParseDurationSec(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{"", 0},
		{"211", 211},
		{"211.0", 211},
		{"211.4", 211},
		{"211.6", 212},
		{"-3", 0},
		{"nope", 0},
	}
	for _, tc := range cases {
		if got := parseDurationSec(tc.raw); got != tc.want {
			t.Fatalf("parseDurationSec(%q)=%d want %d", tc.raw, got, tc.want)
		}
	}
}
