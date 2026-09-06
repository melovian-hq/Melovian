// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"strings"
)

const wailsMobileOrigin = "https://wails.localhost"

// corsOrigins is the allowlist for credentialed cross-origin clients
// (mobile Wails apps). https://wails.localhost is always included.
var corsOrigins = []string{wailsMobileOrigin}

// ConfigureCORSOrigins replaces the extra CORS allowlist. The mobile Wails
// origin is always kept so Android and iOS can reach a remote Melovian host.
func ConfigureCORSOrigins(extra []string) {
	seen := map[string]struct{}{wailsMobileOrigin: {}}
	out := []string{wailsMobileOrigin}
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

func corsOriginAllowed(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return false
	}
	for _, allowed := range corsOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if corsOriginAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
			w.Header().Set(
				"Access-Control-Allow-Headers",
				"Accept, Authorization, Content-Type, Range, X-Instance-Id, X-Device-Id, X-Request-Id, X-Melovian-Client-Version, X-Melovian-API-Version, X-Melovian-Capabilities",
			)
			w.Header().Set(
				"Access-Control-Allow-Methods",
				"GET, POST, PUT, PATCH, DELETE, OPTIONS",
			)
			w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges, X-Request-Id")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
