// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package compat

import "testing"

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.1.0", "0.1.0", 0},
		{"v0.1.0", "0.1.0", 0},
		{"0.1.0", "0.2.0", -1},
		{"0.2.0", "0.1.9", 1},
		{"0.1", "0.1.0", 0},
		{"1.0.0-rc1", "1.0.0", 0},
		{"0.9.9", "0.10.0", -1},
	}
	for _, tc := range cases {
		got := CompareSemver(tc.a, tc.b)
		if got != tc.want {
			t.Fatalf("CompareSemver(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestClientTooOld(t *testing.T) {
	if ClientTooOld("") {
		t.Fatal("empty client should not be rejected")
	}
	if ClientTooOld(MinClientVersion) {
		t.Fatal("equal min should be accepted")
	}
	if !ClientTooOld("0.0.1") {
		t.Fatal("0.0.1 should be too old")
	}
}

func TestIntersect(t *testing.T) {
	got := Intersect([]string{"a", "b", "c"}, []string{"b", "c", "d"})
	if len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Fatalf("unexpected intersect: %v", got)
	}
}

func TestParseJoinCapabilities(t *testing.T) {
	raw := JoinCapabilities([]string{"browse", "playback"})
	got := ParseCapabilities(raw)
	if len(got) != 2 || got[0] != "browse" || got[1] != "playback" {
		t.Fatalf("round trip failed: %v", got)
	}
}
