// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// secretCipher encrypts small secret values stored in the database with
// AES-256-GCM. Ciphertext is stored as "enc1:<base64 nonce+ciphertext>" so
// plaintext rows written before encryption existed remain readable.
const secretCipherPrefix = "enc1:"

type secretCipher struct {
	gcm cipher.AEAD
}

// LoadSecretCipher returns the encryption key for stored credentials. The key
// comes from MELOVIAN_INSTANCE_KEY (hex) or a generated file in dataDir with
// 0600 permissions. Returns nil when dataDir is empty.
func LoadSecretCipher(dataDir string) (*secretCipher, error) {
	if raw := strings.TrimSpace(os.Getenv("MELOVIAN_INSTANCE_KEY")); raw != "" {
		key, err := hex.DecodeString(raw)
		if err != nil || len(key) != 32 {
			return nil, fmt.Errorf("MELOVIAN_INSTANCE_KEY must be 64 hex chars")
		}
		return newSecretCipher(key)
	}
	if strings.TrimSpace(dataDir) == "" {
		return nil, nil
	}
	path := filepath.Join(dataDir, "instance.key")
	key, err := os.ReadFile(path) //#nosec G304 -- key file under configured data dir
	if errors.Is(err, os.ErrNotExist) {
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(dataDir, 0o750); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, key, 0o600); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("instance key file %s has %d bytes, want 32", path, len(key))
	}
	return newSecretCipher(key)
}

func newSecretCipher(key []byte) (*secretCipher, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &secretCipher{gcm: gcm}, nil
}

// encrypt returns "enc1:<base64>". Plaintext empty string stays empty so NOT
// NULL defaults and empty-field semantics do not change.
func (c *secretCipher) encrypt(plain string) (string, error) {
	if c == nil || plain == "" {
		return plain, nil
	}
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := c.gcm.Seal(nonce, nonce, []byte(plain), nil)
	return secretCipherPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// decrypt reverses encrypt. Values without the enc1 prefix are legacy
// plaintext and pass through unchanged.
func (c *secretCipher) decrypt(stored string) (string, error) {
	if c == nil || !strings.HasPrefix(stored, secretCipherPrefix) {
		return stored, nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, secretCipherPrefix))
	if err != nil {
		return "", err
	}
	if len(raw) < c.gcm.NonceSize() {
		return "", errors.New("corrupt stored secret")
	}
	plain, err := c.gcm.Open(nil, raw[:c.gcm.NonceSize()], raw[c.gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
