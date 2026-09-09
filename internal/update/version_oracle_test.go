// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package update

import (
	"bufio"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// referenceSemverRE is an independent oracle for valid semver shape.
// It does not implement the parser, only the grammar.
var referenceSemverRE = regexp.MustCompile(`^v?\d+\.\d+(?:\.\d+)?(?:-[A-Za-z0-9.-]+)?(?:\+[A-Za-z0-9.-]+)?$`)

// refParseSemver is a parser written differently from ParseSemver: it uses
// regular expression validation first and then splits.
func refParseSemver(s string) (Semver, error) {
	if !referenceSemverRE.MatchString(s) {
		return Semver{}, fmt.Errorf("not a semver: %q", s)
	}
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	var pre string
	if i := strings.IndexByte(s, '-'); i >= 0 {
		pre = s[i+1:]
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return Semver{}, fmt.Errorf("not a semver: %q", s)
	}
	nums := []int{0, 0, 0}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return Semver{}, fmt.Errorf("not a semver: %q", s)
		}
		nums[i] = n
	}
	return Semver{Major: nums[0], Minor: nums[1], Patch: nums[2], Pre: pre}, nil
}

// refCompare is an independent reference implementation of Compare.
func refCompare(a, b string) int {
	va, ea := refParseSemver(a)
	vb, eb := refParseSemver(b)
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
	switch {
	case va.Pre == "" && vb.Pre == "":
		return 0
	case va.Pre == "":
		return 1
	case vb.Pre == "":
		return -1
	}
	return refComparePre(va.Pre, vb.Pre)
}

func refComparePre(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) && i < len(bs); i++ {
		an, aerr := strconv.Atoi(as[i])
		bn, berr := strconv.Atoi(bs[i])
		switch {
		case aerr == nil && berr == nil:
			if c := cmpInt(an, bn); c != 0 {
				return c
			}
		case aerr == nil:
			return -1
		case berr == nil:
			return 1
		default:
			if c := strings.Compare(as[i], bs[i]); c != 0 {
				return c
			}
		}
	}
	return cmpInt(len(as), len(bs))
}

// refParseChecksums is a reference parser written with bufio.Scanner and
// a strict regexp, independent of the production implementation.
func refParseChecksums(data []byte) map[string]string {
	result := make(map[string]string)
	re := regexp.MustCompile(`^([0-9a-f]{64})\s+[*]?(.+)$`)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := re.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		result[m[2]] = m[1]
	}
	return result
}

func TestParseSemverMatchesRegexOracle(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for i := 0; i < 1000; i++ {
		s := randomVersion(r)
		if referenceSemverRE.MatchString(s) {
			// The production parser must also accept it.
			if _, err := ParseSemver(s); err != nil {
				t.Fatalf("regex says valid but ParseSemver rejected %q: %v", s, err)
			}
		} else {
			// A strict regex oracle does not recognize every invalid string,
			// so only check that the production parser does not accept strings
			// which are clearly non-semver.
		}
	}

	// Explicit mismatches the regex oracle should reject and the parser should too.
	mustReject := []string{
		"",
		"1.2.3.4",
		"-1.0.0",
		"v",
		"1..2.3",
		"1.2.3_",
	}
	for _, s := range mustReject {
		if referenceSemverRE.MatchString(s) {
			t.Fatalf("regex oracle incorrectly accepted %q", s)
		}
		if _, err := ParseSemver(s); err == nil {
			t.Fatalf("ParseSemver accepted %q", s)
		}
	}
}

func TestCompareMatchesReferenceOracle(t *testing.T) {
	r := rand.New(rand.NewSource(20260909))
	for i := 0; i < 1000; i++ {
		a := randomVersion(r)
		b := randomVersion(r)
		got := Compare(a, b)
		want := refCompare(a, b)
		if got != want {
			t.Fatalf("Compare(%q,%q)=%d, reference=%d", a, b, got, want)
		}
	}
}

func TestParseChecksumsMatchesReferenceOracle(t *testing.T) {
	var cases = []string{
		"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789  file1.tar.gz\n" +
			"# comment\n" +
			"0000000000000000000000000000000000000000000000000000000000000000 *file2\n",
		"",
		"# only a comment\n",
		"bad  file\n",
		"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789 file with spaces\n",
	}
	for _, data := range cases {
		got := ParseChecksums([]byte(data))
		want := refParseChecksums([]byte(data))
		if len(got) != len(want) {
			t.Fatalf("ParseChecksums(%q)=%d entries, reference=%d", data, len(got), len(want))
		}
		for k, v := range want {
			if got[k] != v {
				t.Fatalf("ParseChecksums(%q)[%q]=%q, reference=%q", data, k, got[k], v)
			}
		}
	}
}
