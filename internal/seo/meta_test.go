// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package seo

import (
	"strings"
	"testing"
)

func TestForPath(t *testing.T) {
	page := ForPath("/music/albums")
	if page.Title != "Albums" {
		t.Fatalf("title %q", page.Title)
	}
	if !strings.Contains(page.Description, "albums") {
		t.Fatalf("description %q", page.Description)
	}

	settings := ForPath("/settings/playback")
	if settings.Index {
		t.Fatal("settings should be noindex")
	}

	unknown := ForPath("/nope")
	if unknown.Title == "" || unknown.Description == "" {
		t.Fatal("unknown path should still have defaults")
	}
}

func TestInject(t *testing.T) {
	raw := []byte(`<!doctype html><html><head>
<title>Old</title>
<meta name="description" content="old" />
</head><body></body></html>`)
	page := Page{
		Title:       "Albums",
		Description: "Browse albums",
		Image:       "/og.png",
		Type:        "website",
		Index:       true,
	}
	out := string(Inject(raw, page, "https://music.example", "/music/albums"))
	checks := []string{
		"<title>Albums · Melovian</title>",
		`content="Browse albums"`,
		`property="og:url" content="https://music.example/music/albums"`,
		`property="og:image" content="https://music.example/og.png"`,
		`rel="canonical" href="https://music.example/music/albums"`,
		`name="twitter:card" content="summary_large_image"`,
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}
