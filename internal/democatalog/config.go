// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package democatalog

import (
	"strings"

	"melovian/internal/appconfig"
)

// UsesFakeCatalog reports whether demo mode should serve the built-in catalog.
func UsesFakeCatalog(cfg appconfig.Config) bool {
	if !cfg.DemoModeEffective() {
		return false
	}
	server := strings.TrimSpace(cfg.LegacyServer)
	if server == "" {
		return true
	}
	return IsFakeURL(server)
}

// ApplyDefaults fills LegacyServer credentials with the fake catalog when demo
// mode has no real Subsonic upstream configured.
func ApplyDefaults(cfg *appconfig.Config) {
	if cfg == nil || !cfg.DemoModeEffective() {
		return
	}
	if !UsesFakeCatalog(*cfg) && !IsFakeURL(cfg.LegacyServer) {
		return
	}
	cfg.LegacyServer = ServerURL
	if strings.TrimSpace(cfg.LegacyUser) == "" {
		cfg.LegacyUser = Username
	}
	if strings.TrimSpace(cfg.LegacyPass) == "" {
		cfg.LegacyPass = Password
	}
}
