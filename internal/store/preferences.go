// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"database/sql"
	"fmt"
)

type PreferencesStore struct {
	db *DB
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

func (s *PreferencesStore) Get(userID, key string) (string, error) {
	var value string
	err := s.db.queryRow(
		`SELECT value FROM user_preferences WHERE user_id = ? AND key = ?`,
		userID, key,
	).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (s *PreferencesStore) Set(userID, key, value string) error {
	if userID == "" || key == "" {
		return fmt.Errorf("user id and key are required")
	}
	now := nowUnix()
	_, err := s.db.exec(
		`INSERT INTO user_preferences (user_id, key, value, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(user_id, key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		userID, key, value, now,
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
