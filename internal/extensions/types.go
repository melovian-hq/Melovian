// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"encoding/json"

	"melovian/internal/brand"
)

// ManifestName is the extension manifest file name. It follows brand.Slug.
var ManifestName = brand.ManifestName()

type Manifest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Author      string `json:"author,omitempty"`
	// Icon is an optional path relative to the extension folder (for example assets/icon.png).
	Icon string `json:"icon,omitempty"`
	// Image is an optional larger artwork path relative to the extension folder.
	Image string `json:"image,omitempty"`
	// AppTheme is an optional named app chrome theme (for example "neon").
	AppTheme string `json:"appTheme,omitempty"`
	// Styles are optional stylesheet paths relative to the extension folder.
	Styles      []string     `json:"styles,omitempty"`
	Script      string       `json:"script,omitempty"`
	TrackRules  []TrackRule  `json:"trackRules,omitempty"`
	PlayerHooks []PlayerHook `json:"playerHooks,omitempty"`
}

type TrackMatch struct {
	GenreContains  string `json:"genreContains,omitempty"`
	ArtistContains string `json:"artistContains,omitempty"`
	AlbumContains  string `json:"albumContains,omitempty"`
	TitleContains  string `json:"titleContains,omitempty"`
	TitleRegex     string `json:"titleRegex,omitempty"`
	TagEquals      string `json:"tagEquals,omitempty"`
	MinRating      int    `json:"minRating,omitempty"`
	IsLocal        *bool  `json:"isLocal,omitempty"`
}

type TrackDecoration struct {
	ProgressColor       string `json:"progressColor,omitempty"`
	ProgressGradient    string `json:"progressGradient,omitempty"`
	ProgressThumbUrl    string `json:"progressThumbUrl,omitempty"`
	ProgressParticleUrl string `json:"progressParticleUrl,omitempty"`
	Icon                string `json:"icon,omitempty"`
	IconUrl             string `json:"iconUrl,omitempty"`
	TitlePrefix         string `json:"titlePrefix,omitempty"`
	CoverOverlayIcon    string `json:"coverOverlayIcon,omitempty"`
	PlayerTheme         string `json:"playerTheme,omitempty"`
}

type TrackRule struct {
	Match      TrackMatch      `json:"match"`
	Decoration TrackDecoration `json:"decoration"`
}

type PlayerHook struct {
	When  string            `json:"when"`
	Style map[string]string `json:"style,omitempty"`
}

type Entry struct {
	Manifest Manifest `json:"manifest"`
	Dir      string   `json:"dir"`
	Enabled  bool     `json:"enabled"`
	// InstalledAt is the extension folder mtime as RFC3339 UTC.
	InstalledAt string `json:"installedAt,omitempty"`
}

func ParseManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}
