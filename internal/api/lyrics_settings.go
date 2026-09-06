// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

	"melovian/internal/lyrics"
	"melovian/internal/osutil"
	"melovian/internal/store"
)

type LyricsProviderSetting struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Enabled bool   `json:"enabled"`
	Custom  bool   `json:"custom,omitempty"`
	URL     string `json:"url,omitempty"`
}

type LyricsSettings struct {
	StorageDir string                  `json:"storageDir"`
	AutoFetch  bool                    `json:"autoFetch"`
	Providers  []LyricsProviderSetting `json:"providers"`
	// WhisperURL points at a whisper.cpp-compatible server (for example a
	// transcriptasm host) used to generate synced lyrics from track audio.
	WhisperURL string `json:"whisperUrl,omitempty"`
}

func defaultLyricsSettings(dataDir string) LyricsSettings {
	_ = dataDir
	return LyricsSettings{
		StorageDir: "",
		AutoFetch:  false,
		Providers: []LyricsProviderSetting{
			{ID: "subsonic", Name: "Navidrome / Subsonic", Enabled: true},
			{ID: "lrclib", Name: "LRCLIB", Enabled: true},
			{ID: "lyrics-ovh", Name: "Lyrics.ovh", Enabled: true},
		},
	}
}

func defaultLyricsStorageDir(dataDir string) string {
	return filepath.Join(dataDir, "lyrics")
}

func mergeLyricsSettings(raw json.RawMessage, dataDir string) (LyricsSettings, error) {
	settings := defaultLyricsSettings(dataDir)
	if len(raw) == 0 {
		return settings, nil
	}
	if err := json.Unmarshal(raw, &settings); err != nil {
		return LyricsSettings{}, err
	}
	if settings.Providers == nil {
		settings.Providers = defaultLyricsSettings(dataDir).Providers
	}
	seen := map[string]bool{}
	normalized := make([]LyricsProviderSetting, 0, len(settings.Providers))
	for _, provider := range settings.Providers {
		id := strings.TrimSpace(provider.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		provider.ID = id
		if provider.Custom && strings.TrimSpace(provider.URL) == "" {
			continue
		}
		normalized = append(normalized, provider)
	}
	for _, builtin := range defaultLyricsSettings(dataDir).Providers {
		if !seen[builtin.ID] {
			normalized = append(normalized, builtin)
		}
	}
	settings.Providers = normalized
	settings.StorageDir = strings.TrimSpace(settings.StorageDir)
	settings.WhisperURL = strings.TrimSpace(settings.WhisperURL)
	return settings, nil
}

func (s *Server) loadLyricsSettings(userID string) (LyricsSettings, error) {
	raw, err := s.preferences.Get(userID, store.PrefKeyLyricsSettings)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return defaultLyricsSettings(s.cfg.DataDir), nil
		}
		return LyricsSettings{}, err
	}
	return mergeLyricsSettings(json.RawMessage(raw), s.cfg.DataDir)
}

func (s *Server) lyricsRoot(settings LyricsSettings) string {
	root := strings.TrimSpace(settings.StorageDir)
	if root == "" {
		return defaultLyricsStorageDir(s.cfg.DataDir)
	}
	if !filepath.IsAbs(root) {
		root = filepath.Join(s.cfg.DataDir, root)
		root = filepath.Clean(root)
		if err := osutil.PathEscapesRoot(s.cfg.DataDir, root); err != nil {
			return defaultLyricsStorageDir(s.cfg.DataDir)
		}
		return root
	}
	return filepath.Clean(root)
}

func (s *Server) enabledLyricsProviders(settings LyricsSettings, subsonicFn lyrics.SubsonicFetcher) []lyrics.Provider {
	providers := make([]lyrics.Provider, 0, len(settings.Providers))
	for _, cfg := range settings.Providers {
		if !cfg.Enabled {
			continue
		}
		if cfg.Custom {
			providers = append(providers, lyrics.CustomProvider{
				IDValue:   cfg.ID,
				NameValue: cfg.Name,
				URL:       cfg.URL,
			})
			continue
		}
		switch cfg.ID {
		case "subsonic":
			providers = append(providers, lyrics.SubsonicProvider{FetchFn: subsonicFn})
		case "lrclib":
			providers = append(providers, lyrics.LRCLIBProvider{})
		case "lyrics-ovh":
			providers = append(providers, lyrics.LyricsOvhProvider{})
		}
	}
	return providers
}

func lyricsDocumentToMap(doc *lyrics.Document) map[string]any {
	if doc == nil {
		return nil
	}
	lines := make([]map[string]any, 0, len(doc.Lines))
	for _, line := range doc.Lines {
		entry := map[string]any{"text": line.Text}
		if line.StartMs != nil {
			entry["startMs"] = *line.StartMs
		}
		lines = append(lines, entry)
	}
	return map[string]any{
		"source":   doc.Source,
		"artist":   doc.Artist,
		"title":    doc.Title,
		"synced":   doc.Synced,
		"offsetMs": doc.OffsetMs,
		"rawValue": doc.RawValue,
		"lines":    lines,
	}
}
