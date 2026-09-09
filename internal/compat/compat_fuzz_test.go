// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package compat

import (
	"slices"
	"strings"
	"testing"
)

func FuzzParseCapabilities(f *testing.F) {
	f.Add("browse,playback,auth")
	f.Add(" browse , , playback ")
	f.Add("")
	f.Add(",")

	f.Fuzz(func(t *testing.T, raw string) {
		caps := ParseCapabilities(raw)
		for _, c := range caps {
			if c == "" {
				t.Fatalf("ParseCapabilities returned empty token from %q", raw)
			}
			if c != strings.TrimSpace(c) {
				t.Fatalf("ParseCapabilities returned un-trimmed token %q from %q", c, raw)
			}
		}
	})
}

func FuzzJoinCapabilities(f *testing.F) {
	f.Add("browse,playback")
	f.Add("")
	f.Add(",,browse")

	f.Fuzz(func(t *testing.T, raw string) {
		caps := ParseCapabilities(raw)
		joined := JoinCapabilities(caps)
		reparsed := ParseCapabilities(joined)
		if !slices.Equal(caps, reparsed) {
			t.Fatalf("Parse(%q)=%v, Join=%q, Reparse=%v", raw, caps, joined, reparsed)
		}
	})
}

func FuzzIntersect(f *testing.F) {
	f.Add("a,b", "b,c")
	f.Add("", "a")
	f.Add("x", "x")

	f.Fuzz(func(t *testing.T, a, b string) {
		alist := ParseCapabilities(a)
		blist := ParseCapabilities(b)
		ab := Intersect(alist, blist)
		for _, c := range ab {
			if !HasCapability(alist, c) || !HasCapability(blist, c) {
				t.Fatalf("%v in Intersect(%v,%v) but not in both inputs", c, alist, blist)
			}
		}
	})
}

func FuzzCompareSemver(f *testing.F) {
	f.Add("1.0.0", "1.0.1")
	f.Add("1.10.0", "1.2.0")
	f.Add("v1.0.0", "1.0.0")
	f.Add("1.0.0-rc.1", "1.0.0")
	f.Add("dev", "1.0.0")

	f.Fuzz(func(t *testing.T, a, b string) {
		got := CompareSemver(a, b)
		if got < -1 || got > 1 {
			t.Fatalf("CompareSemver(%q,%q)=%d out of range", a, b, got)
		}
		swapped := CompareSemver(b, a)
		if got != -swapped {
			t.Fatalf("antisymmetric: CompareSemver(%q,%q)=%d, CompareSemver(%q,%q)=%d", a, b, got, b, a, swapped)
		}
	})
}
