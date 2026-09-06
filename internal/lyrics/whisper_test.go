// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lyrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseWhisperResponseVerboseJSON(t *testing.T) {
	body := `{"task":"transcribe","segments":[{"id":0,"start":1.25,"end":4.5,"text":" hello world "},{"id":1,"start":4.5,"end":8.0,"text":"second line"}]}`
	segments, err := ParseWhisperResponse([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(segments) != 2 {
		t.Fatalf("segments = %d, want 2", len(segments))
	}
	if segments[0].StartMs != 1250 || segments[0].Text != "hello world" {
		t.Fatalf("segment 0 = %+v", segments[0])
	}
}

func TestParseWhisperResponseTranscriptionOffsets(t *testing.T) {
	body := `{"transcription":[{"timestamps":{"from":"00:00:01,000","to":"00:00:03,000"},"offsets":{"from":1000,"to":3000},"text":"line one"}]}`
	segments, err := ParseWhisperResponse([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(segments) != 1 || segments[0].StartMs != 1000 {
		t.Fatalf("segments = %+v", segments)
	}
}

func TestParseWhisperResponseEmpty(t *testing.T) {
	if _, err := ParseWhisperResponse([]byte(`{"text":"hi"}`)); err == nil {
		t.Fatal("expected error for response without segments")
	}
}

func TestDocumentFromWhisper(t *testing.T) {
	doc, err := DocumentFromWhisper(FetchInput{Artist: "A", Title: "T"}, []WhisperSegment{
		{StartMs: 1250, Text: "hello"},
		{StartMs: 65000, Text: "world"},
		{StartMs: 70000, Text: "  "},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !doc.Synced {
		t.Fatal("expected synced document")
	}
	if doc.Source != WhisperSourceID {
		t.Fatalf("source = %q", doc.Source)
	}
	if len(doc.Lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(doc.Lines))
	}
	if !strings.Contains(doc.RawValue, "[00:01.25] hello") {
		t.Fatalf("raw missing LRC timestamp: %q", doc.RawValue)
	}
	if !strings.Contains(doc.RawValue, "[01:05.00] world") {
		t.Fatalf("raw missing LRC timestamp: %q", doc.RawValue)
	}
}

func TestTranscribeWithWhisperServer(t *testing.T) {
	var sawFile bool
	var sawFormat bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/inference" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("multipart: %v", err)
		}
		sawFile = r.MultipartForm != nil && len(r.MultipartForm.File["file"]) == 1
		sawFormat = r.FormValue("response-format") == "verbose_json"
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"segments":[{"start":0.5,"end":2.0,"text":"hi"}]}`))
	}))
	defer srv.Close()

	segments, err := TranscribeWithWhisperServer(context.Background(), srv.Client(), srv.URL, "a.wav", []byte("RIFFdata"), "en")
	if err != nil {
		t.Fatal(err)
	}
	if !sawFile || !sawFormat {
		t.Fatalf("file=%v format=%v", sawFile, sawFormat)
	}
	if len(segments) != 1 || segments[0].StartMs != 500 || segments[0].Text != "hi" {
		t.Fatalf("segments = %+v", segments)
	}
}

func TestWhisperServerURLValidation(t *testing.T) {
	if _, err := WhisperServerURL("  "); err == nil {
		t.Fatal("expected error for empty URL")
	}
	if _, err := WhisperServerURL("ftp://x"); err == nil {
		t.Fatal("expected error for non-http scheme")
	}
	got, err := WhisperServerURL("http://127.0.0.1:8080/")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://127.0.0.1:8080" {
		t.Fatalf("url = %q", got)
	}
}
