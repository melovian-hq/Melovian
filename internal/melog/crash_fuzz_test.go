// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package melog

import (
	"strings"
	"testing"
)

func TestSanitizeFilenameCrashSafe(t *testing.T) {
	cases := []string{"", "!!!", "../etc/passwd", "panic kind", "ok_Name-1", strings.Repeat("x", 300)}
	for _, raw := range cases {
		got := sanitizeFilename(raw)
		if got == "" {
			t.Fatalf("sanitizeFilename(%q) empty", raw)
		}
		if strings.ContainsAny(got, "/\\:") {
			t.Fatalf("sanitizeFilename(%q)=%q contains path separators", raw, got)
		}
	}
}

func FuzzSanitizeFilename(f *testing.F) {
	f.Add("panic")
	f.Add("../x")
	f.Add("")
	f.Fuzz(func(t *testing.T, raw string) {
		got := sanitizeFilename(raw)
		if got == "" {
			t.Fatal("empty sanitized name")
		}
	})
}
