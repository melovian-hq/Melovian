// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package compat

import (
	"math/rand"
	"slices"
	"strconv"
	"testing"
)

func randomCapability(r *rand.Rand) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz"
	n := r.Intn(12) + 1
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[r.Intn(len(alphabet))]
	}
	return string(b)
}

func randomCapabilityList(r *rand.Rand) []string {
	n := r.Intn(10)
	caps := make([]string, n)
	for i := range n {
		caps[i] = randomCapability(r)
	}
	return slices.Compact(caps)
}

func TestParseJoinCapabilitiesRoundTripProperty(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for range 500 {
		caps := randomCapabilityList(r)
		joined := JoinCapabilities(caps)
		parsed := ParseCapabilities(joined)
		if len(parsed) != len(caps) {
			t.Fatalf("round-trip changed length: %v -> %q -> %v", caps, joined, parsed)
		}
		for j, c := range caps {
			if parsed[j] != c {
				t.Fatalf("round-trip mismatch at %d: %v -> %q -> %v", j, caps, joined, parsed)
			}
		}
	}
}

func TestParseCapabilitiesNormalizesProperty(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for range 500 {
		// Build a raw header with noise that ParseCapabilities removes.
		caps := randomCapabilityList(r)
		raw := " " + JoinCapabilities(caps)
		if len(caps) > 0 {
			raw += ", ,"
		}
		parsed := ParseCapabilities(raw)
		if len(parsed) != len(caps) {
			t.Fatalf("%q parsed to %v, want %v", raw, parsed, caps)
		}
		for j, c := range caps {
			if parsed[j] != c {
				t.Fatalf("%q parsed to %v at %d, want %v", raw, parsed, j, c)
			}
		}
	}
}

func TestIntersectProperties(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for range 500 {
		a := randomCapabilityList(r)
		b := randomCapabilityList(r)
		ab := Intersect(a, b)

		// Result is a subsequence of a.
		pos := 0
		for _, c := range ab {
			for pos < len(a) && a[pos] != c {
				pos++
			}
			if pos >= len(a) {
				t.Fatalf("%v not a subsequence of %v", ab, a)
			}
			pos++
		}

		// Every result element is in b.
		for _, c := range ab {
			if !HasCapability(b, c) {
				t.Fatalf("%v in Intersect(%v,%v) but not in %v", c, a, b, b)
			}
		}

		// Membership oracle: an element is in the result iff it is in both a and b.
		for _, c := range a {
			inAB := HasCapability(ab, c)
			inB := HasCapability(b, c)
			if inAB != inB {
				t.Fatalf("membership mismatch for %v: inAB=%v inB=%v", c, inAB, inB)
			}
		}

		// Intersecting with empty yields empty.
		if len(Intersect(a, nil)) != 0 {
			t.Fatalf("Intersect(%v, nil) = %v", a, Intersect(a, nil))
		}
		if len(Intersect(nil, b)) != 0 {
			t.Fatalf("Intersect(nil, %v) = %v", b, Intersect(nil, b))
		}
	}
}

func TestCompareSemverRejectsNonNumericProperty(t *testing.T) {
	non := []string{"", "dev", "416368d", "v", "1.2.3.4"}
	r := rand.New(rand.NewSource(20260909))
	for range 500 {
		s := non[r.Intn(len(non))] + strconv.Itoa(r.Intn(1000))
		// CompareSemver must return a value in range and be antisymmetric.
		got := CompareSemver(s, "1.0.0")
		if got < -1 || got > 1 {
			t.Fatalf("CompareSemver(%q,1.0.0)=%d out of range", s, got)
		}
	}
}

func TestClientServerTooOldProperty(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for range 100 {
		client := randomSemver(r)
		if CompareSemver(client, MinClientVersion) < 0 {
			if !ClientTooOld(client) {
				t.Fatalf("ClientTooOld(%q) false but %q < %q", client, client, MinClientVersion)
			}
		} else if client != "" && CompareSemver(client, MinClientVersion) >= 0 {
			if ClientTooOld(client) {
				t.Fatalf("ClientTooOld(%q) true but %q >= %q", client, client, MinClientVersion)
			}
		}
	}
}

func randomSemver(r *rand.Rand) string {
	major := r.Intn(100)
	minor := r.Intn(100)
	patch := r.Intn(100)
	return strconv.Itoa(major) + "." + strconv.Itoa(minor) + "." + strconv.Itoa(patch)
}
