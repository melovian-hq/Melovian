// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package video

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizeInstanceURL(t *testing.T) {
	got, err := NormalizeInstanceURL("https://invidious.example.com/")
	if err != nil {
		t.Fatalf("NormalizeInstanceURL: %v", err)
	}
	if got != "https://invidious.example.com" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeInstanceURL("ftp://bad"); err == nil {
		t.Fatal("expected error for ftp")
	}
}

func TestEmbedURL(t *testing.T) {
	got, err := EmbedURL("https://invidious.example.com", "abc123")
	if err != nil {
		t.Fatalf("EmbedURL: %v", err)
	}
	if got != "https://invidious.example.com/embed/abc123" {
		t.Fatalf("got %q", got)
	}
}

func TestGetInvidiousVideo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/videos/Vu0nRz_bQ0Y" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"title":"Official Music Video",
			"author":"Artist",
			"videoId":"Vu0nRz_bQ0Y",
			"lengthSeconds":210
		}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient()
	hit, err := client.GetInvidiousVideo(context.Background(), server.URL, "Vu0nRz_bQ0Y")
	if err != nil {
		t.Fatalf("GetInvidiousVideo: %v", err)
	}
	if hit.Title != "Official Music Video" {
		t.Fatalf("title=%q", hit.Title)
	}
	if hit.Author != "Artist" {
		t.Fatalf("author=%q", hit.Author)
	}
	if hit.ID != "Vu0nRz_bQ0Y" {
		t.Fatalf("id=%q", hit.ID)
	}
}

func TestYouTubeEmbedURL(t *testing.T) {
	got, err := YouTubeEmbedURL("yt123")
	if err != nil {
		t.Fatalf("YouTubeEmbedURL: %v", err)
	}
	if got != "https://www.youtube.com/embed/yt123" {
		t.Fatalf("got %q", got)
	}
}

func BenchmarkEmbedURL(b *testing.B) {
	for b.Loop() {
		_, _ = EmbedURL("https://invidious.example.com", "Vu0nRz_bQ0Y")
	}
}

func BenchmarkNormalizeInstanceURL(b *testing.B) {
	for b.Loop() {
		_, _ = NormalizeInstanceURL("https://invidious.example.com/")
	}
}
