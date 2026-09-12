// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

type PreferencesStore struct {
	db         *DB
	cipher     *secretCipher
	cipherOnce sync.Once
	cipherErr  error
}

func NewPreferencesStore(db *DB) *PreferencesStore {
	return &PreferencesStore{db: db}
}

const PrefKeyMusicEq = "music_eq"
const PrefKeyConnectionSettings = "connection_settings"
const PrefKeyCacheSettings = "cache_settings"
const PrefKeyLyricsSettings = "lyrics_settings"
const PrefKeyVideoSettings = "video_settings"
const PrefKeyRockskySettings = "rocksky_settings"
const PrefKeyLastFMSettings = "lastfm_settings"
const PrefKeyListenBrainzSettings = "listenbrainz_settings"
const PrefKeyUpdateSettings = "update_settings"

// secretPrefFields lists the JSON fields inside each preference blob that
// hold third-party credentials. Those fields are encrypted at rest when a
// secret cipher is configured while the rest of the blob stays readable.
var secretPrefFields = map[string][]string{
	PrefKeyLastFMSettings:       {"apiKey", "apiSecret", "sessionKey"},
	PrefKeyRockskySettings:      {"token"},
	PrefKeyListenBrainzSettings: {"token"},
}

// SetCipher enables at-rest encryption of the credential fields listed in
// secretPrefFields. Blobs written before encryption stay readable and are
// upgraded by migrateSecrets or the next Set.
func (s *PreferencesStore) SetCipher(c *secretCipher) {
	s.cipher = c
}

// EnsureSecretCipher lazily loads the instance secret cipher for dataDir and
// runs the plaintext upgrade once. Safe to call on every request.
func (s *PreferencesStore) EnsureSecretCipher(dataDir string) error {
	s.cipherOnce.Do(func() {
		if s.cipher == nil {
			cipher, err := LoadSecretCipher(dataDir)
			if err != nil {
				s.cipherErr = err
				return
			}
			s.cipher = cipher
		}
		if err := s.migrateSecrets(); err != nil {
			s.cipherErr = err
		}
	})
	return s.cipherErr
}

func (s *PreferencesStore) Get(userID, key string) (string, error) {
	var value string
	err := s.db.queryRow(
		`SELECT value FROM user_preferences WHERE user_id = ? AND key = ?`,
		userID, key,
	).Scan(&value)
	if err != nil {
		return "", err
	}
	return s.decryptValue(key, value)
}

func (s *PreferencesStore) Set(userID, key, value string) error {
	if userID == "" || key == "" {
		return fmt.Errorf("user id and key are required")
	}
	sealed, err := s.encryptValue(key, value)
	if err != nil {
		return err
	}
	now := nowUnix()
	_, err = s.db.exec(
		`INSERT INTO user_preferences (user_id, key, value, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(user_id, key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		userID, key, sealed, now,
	)
	return err
}

func (s *PreferencesStore) Delete(userID, key string) error {
	res, err := s.db.exec(`DELETE FROM user_preferences WHERE user_id = ? AND key = ?`, userID, key)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// decryptValue returns raw with the credential fields for key decrypted.
// Legacy plaintext fields pass through unchanged.
func (s *PreferencesStore) decryptValue(key, raw string) (string, error) {
	return s.rewriteSecretFields(key, raw, s.cipher.decrypt)
}

// encryptValue returns raw with the credential fields for key encrypted.
func (s *PreferencesStore) encryptValue(key, raw string) (string, error) {
	return s.rewriteSecretFields(key, raw, s.cipher.encrypt)
}

// rewriteSecretFields applies fn to every credential field of the JSON blob
// stored under key. Non-JSON values and missing or non-string fields pass
// through untouched.
func (s *PreferencesStore) rewriteSecretFields(key, raw string, fn func(string) (string, error)) (string, error) {
	fields, ok := secretPrefFields[key]
	if !ok {
		return raw, nil
	}
	var blob map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &blob); err != nil {
		return raw, nil
	}
	changed := false
	for _, field := range fields {
		var value string
		if err := json.Unmarshal(blob[field], &value); err != nil {
			continue
		}
		next, err := fn(value)
		if err != nil {
			return "", err
		}
		if next == value {
			continue
		}
		data, err := json.Marshal(next)
		if err != nil {
			return "", err
		}
		blob[field] = data
		changed = true
	}
	if !changed {
		return raw, nil
	}
	data, err := json.Marshal(blob)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// hasPlaintextSecrets reports whether any credential field in the stored blob
// is a non-empty string without the enc1 prefix.
func (s *PreferencesStore) hasPlaintextSecrets(key, raw string) bool {
	var blob map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &blob); err != nil {
		return false
	}
	for _, field := range secretPrefFields[key] {
		var value string
		if err := json.Unmarshal(blob[field], &value); err != nil {
			continue
		}
		if value != "" && !strings.HasPrefix(value, secretCipherPrefix) {
			return true
		}
	}
	return false
}

// migrateSecrets rewrites stored blobs whose credential fields are still
// plaintext. Safe to call at startup and idempotent.
func (s *PreferencesStore) migrateSecrets() error {
	if s.cipher == nil {
		return nil
	}
	for key := range secretPrefFields {
		if err := s.migrateSecretKey(key); err != nil {
			return err
		}
	}
	return nil
}

func (s *PreferencesStore) migrateSecretKey(key string) error {
	rows, err := s.db.query(
		`SELECT user_id, value FROM user_preferences WHERE key = ?`, key,
	)
	if err != nil {
		return err
	}
	type rowPair struct{ userID, value string }
	var pending []rowPair
	for rows.Next() {
		var p rowPair
		if err := rows.Scan(&p.userID, &p.value); err != nil {
			_ = rows.Close()
			return err
		}
		pending = append(pending, p)
	}
	_ = rows.Close()
	for _, p := range pending {
		if !s.hasPlaintextSecrets(key, p.value) {
			continue
		}
		plain, err := s.decryptValue(key, p.value)
		if err != nil {
			return err
		}
		sealed, err := s.encryptValue(key, plain)
		if err != nil {
			return err
		}
		if _, err := s.db.exec(
			`UPDATE user_preferences SET value = ? WHERE user_id = ? AND key = ?`,
			sealed, p.userID, key,
		); err != nil {
			return err
		}
	}
	return nil
}
