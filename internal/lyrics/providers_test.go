// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lyrics

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLRCLIBProviderUpgradesPlainGetToSyncedSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/get") || r.URL.Query().Get("artist_name") != "":
			if r.URL.Query().Get("artist_name") == "Drake" && r.URL.Query().Get("track_name") == "Nice For What" {
				_, _ = w.Write([]byte(`{"id":23074,"artistName":"Drake","trackName":"Nice For What","plainLyrics":"Plain only","syncedLyrics":""}`))
				return
			}
			http.NotFound(w, r)
		case strings.Contains(r.URL.Path, "/search") || r.URL.Query().Get("q") != "":
			_, _ = w.Write([]byte(`[
				{"artistName":"Drake","trackName":"Nice For What","plainLyrics":"Plain only","syncedLyrics":""},
				{"artistName":"Drake","trackName":"Nice For What","plainLyrics":"also","syncedLyrics":"[00:01.00]Synced line"}
			]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := LRCLIBProvider{
		BaseURL:       server.URL + "/get",
		SearchBaseURL: server.URL + "/search",
	}
	doc, err := provider.Fetch(context.Background(), server.Client(), FetchInput{
		Artist: "Drake",
		Title:  "Nice For What",
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if !doc.Synced {
		t.Fatalf("expected synced lyrics from search after plain /get, got plain")
	}
	if len(doc.Lines) == 0 || doc.Lines[0].Text != "Synced line" {
		t.Fatalf("unexpected lines: %+v", doc.Lines)
	}
}

func TestLRCLIBProviderKeepsPlainWhenSearchHasNoSynced(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("artist_name") != "" {
			_, _ = w.Write([]byte(`{"artistName":"Artist","trackName":"Song","plainLyrics":"Only plain","syncedLyrics":""}`))
			return
		}
		_, _ = w.Write([]byte(`[{"artistName":"Artist","trackName":"Song","plainLyrics":"Search plain","syncedLyrics":""}]`))
	}))
	defer server.Close()

	provider := LRCLIBProvider{
		BaseURL:       server.URL,
		SearchBaseURL: server.URL,
	}
	doc, err := provider.Fetch(context.Background(), server.Client(), FetchInput{
		Artist: "Artist",
		Title:  "Song",
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if doc.Synced {
		t.Fatal("expected plain lyrics when search has no synced match")
	}
	if len(doc.Lines) == 0 || doc.Lines[0].Text != "Only plain" {
		t.Fatalf("expected metadata plain lyrics, got %+v", doc.Lines)
	}
}

func TestLRCLIBProviderFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("artist_name") != "Artist" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"artistName":"Artist","trackName":"Song","syncedLyrics":"[00:01.00]Hello\n[00:03.00]World"}`))
	}))
	defer server.Close()
	provider := LRCLIBProvider{BaseURL: server.URL}
	doc, err := provider.Fetch(context.Background(), server.Client(), FetchInput{
		Artist: "Artist",
		Title:  "Song",
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if !doc.Synced || len(doc.Lines) != 2 {
		t.Fatalf("unexpected doc: synced=%v lines=%d", doc.Synced, len(doc.Lines))
	}
}

func TestLRCLIBProviderFetchRussianMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("artist_name") != "МакSим" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("track_name") != "Трудный возраст" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"artistName":"МакSим","trackName":"Трудный возраст","plainLyrics":"Первая строка\nВторая строка"}`))
	}))
	defer server.Close()
	provider := LRCLIBProvider{BaseURL: server.URL}
	doc, err := provider.Fetch(context.Background(), server.Client(), FetchInput{
		Artist: "МакSим",
		Title:  "Трудный возраст",
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(doc.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(doc.Lines))
	}
}

func TestLRCLIBProviderNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	}))
	defer server.Close()
	provider := LRCLIBProvider{BaseURL: server.URL}
	_, err := provider.Fetch(context.Background(), server.Client(), FetchInput{
		Artist: "Missing",
		Title:  "Track",
	})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestLyricsOvhProviderFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/Artist/Title") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"lyrics":"Line one\nLine two"}`))
	}))
	defer server.Close()
	provider := LyricsOvhProvider{BaseURL: server.URL}
	doc, err := provider.Fetch(context.Background(), server.Client(), FetchInput{
		Artist: "Artist",
		Title:  "Title",
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(doc.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(doc.Lines))
	}
}

func TestLyricsOvhProviderDown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	provider := LyricsOvhProvider{BaseURL: server.URL}
	_, err := provider.Fetch(context.Background(), server.Client(), FetchInput{
		Artist: "Artist",
		Title:  "Title",
	})
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("expected provider down error, got %v", err)
	}
}

func TestCustomProviderFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "track=Hello") {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"lyrics":"Custom line"}`))
	}))
	defer server.Close()
	provider := CustomProvider{
		IDValue:   "custom-test",
		NameValue: "Custom",
		URL:       server.URL + "?artist={artist}&track={title}",
	}
	doc, err := provider.Fetch(context.Background(), server.Client(), FetchInput{
		Artist: "Band",
		Title:  "Hello",
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if doc.Lines[0].Text != "Custom line" {
		t.Fatalf("unexpected line: %q", doc.Lines[0].Text)
	}
}

func TestFetcherUsesNextProviderWhenFirstFails(t *testing.T) {
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer first.Close()
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"lyrics":"Recovered"}`))
	}))
	defer second.Close()
	fetcher := &Fetcher{Client: second.Client()}
	doc, err := fetcher.FetchFromProviders(context.Background(), []Provider{
		LyricsOvhProvider{BaseURL: first.URL},
		LyricsOvhProvider{BaseURL: second.URL},
	}, FetchInput{Artist: "A", Title: "B"})
	if err != nil {
		t.Fatalf("FetchFromProviders: %v", err)
	}
	if doc.Lines[0].Text != "Recovered" {
		t.Fatalf("unexpected lyrics: %q", doc.Lines[0].Text)
	}
}

type stubDocProvider struct {
	id  string
	doc *Document
	err error
}

func (p stubDocProvider) ID() string   { return p.id }
func (p stubDocProvider) Name() string { return p.id }
func (p stubDocProvider) Fetch(
	_ context.Context,
	_ *http.Client,
	_ FetchInput,
) (*Document, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.doc, nil
}

func TestFetcherPrefersSyncedOverEarlierPlain(t *testing.T) {
	fetcher := &Fetcher{Client: http.DefaultClient}
	doc, err := fetcher.FetchFromProviders(context.Background(), []Provider{
		stubDocProvider{
			id: "plain",
			doc: &Document{
				Source:   "plain",
				RawValue: "Hello\nWorld",
				Lines:    []Line{{Text: "Hello"}, {Text: "World"}},
			},
		},
		stubDocProvider{
			id: "synced",
			doc: &Document{
				Source:   "synced",
				Synced:   true,
				RawValue: "[00:01.00]Hello\n[00:03.00]World",
				Lines: []Line{
					{Text: "Hello", StartMs: new(1000)},
					{Text: "World", StartMs: new(3000)},
				},
			},
		},
	}, FetchInput{Artist: "A", Title: "B"})
	if err != nil {
		t.Fatalf("FetchFromProviders: %v", err)
	}
	if !doc.Synced || doc.Source != "synced" {
		t.Fatalf("expected synced provider, got source=%q synced=%v", doc.Source, doc.Synced)
	}
}

func TestFetcherKeepsPlainWhenNoSyncedAvailable(t *testing.T) {
	fetcher := &Fetcher{Client: http.DefaultClient}
	doc, err := fetcher.FetchFromProviders(context.Background(), []Provider{
		stubDocProvider{
			id: "subsonic",
			doc: &Document{
				Source:   "subsonic",
				RawValue: "Only plain",
				Lines:    []Line{{Text: "Only plain"}},
			},
		},
		stubDocProvider{
			id:  "lrclib",
			err: fmt.Errorf("lrclib: lyrics not found"),
		},
	}, FetchInput{Artist: "A", Title: "B"})
	if err != nil {
		t.Fatalf("FetchFromProviders: %v", err)
	}
	if doc.Synced || doc.Source != "subsonic" {
		t.Fatalf("expected plain subsonic fallback, got source=%q synced=%v", doc.Source, doc.Synced)
	}
}

func TestFetcherStopsAfterFirstSyncedProvider(t *testing.T) {
	called := 0
	later := stubDocProvider{
		id: "later",
		doc: &Document{
			Source: "later",
			Lines:  []Line{{Text: "should not run"}},
		},
	}
	countingLater := countingProvider{stub: later, calls: &called}
	fetcher := &Fetcher{Client: http.DefaultClient}
	doc, err := fetcher.FetchFromProviders(context.Background(), []Provider{
		stubDocProvider{
			id: "synced",
			doc: &Document{
				Source: "synced",
				Synced: true,
				Lines: []Line{
					{Text: "Hello", StartMs: new(0)},
				},
			},
		},
		countingLater,
	}, FetchInput{Artist: "A", Title: "B"})
	if err != nil {
		t.Fatalf("FetchFromProviders: %v", err)
	}
	if doc.Source != "synced" {
		t.Fatalf("unexpected source %q", doc.Source)
	}
	if called != 0 {
		t.Fatalf("expected later provider skipped, calls=%d", called)
	}
}

type countingProvider struct {
	stub  stubDocProvider
	calls *int
}

func (p countingProvider) ID() string   { return p.stub.ID() }
func (p countingProvider) Name() string { return p.stub.Name() }
func (p countingProvider) Fetch(
	ctx context.Context,
	client *http.Client,
	in FetchInput,
) (*Document, error) {
	*p.calls++
	return p.stub.Fetch(ctx, client, in)
}

func TestPickLRCLIBMatchPrefersSyncedWhenScoresTie(t *testing.T) {
	match := pickLRCLIBMatch([]lrclibPayload{
		{
			ArtistName:  "Artist",
			TrackName:   "Song",
			PlainLyrics: "plain only",
		},
		{
			ArtistName:   "Artist",
			TrackName:    "Song",
			SyncedLyrics: "[00:01.00]synced",
			PlainLyrics:  "also plain",
		},
	}, FetchInput{Artist: "Artist", Title: "Song"})
	if match == nil || strings.TrimSpace(match.SyncedLyrics) == "" {
		t.Fatalf("expected synced match, got %+v", match)
	}
}
