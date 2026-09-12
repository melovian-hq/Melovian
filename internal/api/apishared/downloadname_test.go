// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package apishared

import "testing"

func TestContentTypeExtensionVideoVsAudio(t *testing.T) {
	cases := []struct {
		ct   string
		want string
	}{
		{"video/mp4", ".mp4"},
		{"video/webm", ".webm"},
		{"audio/mp4", ".m4a"},
		{"audio/mpeg", ".mp3"},
		{"audio/flac", ".flac"},
	}
	for _, tc := range cases {
		if got := ContentTypeExtension(tc.ct); got != tc.want {
			t.Fatalf("ContentTypeExtension(%q)=%q want %q", tc.ct, got, tc.want)
		}
	}
}
