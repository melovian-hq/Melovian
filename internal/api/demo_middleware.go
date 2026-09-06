// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"melovian/internal/httputil"
)

func DemoReadOnlyMiddleware(demo bool, next http.Handler) http.Handler {
	if !demo {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
		default:
			httputil.WriteJSON(w, http.StatusForbidden, map[string]any{
				"error": "demo mode is read-only",
			})
		}
	})
}
