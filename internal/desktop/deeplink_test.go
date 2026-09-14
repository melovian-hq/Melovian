// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package desktop

import "testing"

func TestParseExtensionDeepLink(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"melovian://install-extension/genre-palette", "genre-palette"},
		{"melovian://extension/genre-palette", "genre-palette"},
		{"melovian://extensions/genre-palette", "genre-palette"},
		{"melovian:///install-extension/genre-palette", "genre-palette"},
		{"melovian://install-extension/Genre-Palette", "genre-palette"},
		{"melovian://install-extension/", ""},
		{"melovian://install-extension/../etc", ""},
		{"melovian://install-extension/Bad_ID", ""},
		{"melovian://other/genre-palette", ""},
		{"https://example.com/install-extension/x", ""},
		{"", ""},
		{"not a url", ""},
	} {
		if got := ParseExtensionDeepLink(tc.in); got != tc.want {
			t.Errorf("ParseExtensionDeepLink(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestDeepLinkArgs(t *testing.T) {
	got := DeepLinkArgs([]string{"melovian", "--verbose", "melovian://install-extension/podcast-style"})
	if got != "podcast-style" {
		t.Fatalf("got %q", got)
	}
	if DeepLinkArgs([]string{"melovian", "--verbose"}) != "" {
		t.Fatal("expected empty for no deep link")
	}
}
