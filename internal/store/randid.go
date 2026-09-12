// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"crypto/rand"
	"encoding/hex"
)

// randomHex returns n cryptographically random bytes hex encoded.
func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// mustRandomHex is randomHex for call sites that cannot return an error. A
// failing crypto/rand means the system is broken; panicking is safer than the
// old timestamp fallbacks, which produced guessable ids.
func mustRandomHex(n int) string {
	s, err := randomHex(n)
	if err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return s
}
