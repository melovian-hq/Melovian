// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

const (
	argon2Time    = 2
	argon2Memory  = 19456
	argon2Threads = 1
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

var errInvalidPasswordHash = errors.New("invalid password hash")

// HashPassword returns an argon2id PHC-encoded hash of password.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
	return encodeArgon2id(salt, key, argon2Time, argon2Memory, argon2Threads), nil
}

// VerifyPassword checks password against a bcrypt or argon2id hash.
func VerifyPassword(hash, password string) (bool, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return false, errInvalidPasswordHash
	}
	switch {
	case strings.HasPrefix(hash, "$argon2id$"):
		return verifyArgon2id(hash, password)
	case strings.HasPrefix(hash, "$2a$"),
		strings.HasPrefix(hash, "$2b$"),
		strings.HasPrefix(hash, "$2y$"):
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		if err != nil {
			if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	default:
		return false, errInvalidPasswordHash
	}
}

// NeedsRehash reports whether hash should be upgraded to current argon2id params.
func NeedsRehash(hash string) bool {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return true
	}
	if strings.HasPrefix(hash, "$2a$") ||
		strings.HasPrefix(hash, "$2b$") ||
		strings.HasPrefix(hash, "$2y$") {
		return true
	}
	params, err := parseArgon2idParams(hash)
	if err != nil {
		return true
	}
	return params.time != argon2Time ||
		params.memory != argon2Memory ||
		params.threads != argon2Threads ||
		params.keyLen != argon2KeyLen ||
		params.saltLen != argon2SaltLen
}

type argon2Params struct {
	time    uint32
	memory  uint32
	threads uint8
	saltLen int
	keyLen  int
}

func encodeArgon2id(salt, key []byte, time, memory uint32, threads uint8) string {
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Key := base64.RawStdEncoding.EncodeToString(key)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memory, time, threads, b64Salt, b64Key,
	)
}

func parseArgon2idParams(hash string) (argon2Params, error) {
	parts := strings.Split(hash, "$")
	// "", "argon2id", "v=19", "m=...,t=...,p=...", salt, key
	if len(parts) != 6 || parts[1] != "argon2id" {
		return argon2Params{}, errInvalidPasswordHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return argon2Params{}, errInvalidPasswordHash
	}
	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return argon2Params{}, errInvalidPasswordHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argon2Params{}, errInvalidPasswordHash
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argon2Params{}, errInvalidPasswordHash
	}
	return argon2Params{
		time:    time,
		memory:  memory,
		threads: threads,
		saltLen: len(salt),
		keyLen:  len(key),
	}, nil
}

func verifyArgon2id(hash, password string) (bool, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errInvalidPasswordHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, errInvalidPasswordHash
	}
	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, errInvalidPasswordHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, errInvalidPasswordHash
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, errInvalidPasswordHash
	}
	got := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(want))) //#nosec G115 -- want length is argon hash size
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return false, nil
	}
	return true, nil
}
