// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestQueryIntOracle(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		fallback int
		want     int
	}{
		{name: "empty", raw: "", fallback: 7, want: 7},
		{name: "whitespace", raw: "  \t  ", fallback: 3, want: 3},
		{name: "invalid", raw: "not-a-number", fallback: 9, want: 9},
		{name: "valid", raw: "42", fallback: 0, want: 42},
		{name: "negative", raw: "-4", fallback: 8, want: -4},
		{name: "padded", raw: "  12  ", fallback: 1, want: 12},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := url.Values{}
			values.Set("limit", tc.raw)
			req := httptest.NewRequest("GET", "/?"+values.Encode(), nil)
			got := QueryInt(req, "limit", tc.fallback)
			if got != tc.want {
				t.Fatalf("QueryInt(%q, %d) = %d, want %d", tc.raw, tc.fallback, got, tc.want)
			}
		})
	}
}

func TestIsHopByHopHeaderOracle(t *testing.T) {
	hopByHop := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"TE",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
		"connection",
		"KEEP-ALIVE",
	}
	for _, key := range hopByHop {
		if !IsHopByHopHeader(key) {
			t.Fatalf("IsHopByHopHeader(%q) = false, want true", key)
		}
	}

	notHop := []string{"Content-Type", "X-Custom", "Authorization", "Accept"}
	for _, key := range notHop {
		if IsHopByHopHeader(key) {
			t.Fatalf("IsHopByHopHeader(%q) = true, want false", key)
		}
	}
}

func TestQueryIntOracleMatchesAtoi(t *testing.T) {
	inputs := []string{"", " ", "10", "0", "-1", "999999", "1.5", "abc", "\t8\n"}
	fallback := 5
	for _, raw := range inputs {
		values := url.Values{}
		values.Set("n", raw)
		req := httptest.NewRequest("GET", "/?"+values.Encode(), nil)
		got := QueryInt(req, "n", fallback)
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			if got != fallback {
				t.Fatalf("empty %q: got %d want fallback %d", raw, got, fallback)
			}
			continue
		}
		if parsed, err := strconv.Atoi(trimmed); err == nil {
			if got != parsed {
				t.Fatalf("valid %q: got %d want %d", raw, got, parsed)
			}
			continue
		}
		if got != fallback {
			t.Fatalf("invalid %q: got %d want fallback %d", raw, got, fallback)
		}
	}
}

func TestQueryIntClampedOracle(t *testing.T) {
	req := httptest.NewRequest("GET", "/?limit=2147483647", nil)
	if got := QueryIntClamped(req, "limit", 50, 1, 500); got != 500 {
		t.Fatalf("QueryIntClamped huge = %d want 500", got)
	}
	req = httptest.NewRequest("GET", "/?limit=-4", nil)
	if got := QueryIntClamped(req, "limit", 50, 1, 500); got != 1 {
		t.Fatalf("QueryIntClamped negative = %d want 1", got)
	}
}

func TestReadLimitedRejectsOversizeBody(t *testing.T) {
	body := strings.NewReader(strings.Repeat("x", 64))
	if _, err := ReadLimited(body, 16); err == nil {
		t.Fatal("expected oversize body to fail")
	}
	ok, err := ReadLimited(strings.NewReader(`{"ok":true}`), 64)
	if err != nil {
		t.Fatalf("small body: %v", err)
	}
	if string(ok) != `{"ok":true}` {
		t.Fatalf("unexpected body %q", ok)
	}
}

func TestDecodeJSONBodyEmpty(t *testing.T) {
	req := httptest.NewRequest("POST", "/", http.NoBody)
	var dst map[string]any
	err := DecodeJSONBody(req, &dst)
	if err == nil {
		t.Fatal("expected empty body to fail")
	}
	if !strings.Contains(err.Error(), "empty request body") {
		t.Fatalf("got %v", err)
	}
}

func TestDecodeJSONBodyOK(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"home"}`))
	var dst struct {
		Name string `json:"name"`
	}
	if err := DecodeJSONBody(req, &dst); err != nil {
		t.Fatalf("DecodeJSONBody: %v", err)
	}
	if dst.Name != "home" {
		t.Fatalf("name %q", dst.Name)
	}
}
