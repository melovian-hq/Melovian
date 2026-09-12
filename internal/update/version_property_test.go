// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package update

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

// generators and shared helpers for version property, oracle, mutation,
// metamorphic and fuzz tests.

func semverString(v Semver) string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Pre != "" {
		s += "-" + v.Pre
	}
	return s
}

func randomPrerelease(r *rand.Rand) string {
	n := r.Intn(4) + 1
	parts := make([]string, n)
	for i := range n {
		if r.Intn(2) == 1 {
			parts[i] = strconv.Itoa(r.Intn(100))
		} else {
			parts[i] = randomIdentifier(r)
		}
	}
	return strings.Join(parts, ".")
}

func randomIdentifier(r *rand.Rand) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	n := r.Intn(8) + 1
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[r.Intn(len(alphabet))]
	}
	return string(b)
}

func randomSemver(r *rand.Rand) string {
	major := r.Intn(100)
	minor := r.Intn(100)
	patch := r.Intn(100)
	s := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	if r.Intn(2) == 1 {
		s += "-" + randomPrerelease(r)
	}
	return maybeVPrefix(r, s)
}

func maybeVPrefix(r *rand.Rand, s string) string {
	if r.Intn(2) == 1 {
		return "v" + s
	}
	return s
}

func randomVersion(r *rand.Rand) string {
	switch r.Intn(5) {
	case 0:
		return randomSemver(r)
	case 1:
		return randomNonSemver(r)
	case 2:
		return randomSemver(r) + "+build." + randomIdentifier(r)
	case 3:
		return "v" + randomSemver(r)
	case 4:
		return strings.ToUpper(maybeVPrefix(r, randomSemver(r)))
	}
	return ""
}

func randomNonSemver(r *rand.Rand) string {
	candidates := []string{
		"",
		"latest",
		"dev",
		"416368d-dirty",
		"not-a-version",
		"1.2.3.4.5",
		"1.2",
		"01.02.03",
		"-1.0.0",
		"1.-1.0",
		"1.0",
		"v",
	}
	return candidates[r.Intn(len(candidates))]
}

func TestParseSemverRoundTripProperty(t *testing.T) {
	// parse∘print = id for valid semvers.
	r := rand.New(rand.NewSource(20260909))
	for range 500 {
		in := randomSemver(r)
		v, err := ParseSemver(in)
		if err != nil {
			t.Fatalf("ParseSemver(%q) unexpected error: %v", in, err)
		}
		if v.Major < 0 || v.Minor < 0 || v.Patch < 0 {
			t.Fatalf("negative component in %q: %+v", in, v)
		}
		out := semverString(v)
		v2, err := ParseSemver(out)
		if err != nil {
			t.Fatalf("ParseSemver(%q) from %q unexpected error: %v", out, in, err)
		}
		if v != v2 {
			t.Fatalf("round trip failed: %q -> %+v -> %q -> %+v", in, v, out, v2)
		}
	}
}

func TestParseSemverRejectsInvalidProperty(t *testing.T) {
	invalid := []string{
		"",
		" ",
		"abc1234",
		"1.2.3.4",
		"-1.0.0",
		"1.-1.0",
		"v",
		"1..2.3",
		"1.2.3_",
		"1.2.3.4.5",
		"1.2.3..4",
	}
	for _, in := range invalid {
		if _, err := ParseSemver(in); err == nil {
			t.Fatalf("ParseSemver(%q) expected error", in)
		}
	}
}

func TestCompareAntisymmetricProperty(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for range 1000 {
		a := randomVersion(r)
		b := randomVersion(r)
		c1 := Compare(a, b)
		c2 := Compare(b, a)
		if c1 != -c2 {
			t.Fatalf("Compare(%q,%q)=%d but Compare(%q,%q)=%d", a, b, c1, b, a, c2)
		}
	}
}

func TestCompareTransitiveProperty(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for range 500 {
		a := randomVersion(r)
		b := randomVersion(r)
		c := randomVersion(r)
		ab := Compare(a, b)
		bc := Compare(b, c)
		if ab < 0 && bc < 0 {
			if got := Compare(a, c); got >= 0 {
				t.Fatalf("transitivity: %q<%q and %q<%q but %q vs %q = %d", a, b, b, c, a, c, got)
			}
		}
		if ab > 0 && bc > 0 {
			if got := Compare(a, c); got <= 0 {
				t.Fatalf("transitivity: %q>%q and %q>%q but %q vs %q = %d", a, b, b, c, a, c, got)
			}
		}
		if ab == 0 && bc == 0 {
			if got := Compare(a, c); got != 0 {
				t.Fatalf("transitivity: %q=%q and %q=%q but %q vs %q = %d", a, b, b, c, a, c, got)
			}
		}
	}
}

func TestIsNewerProperty(t *testing.T) {
	// IsNewer(current, candidate) == (current is not semver or candidate > current).
	r := rand.New(rand.NewSource(20260909))
	for range 500 {
		current := randomVersion(r)
		candidate := randomVersion(r)
		got := IsNewer(current, candidate)
		_, parseErr := ParseSemver(current)
		want := parseErr != nil || Compare(candidate, current) > 0
		if got != want {
			t.Fatalf("IsNewer(%q,%q)=%v, want %v", current, candidate, got, want)
		}
	}
}

func TestVerifyDigestProperty(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for range 200 {
		data := make([]byte, r.Intn(256))
		_, _ = r.Read(data)
		h := sha256.Sum256(data)
		want := hex.EncodeToString(h[:])
		if err := VerifyDigest(bytes.NewReader(data), want); err != nil {
			t.Fatalf("VerifyDigest valid hash: %v", err)
		}
		bad := hex.EncodeToString(h[1:]) + "00"
		if err := VerifyDigest(bytes.NewReader(data), bad); err == nil {
			t.Fatalf("VerifyDigest accepted bad hash")
		}
	}
}
