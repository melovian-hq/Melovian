// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"database/sql"
	"errors"

	"melovian/internal/appconfig"
)

const PrefKeySourceViewMode = "source_view_mode"
const PrefKeyMultiLocalLibrary = "multi_local_library"

const (
	SourceViewSubsonic = "subsonic"
	SourceViewLocal    = "local"
	SourceViewUnified  = "unified"
)

func normalizeSourceViewMode(value string) string {
	switch value {
	case SourceViewLocal, SourceViewUnified:
		return value
	default:
		return SourceViewSubsonic
	}
}

func (s *PreferencesStore) GetSourceViewMode(userID string) (string, error) {
	if userID == "" {
		value, err := s.db.getSetting(appconfig.SettingSourceViewMode)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return SourceViewSubsonic, nil
			}
			return "", err
		}
		return normalizeSourceViewMode(value), nil
	}
	value, err := s.Get(userID, PrefKeySourceViewMode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SourceViewSubsonic, nil
		}
		return "", err
	}
	return normalizeSourceViewMode(value), nil
}

func (s *PreferencesStore) SetSourceViewMode(userID, mode string) error {
	switch mode {
	case SourceViewSubsonic, SourceViewLocal, SourceViewUnified:
	default:
		return errors.New("invalid source view mode")
	}
	if userID == "" {
		return s.db.setSetting(appconfig.SettingSourceViewMode, mode)
	}
	return s.Set(userID, PrefKeySourceViewMode, mode)
}

func IsUnifiedSourceView(mode string) bool {
	return mode == SourceViewUnified
}

func IsLocalSourceView(mode string) bool {
	return mode == SourceViewLocal
}

func IsSubsonicSourceView(mode string) bool {
	return mode == SourceViewSubsonic
}

func (s *PreferencesStore) GetMultiLocalLibrary(userID string) (bool, error) {
	if userID == "" {
		value, err := s.db.getSetting(appconfig.SettingMultiLocalLibrary)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return false, nil
			}
			return false, err
		}
		return value == "true", nil
	}
	value, err := s.Get(userID, PrefKeyMultiLocalLibrary)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return value == "true", nil
}

func (s *PreferencesStore) SetMultiLocalLibrary(userID string, enabled bool) error {
	value := "false"
	if enabled {
		value = "true"
	}
	if userID == "" {
		return s.db.setSetting(appconfig.SettingMultiLocalLibrary, value)
	}
	return s.Set(userID, PrefKeyMultiLocalLibrary, value)
}
