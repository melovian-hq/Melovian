// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package apishared

import (
	"net/url"
	"slices"
	"strings"
)

const wailsMobileOrigin = "https://wails.localhost"

// wailsDesktopOrigin is the origin the desktop webview sends on Linux and
// Windows, where Wails serves assets over the custom wails scheme.
const wailsDesktopOrigin = "wails://localhost"

// corsOrigins is the allowlist for credentialed cross-origin clients
// (desktop and mobile Wails apps). https://wails.localhost is always included.
var corsOrigins = []string{wailsMobileOrigin, wailsDesktopOrigin}

// ConfigureCORSOrigins replaces the extra CORS allowlist. The mobile Wails
// origin is always kept so Android and iOS can reach a remote Melovian host.
func ConfigureCORSOrigins(extra []string) {
	seen := map[string]struct{}{wailsMobileOrigin: {}, wailsDesktopOrigin: {}}
	out := []string{wailsMobileOrigin, wailsDesktopOrigin}
	for _, raw := range extra {
		origin := strings.TrimRight(strings.TrimSpace(raw), "/")
		if origin == "" {
			continue
		}
		if _, ok := seen[origin]; ok {
			continue
		}
		seen[origin] = struct{}{}
		out = append(out, origin)
	}
	corsOrigins = out
}

// WSOriginPatterns returns host patterns for the configured CORS origins,
// for use as websocket OriginPatterns. Host patterns ignore the scheme so a
// trusted origin works over both http and https. The request host is always
// authorized by the websocket library, so same-origin upgrades need no entry.
func WSOriginPatterns() []string {
	out := make([]string, 0, len(corsOrigins))
	for _, origin := range corsOrigins {
		if u, err := url.Parse(origin); err == nil && u.Host != "" {
			// Non-http(s) schemes stay scheme-bound so wails://localhost does
			// not widen into a bare localhost pattern that would allow any
			// local web page to open the control socket.
			if u.Scheme != "http" && u.Scheme != "https" {
				out = append(out, u.Scheme+"://"+u.Host)
				continue
			}
			out = append(out, u.Host)
			continue
		}
		out = append(out, origin)
	}
	return out
}

func CORSOriginAllowed(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return false
	}
	return slices.Contains(corsOrigins, origin)
}
