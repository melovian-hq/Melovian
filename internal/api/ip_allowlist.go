// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"melovian/internal/httputil"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

func IPAllowlistMiddleware(allowed []netip.Prefix, trustProxy bool, next http.Handler) http.Handler {
	if len(allowed) == 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addr, ok := clientIP(r, trustProxy)
		if !ok || !ipAllowed(addr, allowed) {
			httputil.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request, trustProxy bool) (netip.Addr, bool) {
	if trustProxy {
		if raw := strings.TrimSpace(r.Header.Get("X-Real-IP")); raw != "" {
			if addr, err := netip.ParseAddr(raw); err == nil {
				return addr, true
			}
		}
		if raw := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); raw != "" {
			first := strings.TrimSpace(strings.Split(raw, ",")[0])
			if addr, err := netip.ParseAddr(first); err == nil {
				return addr, true
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		addr, parseErr := netip.ParseAddr(r.RemoteAddr)
		if parseErr != nil {
			return netip.Addr{}, false
		}
		return addr, true
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, false
	}
	return addr, true
}

func ipAllowed(addr netip.Addr, allowed []netip.Prefix) bool {
	for _, prefix := range allowed {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
