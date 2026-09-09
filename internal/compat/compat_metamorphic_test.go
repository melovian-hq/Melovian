// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package compat

import (
	"math/rand"
	"slices"
	"testing"
)

func TestParseJoinMetamorphicRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for i := 0; i < 500; i++ {
		caps := randomCapabilityList(r)
		joined := JoinCapabilities(caps)
		parsed := ParseCapabilities(joined)
		rejoined := JoinCapabilities(parsed)
		reparsed := ParseCapabilities(rejoined)
		if !slices.Equal(parsed, reparsed) {
			t.Fatalf("Parse∘Join not idempotent: %v -> %q -> %v -> %q -> %v", caps, joined, parsed, rejoined, reparsed)
		}
	}
}

func TestIntersectMetamorphicRelations(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for i := 0; i < 500; i++ {
		a := randomCapabilityList(r)
		b := randomCapabilityList(r)

		// Commutativity on the set of capabilities.
		ab := Intersect(a, b)
		ba := Intersect(b, a)
		if !setEqual(ab, ba) {
			t.Fatalf("Intersect(%v,%v)=%v, Intersect(%v,%v)=%v not equal as sets", a, b, ab, b, a, ba)
		}

		// Intersecting with an all-encompassing list returns the original.
		all := append([]string(nil), a...)
		all = append(all, b...)
		all = uniqueCaps(all)
		ia := Intersect(a, all)
		if !slices.Equal(ia, a) {
			t.Fatalf("Intersect(%v, all=%v) = %v, want %v", a, all, ia, a)
		}

		// Membership: cap in a and b iff cap in Intersect(a,b).
		for _, c := range all {
			inAB := HasCapability(ab, c)
			inA := HasCapability(a, c)
			inB := HasCapability(b, c)
			if inAB != (inA && inB) {
				t.Fatalf("membership mismatch for %v: inAB=%v inA=%v inB=%v", c, inAB, inA, inB)
			}
		}
	}
}

func TestCompareSemverMetamorphic(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for i := 0; i < 500; i++ {
		a := randomSemver(r)
		b := randomSemver(r)

		// Antisymmetry.
		got := CompareSemver(a, b)
		if got < -1 || got > 1 {
			t.Fatalf("CompareSemver(%q,%q)=%d out of range", a, b, got)
		}
		if got != -CompareSemver(b, a) {
			t.Fatalf("antisymmetric: CompareSemver(%q,%q)=%d, CompareSemver(%q,%q)=%d", a, b, got, b, a, -got)
		}

		// Build and prerelease metadata are ignored by compat.
		withBuildA := a + "+build.1"
		withPreB := b + "-rc.1"
		if got2 := CompareSemver(withBuildA, withPreB); got2 != got {
			t.Fatalf("metadata changed order: CompareSemver(%q,%q)=%d, CompareSemver(%q,%q)=%d", a, b, got, withBuildA, withPreB, got2)
		}
	}
}

func TestCompareSemverTransitive(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for i := 0; i < 500; i++ {
		a := randomSemver(r)
		b := randomSemver(r)
		c := randomSemver(r)
		ab := CompareSemver(a, b)
		bc := CompareSemver(b, c)
		if ab < 0 && bc < 0 {
			if got := CompareSemver(a, c); got >= 0 {
				t.Fatalf("transitivity: %q<%q and %q<%q but %q vs %q = %d", a, b, b, c, a, c, got)
			}
		}
		if ab > 0 && bc > 0 {
			if got := CompareSemver(a, c); got <= 0 {
				t.Fatalf("transitivity: %q>%q and %q>%q but %q vs %q = %d", a, b, b, c, a, c, got)
			}
		}
		if ab == 0 && bc == 0 {
			if got := CompareSemver(a, c); got != 0 {
				t.Fatalf("transitivity: %q=%q and %q=%q but %q vs %q = %d", a, b, b, c, a, c, got)
			}
		}
	}
}

func setEqual(a, b []string) bool {
	m := make(map[string]int)
	for _, x := range a {
		m[x]++
	}
	for _, x := range b {
		m[x]--
		if m[x] < 0 {
			return false
		}
	}
	for _, v := range m {
		if v != 0 {
			return false
		}
	}
	return true
}

func uniqueCaps(caps []string) []string {
	m := make(map[string]struct{})
	var out []string
	for _, c := range caps {
		if _, ok := m[c]; !ok {
			m[c] = struct{}{}
			out = append(out, c)
		}
	}
	return out
}
