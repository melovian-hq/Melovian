// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"context"

	"melovian/internal/navidrome"
	"melovian/internal/store"
)

// NavidromeSourceID is the bundled Navidrome source extension.
const NavidromeSourceID = "navidrome"

// navidromeSource is the bundled Navidrome source. REST traffic uses
// the same Subsonic passthrough; the source adds native capabilities
// (smart playlists, SSE events) on top.
type navidromeSource struct{ subsonicSource }

// Navidrome returns the bundled Navidrome source.
func Navidrome() Source { return navidromeSource{} }

func (navidromeSource) ID() string          { return NavidromeSourceID }
func (navidromeSource) DisplayName() string { return "Navidrome" }

func (navidromeSource) Caps() Capabilities {
	return Capabilities{
		SmartPlaylists: true,
		Events:         true,
		EventsPath:     "/api/events",
		EventsAuthMode: "query-jwt",
	}
}

// NativeClient builds a Navidrome native API client for the instance.
// The returned client authenticates lazily and holds its JWT internally.
func NativeClient(inst store.SourceInstance) *navidrome.Client {
	return navidrome.NewClient(inst.ServerURL, inst.Username, inst.Password)
}

// StreamEvents connects to the Navidrome SSE endpoint and emits each
// broker event. The caller owns reconnects.
func (navidromeSource) StreamEvents(ctx context.Context, inst store.SourceInstance, emit EventEmit) error {
	return NativeClient(inst).StreamEvents(ctx, emit)
}

// DefaultRegistry returns a registry with the bundled sources
// registered. enabledFn should report extension enable state.
func DefaultRegistry(enabledFn func(sourceID string) bool) *Registry {
	reg := New(enabledFn)
	reg.Register(Subsonic())
	reg.Register(Navidrome())
	return reg
}
