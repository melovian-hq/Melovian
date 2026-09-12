// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package update

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"melovian/internal/compat"
)

func TestParseSemverMetamorphicBuildMetadata(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for range 200 {
		base := randomSemver(r)
		v, err := ParseSemver(base)
		if err != nil {
			t.Fatalf("base %q: %v", base, err)
		}
		withBuild := base + "+build." + randomIdentifier(r)
		v2, err := ParseSemver(withBuild)
		if err != nil {
			t.Fatalf("build variant %q: %v", withBuild, err)
		}
		if v != v2 {
			t.Fatalf("build metadata changed parse: %q -> %+v, %q -> %+v", base, v, withBuild, v2)
		}
	}
}

func TestParseSemverMetamorphicVPrefix(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for range 200 {
		s := randomSemver(r)
		bare := strings.TrimPrefix(s, "v")
		withV := "v" + bare
		v, err := ParseSemver(bare)
		if err != nil {
			t.Fatalf("%q: %v", bare, err)
		}
		v2, err := ParseSemver(withV)
		if err != nil {
			t.Fatalf("%q: %v", withV, err)
		}
		if v != v2 {
			t.Fatalf("v prefix changed parse: %q -> %+v, %q -> %+v", bare, v, withV, v2)
		}
	}
}

func TestCompareMetamorphicPrereleaseLowerThanRelease(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for range 200 {
		base := randomSemverWithoutPre(r)
		pre := base + "-" + randomPrerelease(r)
		if got := Compare(pre, base); got != -1 {
			t.Fatalf("Compare(%q,%q)=%d, want -1", pre, base, got)
		}
		if got := Compare(base, pre); got != 1 {
			t.Fatalf("Compare(%q,%q)=%d, want 1", base, pre, got)
		}
	}
}

func TestCompareMetamorphicBuildDoesNotAffectOrder(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for range 200 {
		a := randomSemver(r)
		b := randomSemver(r)
		c1 := Compare(a, b)
		c2 := Compare(a+"+build.1", b+"+build.2")
		if c1 != c2 {
			t.Fatalf("build changed order: Compare(%q,%q)=%d, Compare(%q,%q)=%d", a, b, c1, a+"+build.1", b+"+build.2", c2)
		}
	}
}

func TestCompareDifferentialUpdateVsCompat(t *testing.T) {
	// For non-prerelease semver versions, update.Compare and compat.CompareSemver
	// must agree. They diverge by design when prerelease suffixes are present.
	r := rand.New(rand.NewSource(20260909))
	for range 500 {
		a := randomSemverWithoutPre(r)
		b := randomSemverWithoutPre(r)
		got := Compare(a, b)
		want := compat.CompareSemver(a, b)
		if got != want {
			t.Fatalf("differential mismatch Compare(%q,%q)=%d, compat=%d", a, b, got, want)
		}
	}
}

func TestIsNewerMetamorphic(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for range 500 {
		current := randomVersion(r)
		candidate := randomVersion(r)
		_, parseErr := ParseSemver(current)
		got := IsNewer(current, candidate)
		want := parseErr != nil || Compare(candidate, current) > 0
		if got != want {
			t.Fatalf("IsNewer(%q,%q)=%v, want %v", current, candidate, got, want)
		}
	}
}

func randomSemverWithoutPre(r *rand.Rand) string {
	major := r.Intn(100)
	minor := r.Intn(100)
	patch := r.Intn(100)
	return maybeVPrefix(r, fmt.Sprintf("%d.%d.%d", major, minor, patch))
}
