// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package update

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ErrUnsignedRelease is returned when a public key is compiled in but the
// release has no valid signature. Updates fail closed: unsigned releases
// are never installed when a trust root exists.
var ErrUnsignedRelease = errors.New("release checksums are not signed")

// ErrNoPinnedKey is a soft condition reported when no public key was
// compiled in. Checksum verification still applies.
var ErrNoPinnedKey = errors.New("no release signing key compiled in")

// ParseChecksums parses sha256sum-format output: hex digest, whitespace,
// optional "*" marker, filename.
func ParseChecksums(data []byte) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		sum, name, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		name = strings.TrimPrefix(name, "*")
		if len(sum) == sha256.Size*2 && name != "" {
			out[name] = strings.ToLower(sum)
		}
	}
	return out
}

// VerifySignature checks a detached Ed25519 signature over data. The sig
// file holds base64 text (possibly whitespace-wrapped).
func VerifySignature(data, sigB64 []byte) error {
	key, err := pinnedKey()
	if err != nil {
		return err
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sigB64)))
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	if len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("signature has %d bytes, want %d", len(sig), ed25519.SignatureSize)
	}
	if !ed25519.Verify(key, data, sig) {
		return errors.New("checksum signature does not verify")
	}
	return nil
}

// Pinned reports whether a release signing key is compiled in.
func Pinned() bool {
	_, err := pinnedKey()
	return err == nil
}

// PinnedKeyBytes returns the raw Ed25519 public key compiled in at link
// time, or nil when none is set.
func PinnedKeyBytes() []byte {
	key, err := pinnedKey()
	if err != nil {
		return nil
	}
	return key
}

func pinnedKey() (ed25519.PublicKey, error) {
	if strings.TrimSpace(ReleasePublicKey) == "" {
		return nil, ErrNoPinnedKey
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(ReleasePublicKey))
	if err != nil {
		return nil, fmt.Errorf("decode pinned public key: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("pinned public key has %d bytes, want %d", len(raw), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(raw), nil
}

// VerifyChecksum streams r through sha256 and compares with the hex digest
// recorded for name in sums.
func VerifyChecksum(r io.Reader, sums map[string]string, name string) error {
	want, ok := sums[name]
	if !ok {
		return fmt.Errorf("no checksum entry for %s", name)
	}
	return VerifyDigest(r, want)
}

// VerifyDigest hashes r and compares with the expected hex sha256.
func VerifyDigest(r io.Reader, wantHex string) error {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != strings.ToLower(strings.TrimSpace(wantHex)) {
		return fmt.Errorf("sha256 mismatch: got %s, want %s", got, wantHex)
	}
	return nil
}
