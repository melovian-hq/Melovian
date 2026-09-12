// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"melovian/internal/api/apishared"
	"net/http"
)

// ConfigureCORSOrigins replaces the extra CORS allowlist. The mobile Wails
// origin is always kept so Android and iOS can reach a remote Melovian host.
func ConfigureCORSOrigins(extra []string) {
	apishared.ConfigureCORSOrigins(extra)
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if apishared.CORSOriginAllowed(origin) {
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
