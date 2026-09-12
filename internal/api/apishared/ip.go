// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package apishared

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

func ClientIP(r *http.Request, trustProxy bool) (netip.Addr, bool) {
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
