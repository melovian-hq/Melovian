// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"melovian/internal/api/apishared"
	"melovian/internal/httputil"
	"net/http"
	"net/netip"
)

func IPAllowlistMiddleware(allowed []netip.Prefix, trustProxy bool, next http.Handler) http.Handler {
	if len(allowed) == 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addr, ok := apishared.ClientIP(r, trustProxy)
		if !ok || !ipAllowed(addr, allowed) {
			httputil.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func ipAllowed(addr netip.Addr, allowed []netip.Prefix) bool {
	for _, prefix := range allowed {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
