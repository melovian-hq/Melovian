// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package melog

import (
	"strings"
	"testing"
	"unicode"
)

func TestSanitizeFilenameOracle(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "", want: "unknown"},
		{in: "!!!", want: "---"},
		{in: "panic", want: "panic"},
		{in: "ok_Name-1", want: "ok-Name-1"},
		{in: "../etc/passwd", want: "---etc-passwd"},
		{in: "a/b\\c:d", want: "a-b-c-d"},
	}

	for _, tc := range cases {
		got := sanitizeFilename(tc.in)
		if got != tc.want {
			t.Fatalf("sanitizeFilename(%q) = %q, want %q", tc.in, got, tc.want)
		}
		if got == "" {
			t.Fatalf("sanitizeFilename(%q) returned empty", tc.in)
		}
		for _, r := range got {
			if r == '/' || r == '\\' || r == ':' {
				t.Fatalf("sanitizeFilename(%q)=%q contains path separator", tc.in, got)
			}
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
				continue
			}
			t.Fatalf("sanitizeFilename(%q)=%q has unexpected rune %q", tc.in, got, r)
		}
	}
}

func TestSanitizeFilenameOracleAlnumPreserved(t *testing.T) {
	in := "CrashReport42"
	got := sanitizeFilename(in)
	if got != in {
		t.Fatalf("expected alnum preserved, got %q", got)
	}
	if strings.ContainsAny(got, "/\\:") {
		t.Fatalf("unexpected separators in %q", got)
	}
}
