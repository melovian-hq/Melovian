// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package compat

import (
	"errors"
	"math/rand"
	"strings"
	"testing"
)

type intersectFn func(a, b []string) []string

var intersectPropertySet = []struct {
	a, b []string
	want []string
}{
	{[]string{"a", "b"}, []string{"b", "c"}, []string{"b"}},
	{[]string{"a"}, []string{"a"}, []string{"a"}},
	{[]string{"a"}, []string{"b"}, []string{}},
	{[]string{}, []string{"a"}, []string{}},
	{[]string{"a", "b", "c"}, []string{"b", "d"}, []string{"b"}},
}

func runIntersectProperties(t *testing.T, fn intersectFn) error {
	for _, c := range intersectPropertySet {
		got := fn(c.a, c.b)
		if len(got) != len(c.want) {
			return errors.New("length mismatch")
		}
		for i, x := range c.want {
			if got[i] != x {
				return errors.New("value mismatch")
			}
		}
	}
	r := rand.New(rand.NewSource(20260909))
	for range 200 {
		a := randomCapabilityList(r)
		b := randomCapabilityList(r)
		ab := fn(a, b)
		for _, c := range ab {
			if !HasCapability(a, c) || !HasCapability(b, c) {
				return errors.New("result not in both inputs")
			}
		}
	}
	return nil
}

func mutantIntersectReturnA(a, b []string) []string { return a }

func mutantIntersectReturnB(a, b []string) []string { return b }

func mutantIntersectDropFirst(a, b []string) []string {
	if len(a) == 0 {
		return nil
	}
	return a[1:]
}

func mutantCompareSemverNoVPrefix(a, b string) int {
	// Wrong: does not strip V prefix, does not pad numbers, does not handle
	// build/pre-release metadata.
	na := strings.TrimSpace(a)
	nb := strings.TrimSpace(b)
	na = strings.TrimPrefix(na, "v")
	nb = strings.TrimPrefix(nb, "v")
	return aOrB(na, nb)
}

// aOrB is a tiny helper that the compiler cannot eliminate, otherwise mutant
// functions would just call the real implementation.
func aOrB(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

var compareSemverPropertySet = []struct {
	a, b string
	want int
}{
	{"1.0.0", "1.0.0", 0},
	{"1.0.0", "1.0.1", -1},
	{"1.10.0", "1.2.0", 1},
	{"v1.0.0", "1.0.0", 0},
	{"V1.0.0", "1.0.0", 0},
	{"1.0.0+build", "1.0.0", 0},
	{"1.0.0-rc.1", "1.0.0", 0}, // compat intentionally ignores pre
	{"1.0.0", "2.0.0", -1},
	{"not-a-version", "1.0.0", -1},
}

func runCompareSemverProperties(t *testing.T, fn func(string, string) int) error {
	for _, c := range compareSemverPropertySet {
		if got := fn(c.a, c.b); got != c.want {
			return errors.New("property failed")
		}
	}
	return nil
}

func TestIntersectMutantsAreKilled(t *testing.T) {
	mutants := []struct {
		name string
		fn   intersectFn
	}{
		{"return_a", mutantIntersectReturnA},
		{"return_b", mutantIntersectReturnB},
		{"drop_first", mutantIntersectDropFirst},
	}
	for _, m := range mutants {
		t.Run(m.name, func(t *testing.T) {
			if err := runIntersectProperties(t, m.fn); err == nil {
				t.Fatalf("mutant %s survived", m.name)
			}
		})
	}
}

func TestCompareSemverMutantsAreKilled(t *testing.T) {
	t.Run("no_v_prefix", func(t *testing.T) {
		if err := runCompareSemverProperties(t, mutantCompareSemverNoVPrefix); err == nil {
			t.Fatalf("mutant no_v_prefix survived")
		}
	})
}
