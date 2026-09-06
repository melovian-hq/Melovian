// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonic

import (
	"strings"
	"testing"
)

func TestPickStructuredEntryPrefersSyncedOverUnsyncedMain(t *testing.T) {
	entries := []map[string]any{
		{
			"kind":   "main",
			"synced": false,
			"line":   []any{map[string]any{"value": "Plain"}},
		},
		{
			"kind":   "translation",
			"synced": true,
			"line": []any{
				map[string]any{"start": float64(1000), "value": "Timed"},
			},
		},
	}
	got := pickStructuredEntry(entries)
	if kind, _ := got["kind"].(string); kind != "translation" {
		t.Fatalf("expected synced translation, got kind=%v entry=%+v", kind, got)
	}
}

func TestPickStructuredEntryPrefersSyncedMain(t *testing.T) {
	entries := []map[string]any{
		{
			"kind":   "translation",
			"synced": true,
			"line": []any{
				map[string]any{"start": float64(500), "value": "Traduction"},
			},
		},
		{
			"kind":   "main",
			"synced": true,
			"line": []any{
				map[string]any{"start": float64(0), "value": "Main"},
			},
		},
	}
	got := pickStructuredEntry(entries)
	if kind, _ := got["kind"].(string); kind != "main" {
		t.Fatalf("expected synced main, got kind=%v", kind)
	}
}

func TestParseLyricsResponsePrefersSyncedStructuredEntry(t *testing.T) {
	body := []byte(`{
		"subsonic-response": {
			"status": "ok",
			"lyricsList": {
				"structuredLyrics": [
					{"kind":"main","synced":false,"line":[{"value":"Plain"}]},
					{"kind":"translation","synced":true,"line":[{"start":1500,"value":"Timed"}]}
				]
			}
		}
	}`)
	doc, err := parseLyricsResponse(body)
	if err != nil {
		t.Fatalf("parseLyricsResponse: %v", err)
	}
	if !doc.Synced {
		t.Fatal("expected synced lyrics from structured entry")
	}
	if len(doc.Lines) != 1 || doc.Lines[0].Value != "Timed" {
		t.Fatalf("unexpected lines: %+v", doc.Lines)
	}
	if doc.Lines[0].Start == nil || *doc.Lines[0].Start != 1500 {
		t.Fatalf("expected startMs=1500, got %+v", doc.Lines[0].Start)
	}
}

func TestParseLyricsResponsePlainValueKeepsRawForNormalize(t *testing.T) {
	body := []byte(`{
		"subsonic-response": {
			"status": "ok",
			"lyrics": {
				"artist": "Band",
				"title": "Song",
				"value": "[00:01.00]First\n[00:03.00]Second"
			}
		}
	}`)
	doc, err := parseLyricsResponse(body)
	if err != nil {
		t.Fatalf("parseLyricsResponse: %v", err)
	}
	if !strings.Contains(doc.RawValue, "[00:01.00]") {
		t.Fatalf("expected LRC markers in RawValue, got %q", doc.RawValue)
	}
}
