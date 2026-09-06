// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonic

import (
	"net/url"
	"testing"
)

func FuzzInjectAuth(f *testing.F) {
	f.Add("track-1", "olduser", "oldtoken")
	f.Add("", "", "")

	f.Fuzz(func(t *testing.T, id, oldUser, oldToken string) {
		client := NewClient("http://example.com", "alice", "secret")
		query := url.Values{
			"u":   {oldUser},
			"t":   {oldToken},
			"s":   {"oldsalt"},
			"p":   {"oldpass"},
			"jwt": {"oldjwt"},
		}
		if id != "" {
			query.Set("id", id)
		}

		merged := client.InjectAuth(query)

		if merged.Get("u") != "alice" {
			t.Fatalf("expected injected username, got %q", merged.Get("u"))
		}
		if merged.Get("t") == oldToken || merged.Get("s") == "oldsalt" {
			t.Fatal("expected fresh auth token and salt")
		}
		if merged.Get("p") != "" || merged.Get("jwt") != "" {
			t.Fatal("legacy auth params should not survive injection")
		}
		if id != "" && merged.Get("id") != id {
			t.Fatalf("expected id %q preserved, got %q", id, merged.Get("id"))
		}
		if merged.Get("c") != ClientName || merged.Get("v") != Version {
			t.Fatalf("expected client metadata")
		}
	})
}
