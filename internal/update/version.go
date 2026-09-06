// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package update

import (
	"fmt"
	"strconv"
	"strings"
)

// Semver is a parsed semantic version. Pre-release fields lower the
// precedence of a release per semver 2.0.0 rule 11.
type Semver struct {
	Major, Minor, Patch int
	Pre                 string
}

// ParseSemver parses "1.2.3", "v1.2.3", and prerelease suffixes like
// "1.2.3-rc.1". Returns an error for anything that is not semver-shaped.
func ParseSemver(s string) (Semver, error) {
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
		return v, fmt.Errorf("not a semver: %q", s)
	}
	nums := []int{0, 0, 0}
	for i, p := range parts {
		if p == "" {
			return v, fmt.Errorf("not a semver: %q", s)
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, fmt.Errorf("not a semver: %q", s)
		}
		nums[i] = n
	}
	v.Major, v.Minor, v.Patch = nums[0], nums[1], nums[2]
	return v, nil
}

// IsPrerelease reports whether the version carries a pre-release suffix.
func (v Semver) IsPrerelease() bool { return v.Pre != "" }

// Compare returns -1, 0, or 1 ordering a before b per semver rules.
// Build metadata is ignored. Non-semver inputs compare lower than semver.
func Compare(a, b string) int {
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
	switch {
	case va.Pre == "" && vb.Pre == "":
		return 0
	case va.Pre == "":
		return 1
	case vb.Pre == "":
		return -1
	}
	return comparePre(va.Pre, vb.Pre)
}

// IsNewer reports whether candidate is a newer version than current.
// Non-semver current versions (dev builds, commit shas) are always
// considered outdated.
func IsNewer(current, candidate string) bool {
	if _, err := ParseSemver(current); err != nil {
		return true
	}
	return Compare(candidate, current) > 0
}

func comparePre(a, b string) int {
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

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}
