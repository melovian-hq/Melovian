// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

func (db *DB) GetAppSetting(key string) (string, error) {
	return db.getSetting(key)
}

func (db *DB) SetAppSetting(key, value string) error {
	return db.setSetting(key, value)
}
