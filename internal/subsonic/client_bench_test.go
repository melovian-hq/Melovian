// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonic

import (
	"net/url"
	"testing"
)

func BenchmarkInjectAuth(b *testing.B) {
	client := NewClient("http://127.0.0.1:4040", "user", "secret")
	query := url.Values{
		"id":    {"abc123"},
		"limit": {"50"},
		"sort":  {"name"},
	}

	b.ReportAllocs()
	for b.Loop() {
		_ = client.InjectAuth(query)
	}
}
