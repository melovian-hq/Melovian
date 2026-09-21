// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"melovian/internal/api/apishared"
	"melovian/internal/httputil"
)

// CSRFMiddleware fences the API against cross-site sends and DNS
// rebinding. Session cookies can be SameSite=None for the trusted Wails
// origins, and some deployments run with auth disabled entirely, so
// cookie scope alone is not a boundary. The check is:
//
//   - mutating methods must come from a same-host origin or the
//     configured CORS allowlist. Requests with no Origin are fine for
//     non-browser clients, but a cross-site Sec-Fetch-Site is rejected.
//   - when the listener is loopback-only, browser requests must carry a
//     loopback Host. A rebound domain would send its own hostname. The
//     check keys on fetch metadata because browsers always send it while
//     non-browser clients can set any Host anyway.
func CSRFMiddleware(loopbackOnly bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if loopbackOnly && r.Header.Get("Sec-Fetch-Site") != "" && !isLoopbackHost(r.Host) {
			httputil.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
			return
		}
		if mutatingMethod(r.Method) && !mutationOriginAllowed(r) {
			httputil.WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func mutatingMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	return true
}

func mutationOriginAllowed(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		// Non-browser clients send neither header. Cross-site sends
		// always carry Sec-Fetch-Site, so its absence here is safe.
		return !strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site")
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	if strings.EqualFold(u.Host, r.Host) {
		return true
	}
	return apishared.CORSOriginAllowed(origin)
}

func isLoopbackHost(host string) bool {
	name := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		name = h
	}
	name = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
	if name == "localhost" || name == "localhost.localdomain" {
		return true
	}
	if ip := net.ParseIP(strings.Trim(name, "[]")); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

// ListenAddrLoopbackOnly reports whether addr binds only loopback
// interfaces, where Host pinning can safely apply.
func ListenAddrLoopbackOnly(addr string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return false
	}
	return isLoopbackHost(host)
}
