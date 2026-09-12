// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package apishared

import (
	"strings"

	"melovian/internal/appconfig"
)

// PublicBaseURL returns the externally reachable base URL for this server,
// used in share links and DLNA announcements.
func PublicBaseURL(cfg appconfig.Config) string {
	if base := strings.TrimSpace(cfg.PublicURL); base != "" {
		return NormalizePublicBaseURL(base)
	}
	return NormalizePublicBaseURL("http://" + cfg.ListenAddr)
}

func NormalizePublicBaseURL(raw string) string {
	base := strings.TrimRight(strings.TrimSpace(raw), "/")
	if base == "" {
		return ""
	}
	base = strings.Replace(base, "://0.0.0.0", "://127.0.0.1", 1)
	base = strings.Replace(base, "://[::]", "://127.0.0.1", 1)
	if after, ok := strings.CutPrefix(base, "0.0.0.0:"); ok {
		base = "http://127.0.0.1:" + after
	}
	if after, ok := strings.CutPrefix(base, "[::]:"); ok {
		base = "http://127.0.0.1:" + after
	}
	return base
}
