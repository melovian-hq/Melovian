// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package apishared

import (
	"net/http"
	"strings"
	"time"
)

func IsSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if hdr := r.Header.Get("X-Forwarded-Proto"); strings.EqualFold(hdr, "https") {
		return true
	}
	return false
}

func SetHTTPOnlyCookie(w http.ResponseWriter, r *http.Request, name, value, path string, maxAge int, expires time.Time) {
	sameSite := http.SameSiteLaxMode
	secure := IsSecureRequest(r)
	// Cross-origin mobile clients on HTTPS need SameSite=None so credentialed
	// fetch() from https://wails.localhost includes the session cookie.
	// SameSite=None requires Secure, so this only applies on HTTPS responses.
	// HTTP overlay hosts (Tailscale/Netbird) with auth disabled need no cookie.
	if CORSOriginAllowed(r.Header.Get("Origin")) && IsSecureRequest(r) {
		sameSite = http.SameSiteNoneMode
		secure = true
	}
	cookie := &http.Cookie{ //#nosec G124 -- Secure follows request scheme or SameSite=None rules
		Name:     name,
		Value:    value,
		Path:     path,
		HttpOnly: true,
		SameSite: sameSite,
		Secure:   secure,
		MaxAge:   maxAge,
	}
	if !expires.IsZero() {
		cookie.Expires = expires
	}
	http.SetCookie(w, cookie)
}

func ClearHTTPOnlyCookie(w http.ResponseWriter, r *http.Request, name, path string) {
	sameSite := http.SameSiteLaxMode
	secure := IsSecureRequest(r)
	if CORSOriginAllowed(r.Header.Get("Origin")) && IsSecureRequest(r) {
		sameSite = http.SameSiteNoneMode
		secure = true
	}
	http.SetCookie(w, &http.Cookie{ //#nosec G124 -- clearing cookie with same security attributes as set
		Name:     name,
		Value:    "",
		Path:     path,
		HttpOnly: true,
		SameSite: sameSite,
		Secure:   secure,
		MaxAge:   -1,
	})
}
