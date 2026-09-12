// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"errors"
	"testing"
)

func TestParseHTTPURL(t *testing.T) {
	ok := []string{
		"http://navidrome.local:4533",
		"https://music.example.com/sub/",
		" http://padded.example.com ",
	}
	for _, raw := range ok {
		u, err := ParseHTTPURL(raw)
		if err != nil {
			t.Fatalf("ParseHTTPURL(%q): %v", raw, err)
		}
		if u.Host == "" {
			t.Fatalf("ParseHTTPURL(%q) lost host", raw)
		}
	}
	for _, raw := range []string{"file:///etc/passwd", "ftp://x", "not a url", "//host/path"} {
		if _, err := ParseHTTPURL(raw); !errors.Is(err, ErrURLScheme) {
			t.Fatalf("ParseHTTPURL(%q) should fail with ErrURLScheme, got %v", raw, err)
		}
	}
	for _, raw := range []string{"", "  ", "http://"} {
		if _, err := ParseHTTPURL(raw); !errors.Is(err, ErrInvalidURL) {
			t.Fatalf("ParseHTTPURL(%q) should fail with ErrInvalidURL, got %v", raw, err)
		}
	}
	// userinfo parses fine; callers that store the URL reject it themselves
	u, err := ParseHTTPURL("http://user:pass@example.com")
	if err != nil || u.User == nil {
		t.Fatalf("userinfo should parse: u=%v err=%v", u, err)
	}
}

func TestNormalizeHTTPURL(t *testing.T) {
	got, err := NormalizeHTTPURL("https://ex.com/base/?token=secret#frag")
	if err != nil {
		t.Fatalf("NormalizeHTTPURL: %v", err)
	}
	if got != "https://ex.com/base" {
		t.Fatalf("NormalizeHTTPURL = %q, want query and fragment stripped", got)
	}
	if _, err := NormalizeHTTPURL("gopher://x"); !errors.Is(err, ErrURLScheme) {
		t.Fatalf("expected ErrURLScheme, got %v", err)
	}
}
