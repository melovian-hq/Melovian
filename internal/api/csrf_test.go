// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func csrfProbe(loopbackOnly bool) http.Handler {
	return CSRFMiddleware(loopbackOnly, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
}

func csrfStatus(h http.Handler, method, host, origin, fetchSite string) int {
	r := httptest.NewRequest(method, "http://"+host+"/api/test", nil)
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	if fetchSite != "" {
		r.Header.Set("Sec-Fetch-Site", fetchSite)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Code
}

func TestCSRFMiddleware(t *testing.T) {
	h := csrfProbe(false)

	// Safe methods always pass.
	if code := csrfStatus(h, http.MethodGet, "evil.example", "", "cross-site"); code != http.StatusNoContent {
		t.Fatalf("GET should pass, got %d", code)
	}
	// Same-origin mutation passes.
	if code := csrfStatus(h, http.MethodPost, "melovian.local", "https://melovian.local", ""); code != http.StatusNoContent {
		t.Fatalf("same-origin POST should pass, got %d", code)
	}
	// Allowlisted Wails origins pass.
	if code := csrfStatus(h, http.MethodPost, "melovian.local", "https://wails.localhost", ""); code != http.StatusNoContent {
		t.Fatalf("wails origin POST should pass, got %d", code)
	}
	// Cross-site origin on a mutation is rejected.
	if code := csrfStatus(h, http.MethodPost, "melovian.local", "https://evil.example", "cross-site"); code != http.StatusForbidden {
		t.Fatalf("cross-origin POST should be 403, got %d", code)
	}
	// Origin-stripped cross-site sends are rejected via Sec-Fetch-Site.
	if code := csrfStatus(h, http.MethodPost, "melovian.local", "", "cross-site"); code != http.StatusForbidden {
		t.Fatalf("cross-site fetch POST should be 403, got %d", code)
	}
	// Non-browser clients with no origin markers pass.
	if code := csrfStatus(h, http.MethodPost, "melovian.local", "", ""); code != http.StatusNoContent {
		t.Fatalf("headerless POST should pass, got %d", code)
	}
	// Opaque origins are rejected on mutations.
	if code := csrfStatus(h, http.MethodPost, "melovian.local", "null", "cross-site"); code != http.StatusForbidden {
		t.Fatalf("null origin POST should be 403, got %d", code)
	}
}

func TestCSRFMiddlewareLoopbackHostPinning(t *testing.T) {
	h := csrfProbe(true)

	for _, host := range []string{"127.0.0.1:17337", "localhost:17337", "[::1]:17337"} {
		if code := csrfStatus(h, http.MethodGet, host, "", "same-origin"); code != http.StatusNoContent {
			t.Fatalf("loopback host %s should pass, got %d", host, code)
		}
	}
	// A rebound domain sends its own hostname on every method. Browser
	// requests always carry fetch metadata, which is what keys the check.
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		if code := csrfStatus(h, method, "rebind.attacker.example", "", "same-origin"); code != http.StatusForbidden {
			t.Fatalf("rebind host %s should be 403, got %d", method, code)
		}
	}
	// Non-browser clients can set any Host anyway, so without fetch
	// metadata the pinning stays out of their way.
	if code := csrfStatus(h, http.MethodGet, "rebind.attacker.example", "", ""); code != http.StatusNoContent {
		t.Fatalf("host pinning should only gate browser requests, got %d", code)
	}
}

func TestListenAddrLoopbackOnly(t *testing.T) {
	for _, tc := range []struct {
		addr string
		want bool
	}{
		{"127.0.0.1:17337", true},
		{"localhost:17337", true},
		{"[::1]:17337", true},
		{"0.0.0.0:8080", false},
		{":8080", false},
		{"192.168.1.5:8080", false},
		{"", false},
	} {
		if got := ListenAddrLoopbackOnly(tc.addr); got != tc.want {
			t.Errorf("ListenAddrLoopbackOnly(%q) = %v, want %v", tc.addr, got, tc.want)
		}
	}
}
