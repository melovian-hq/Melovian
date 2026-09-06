// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lyrics

import (
	"net/url"
	"strings"
	"testing"
)

func TestNormalizeDocumentRebuildsLinesFromRawValue(t *testing.T) {
	doc, err := NormalizeDocument(&Document{
		Source:   "lrclib",
		RawValue: "Line one\nLine two",
	})
	if err != nil {
		t.Fatalf("NormalizeDocument: %v", err)
	}
	if len(doc.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(doc.Lines))
	}
}

func TestNormalizeDocumentRejectsEmptyLyrics(t *testing.T) {
	_, err := NormalizeDocument(&Document{Source: "test"})
	if err == nil {
		t.Fatal("expected error for empty lyrics")
	}
}

func TestNormalizeDocumentReparsesEmbeddedLRC(t *testing.T) {
	doc, err := NormalizeDocument(&Document{
		Source:   "subsonic",
		Synced:   false,
		RawValue: "[00:01.00]First line\n[00:03.00]Second line",
		Lines: []Line{
			{Text: "[00:01.00]First line"},
			{Text: "[00:03.00]Second line"},
		},
	})
	if err != nil {
		t.Fatalf("NormalizeDocument: %v", err)
	}
	if !doc.Synced {
		t.Fatal("expected synced lyrics after LRC reparse")
	}
	if doc.Lines[0].StartMs == nil || *doc.Lines[0].StartMs != 1000 {
		t.Fatalf("unexpected first line: %+v", doc.Lines[0])
	}
	if doc.Lines[0].Text != "First line" {
		t.Fatalf("expected cleaned text, got %q", doc.Lines[0].Text)
	}

	again, err := NormalizeDocument(doc)
	if err != nil {
		t.Fatalf("second NormalizeDocument: %v", err)
	}
	if !again.Synced || again.Lines[0].StartMs == nil || *again.Lines[0].StartMs != 1000 {
		t.Fatalf("LRC reparse must stay idempotent: %+v", again.Lines[0])
	}
}

func TestNormalizeDocumentLeavesPlainTextUnsynced(t *testing.T) {
	doc, err := NormalizeDocument(&Document{
		Source:   "subsonic",
		Synced:   false,
		RawValue: "No timestamps here",
		Lines:    []Line{{Text: "No timestamps here"}},
	})
	if err != nil {
		t.Fatalf("NormalizeDocument: %v", err)
	}
	if doc.Synced {
		t.Fatal("plain text must stay unsynced")
	}
}

func TestLRCLIBEncodesCyrillicMetadata(t *testing.T) {
	in := FetchInput{
		Artist: "МакSим",
		Title:  "Трудный возраст",
	}
	query := url.Values{}
	query.Set("artist_name", in.Artist)
	query.Set("track_name", in.Title)
	encoded := query.Encode()
	if !strings.Contains(encoded, "artist_name=") {
		t.Fatalf("expected encoded artist query, got %q", encoded)
	}
	if !strings.Contains(encoded, "track_name=") {
		t.Fatalf("expected encoded title query, got %q", encoded)
	}
}

func TestPickLRCLIBMatchPrefersExactRussianMetadata(t *testing.T) {
	match := pickLRCLIBMatch([]lrclibPayload{
		{
			ArtistName:   "Other",
			TrackName:    "Song",
			PlainLyrics:  "nope",
			Instrumental: false,
		},
		{
			ArtistName:  "МакSим",
			TrackName:   "Трудный возраст",
			PlainLyrics: "lyrics",
		},
	}, FetchInput{
		Artist: "МакSим",
		Title:  "Трудный возраст",
	})
	if match == nil || match.TrackName != "Трудный возраст" {
		t.Fatalf("unexpected match: %+v", match)
	}
}
