// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package compat

import (
	"math/rand"
	"slices"
	"strings"
	"testing"

	"melovian/internal/update"
)

// refParseCapabilities is an independent oracle using strings.Fields and a
// different loop structure.
func refParseCapabilities(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	for p := range strings.SplitSeq(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// refHasCapability is an independent oracle using a manual loop.
func refHasCapability(caps []string, cap string) bool {
	return slices.Contains(caps, cap)
}

func TestParseCapabilitiesMatchesReferenceOracle(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for range 500 {
		raw := randomCapabilityHeader(r)
		got := ParseCapabilities(raw)
		want := refParseCapabilities(raw)
		if len(got) != len(want) {
			t.Fatalf("%q got %v, ref %v", raw, got, want)
		}
		for j, c := range want {
			if got[j] != c {
				t.Fatalf("%q at %d got %q, ref %q", raw, j, got[j], c)
			}
		}
	}
}

func TestHasCapabilityMatchesReferenceOracle(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for range 500 {
		caps := randomCapabilityList(r)
		cap := randomCapability(r)
		got := HasCapability(caps, cap)
		want := refHasCapability(caps, cap)
		if got != want {
			t.Fatalf("HasCapability(%v, %q) got %v, ref %v", caps, cap, got, want)
		}
	}
}

func TestCompareSemverMatchesUpdateOracle(t *testing.T) {
	// For non-prerelease dotted versions, compat.CompareSemver should agree with
	// the update package's stricter Compare. This is a cross-package differential.
	r := rand.New(rand.NewSource(20260909))
	for range 1000 {
		a := randomSemver(r)
		b := randomSemver(r)
		got := CompareSemver(a, b)
		want := update.Compare(a, b)
		if got != want {
			t.Fatalf("differential mismatch for (%q,%q): compat=%d, update=%d", a, b, got, want)
		}
	}
}

func randomCapabilityHeader(r *rand.Rand) string {
	caps := randomCapabilityList(r)
	if len(caps) == 0 {
		return ""
	}
	// Inject arbitrary spaces around the commas.
	var parts []string
	for _, c := range caps {
		spaces := strings.Repeat(" ", r.Intn(3))
		parts = append(parts, spaces+c)
	}
	return strings.Join(parts, ",")
}
