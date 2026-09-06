// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lyrics

import (
	"context"
	"net/http"
)

type Provider interface {
	ID() string
	Name() string
	Fetch(ctx context.Context, client *http.Client, in FetchInput) (*Document, error)
}

type CustomProvider struct {
	IDValue   string
	NameValue string
	URL       string
}

func (p CustomProvider) ID() string {
	if p.IDValue == "" {
		return "custom"
	}
	return p.IDValue
}

func (p CustomProvider) Name() string {
	if p.NameValue == "" {
		return "Custom provider"
	}
	return p.NameValue
}

func BuiltInProviders() []ProviderInfo {
	return []ProviderInfo{
		{ID: "subsonic", Name: "Navidrome / Subsonic", Builtin: true},
		{ID: "lrclib", Name: "LRCLIB", Builtin: true},
		{ID: "lyrics-ovh", Name: "Lyrics.ovh", Builtin: true},
	}
}
