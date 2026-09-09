// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package update

import (
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

// A compareFn matches the signature of Compare for mutation testing.
type compareFn func(a, b string) int

// A parseFn matches the signature of ParseSemver for mutation testing.
type parseFn func(string) (Semver, error)

// comparePropertySet is a list of strong claims any correct Compare must pass.
// If a mutant survives this, the property set is too weak.
var comparePropertySet = []struct {
	a, b string
	want int
}{
	{"1.0.0", "1.0.0", 0},
	{"1.0.0", "1.0.1", -1},
	{"1.0.1", "1.0.0", 1},
	{"1.2.0", "1.10.0", -1},
	{"2.0.0", "1.9.9", 1},
	{"1.0.0-rc.1", "1.0.0", -1},
	{"1.0.0", "1.0.0-rc.1", 1},
	{"1.0.0-10", "1.0.0-2", 1},
	{"1.0.0-alpha", "1.0.0-rc.1", -1},
	{"v1.0.0", "1.0.0", 0},
	{"1.0.0+build.5", "1.0.0", 0},
	{"not-a-version", "1.0.0", -1},
	{"1.0.0", "not-a-version", 1},
	{"abc", "def", -1}, // transitivity of non-semver lexicographic order
}

// parsePropertySet is a list of strong claims any correct ParseSemver must pass.
var parsePropertySet = []struct {
	in    string
	valid bool
	want  Semver
}{
	{"1.2.3", true, Semver{1, 2, 3, ""}},
	{"v1.2.3", true, Semver{1, 2, 3, ""}},
	{"1.2.3-rc.1", true, Semver{1, 2, 3, "rc.1"}},
	{"1.2.3+build.5", true, Semver{1, 2, 3, ""}},
	{"", false, Semver{}},
	{"-1.0.0", false, Semver{}},
	{"1.-1.0", false, Semver{}},
	{"1.2.3.4", false, Semver{}},
	{"1.2.3-", false, Semver{}},
	{"1..2.3", false, Semver{}},
	{"v", false, Semver{}},
	{"01.0.0", true, Semver{1, 0, 0, ""}}, // leading zero accepted by production
}

func runCompareProperties(t *testing.T, fn compareFn) error {
	for _, c := range comparePropertySet {
		if got := fn(c.a, c.b); got != c.want {
			return errors.New("property failed")
		}
	}
	r := rand.New(rand.NewSource(20260909))
	for i := 0; i < 200; i++ {
		a := randomVersion(r)
		b := randomVersion(r)
		if fn(a, b) != -fn(b, a) {
			return errors.New("antisymmetric property failed")
		}
	}
	return nil
}

func runParseProperties(t *testing.T, fn parseFn) error {
	for _, c := range parsePropertySet {
		got, err := fn(c.in)
		if c.valid {
			if err != nil {
				return errors.New("unexpected error")
			}
			if got != c.want {
				return errors.New("parse mismatch")
			}
		} else {
			if err == nil {
				return errors.New("expected error")
			}
		}
	}
	return nil
}

// mutants for Compare.

func mutantCompareIgnorePrerelease(a, b string) int {
	va, ea := ParseSemver(a)
	vb, eb := ParseSemver(b)
	if ea != nil && eb != nil {
		return strings.Compare(a, b)
	}
	if ea != nil {
		return -1
	}
	if eb != nil {
		return 1
	}
	if va.Major != vb.Major {
		return cmpInt(va.Major, vb.Major)
	}
	if va.Minor != vb.Minor {
		return cmpInt(va.Minor, vb.Minor)
	}
	if va.Patch != vb.Patch {
		return cmpInt(va.Patch, vb.Patch)
	}
	return 0 // wrong: ignores prerelease
}

func mutantCompareStringPre(a, b string) int {
	va, ea := ParseSemver(a)
	vb, eb := ParseSemver(b)
	if ea != nil && eb != nil {
		return strings.Compare(a, b)
	}
	if ea != nil {
		return -1
	}
	if eb != nil {
		return 1
	}
	if va.Major != vb.Major {
		return cmpInt(va.Major, vb.Major)
	}
	if va.Minor != vb.Minor {
		return cmpInt(va.Minor, vb.Minor)
	}
	if va.Patch != vb.Patch {
		return cmpInt(va.Patch, vb.Patch)
	}
	if va.Pre == "" && vb.Pre == "" {
		return 0
	}
	if va.Pre == "" {
		return 1
	}
	if vb.Pre == "" {
		return -1
	}
	return strings.Compare(va.Pre, vb.Pre) // wrong: numeric ids compared as strings
}

func mutantCompareAlwaysEqual(a, b string) int { return 0 }

func mutantCompareNonSemverHigh(a, b string) int {
	_, ea := ParseSemver(a)
	_, eb := ParseSemver(b)
	if ea != nil && eb != nil {
		return -strings.Compare(a, b) // wrong: non-semver higher, not lower
	}
	return Compare(a, b)
}

// mutants for ParseSemver.

func mutantParseSemverAllowNegativeMajor(s string) (Semver, error) {
	var v Semver
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	if i := strings.IndexByte(s, '-'); i >= 0 {
		v.Pre = s[i+1:]
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return v, errors.New("not a semver")
	}
	nums := []int{0, 0, 0}
	for i, p := range parts {
		if p == "" {
			return v, errors.New("not a semver")
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return v, errors.New("not a semver")
		}
		nums[i] = n // wrong: allows negative
	}
	v.Major, v.Minor, v.Patch = nums[0], nums[1], nums[2]
	return v, nil
}

func mutantParseSemverNoVPrefix(s string) (Semver, error) {
	s = strings.TrimSpace(s)
	// wrong: does not strip v, so "v1.0.0" fails
	return ParseSemver(s)
}

func TestCompareMutantsAreKilled(t *testing.T) {
	mutants := []struct {
		name string
		fn   compareFn
	}{
		{"ignore_prerelease", mutantCompareIgnorePrerelease},
		{"string_prerelease", mutantCompareStringPre},
		{"always_equal", mutantCompareAlwaysEqual},
		{"nonsemver_high", mutantCompareNonSemverHigh},
	}
	for _, m := range mutants {
		t.Run(m.name, func(t *testing.T) {
			if err := runCompareProperties(t, m.fn); err == nil {
				t.Fatalf("mutant %s survived the property suite", m.name)
			}
		})
	}
}

func TestParseSemverMutantsAreKilled(t *testing.T) {
	mutants := []struct {
		name string
		fn   parseFn
	}{
		{"allow_negative_major", mutantParseSemverAllowNegativeMajor},
		{"no_v_prefix", mutantParseSemverNoVPrefix},
	}
	for _, m := range mutants {
		t.Run(m.name, func(t *testing.T) {
			if err := runParseProperties(t, m.fn); err == nil {
				t.Fatalf("mutant %s survived the property suite", m.name)
			}
		})
	}
}
