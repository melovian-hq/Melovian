// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package update

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func FuzzParseSemver(f *testing.F) {
	f.Add("1.2.3")
	f.Add("v1.2.3-rc.1")
	f.Add("1.2.3+build.1")
	f.Add("1.2.3-rc.1+build.2")
	f.Add("not-a-version")
	f.Add("")
	f.Add("-1.0.0")

	f.Fuzz(func(t *testing.T, s string) {
		v, err := ParseSemver(s)
		if err != nil {
			return
		}
		if v.Major < 0 || v.Minor < 0 || v.Patch < 0 {
			t.Fatalf("negative component in %q: %+v", s, v)
		}
		// Canonical print must round-trip.
		out := semverString(v)
		v2, err := ParseSemver(out)
		if err != nil {
			t.Fatalf("ParseSemver(%q) from %q: %v", out, s, err)
		}
		if v != v2 {
			t.Fatalf("round trip failed: %q -> %+v -> %q -> %+v", s, v, out, v2)
		}
	})
}

func FuzzCompare(f *testing.F) {
	f.Add("1.0.0", "1.0.1")
	f.Add("1.0.1", "1.0.0")
	f.Add("1.0.0-rc.1", "1.0.0")
	f.Add("abc", "1.0.0")
	f.Add("1.0.0", "1.0.0")

	f.Fuzz(func(t *testing.T, a, b string) {
		got := Compare(a, b)
		if got < -1 || got > 1 {
			t.Fatalf("Compare(%q,%q)=%d out of range", a, b, got)
		}
		swapped := Compare(b, a)
		if got != -swapped {
			t.Fatalf("antisymmetric: Compare(%q,%q)=%d, Compare(%q,%q)=%d", a, b, got, b, a, swapped)
		}
		// Compare must be deterministic.
		if Compare(a, b) != got {
			t.Fatalf("Compare not deterministic for %q vs %q", a, b)
		}
	})
}

func FuzzParseChecksums(f *testing.F) {
	f.Add([]byte("abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789  file.tar.gz\n"))
	f.Add([]byte("# comment\n\n"))
	f.Add([]byte("short  file\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		sums := ParseChecksums(data)
		for name, sum := range sums {
			if name == "" {
				t.Fatalf("empty file name in checksums")
			}
			if len(sum) != 64 {
				t.Fatalf("checksum %q for %q has length %d", sum, name, len(sum))
			}
			for _, c := range sum {
				if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
					t.Fatalf("checksum %q for %q has non-hex char %q", sum, name, c)
				}
			}
		}
	})
}

func FuzzVerifyDigest(f *testing.F) {
	data := []byte("hello fuzzer")
	h := sha256.Sum256(data)
	f.Add(data, hex.EncodeToString(h[:]))
	f.Add([]byte("bye"), hex.EncodeToString(h[:]))
	f.Add([]byte(""), "")

	f.Fuzz(func(t *testing.T, data []byte, hexStr string) {
		err := VerifyDigest(bytes.NewReader(data), hexStr)
		want := sha256.Sum256(data)
		wantHex := hex.EncodeToString(want[:])
		if err == nil && hexStr != wantHex {
			t.Fatalf("VerifyDigest accepted %q for data with hash %q", hexStr, wantHex)
		}
		if err != nil && hexStr == wantHex {
			t.Fatalf("VerifyDigest rejected valid hash %q: %v", hexStr, err)
		}
		// The only acceptable pass is exact match; anything else must fail.
		if err == nil && hexStr == wantHex {
			return
		}
		if err != nil {
			return
		}
		t.Fatalf("unexpected pass for %q", hexStr)
	})
}

func FuzzVerifySignatureFormat(f *testing.F) {
	f.Add([]byte("data"), []byte("not-base64"))
	f.Add([]byte("data"), []byte(""))

	f.Fuzz(func(t *testing.T, data, sig []byte) {
		// Without the pinned public key, VerifySignature must fail closed.
		old := ReleasePublicKey
		ReleasePublicKey = ""
		defer func() { ReleasePublicKey = old }()
		if err := VerifySignature(data, sig); err == nil {
			if ReleasePublicKey == "" {
				t.Fatalf("VerifySignature passed with no pinned key")
			}
		}
	})
}
