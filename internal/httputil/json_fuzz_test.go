// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func FuzzQueryInt(f *testing.F) {
	f.Add("limit", "10", 5)
	f.Add("offset", "", 0)
	f.Add("count", "not-a-number", 3)
	f.Add("size", "-4", 8)

	f.Fuzz(func(t *testing.T, key, raw string, fallback int) {
		values := url.Values{}
		values.Set(key, raw)
		req := httptest.NewRequest("GET", "/?"+values.Encode(), nil)
		got := QueryInt(req, key, fallback)
		trimmed := strings.TrimSpace(raw)

		if trimmed == "" {
			if got != fallback {
				t.Fatalf("empty value should return fallback %d, got %d", fallback, got)
			}
			return
		}

		if parsed, err := strconv.Atoi(trimmed); err == nil {
			if got != parsed {
				t.Fatalf("valid int %q should parse to %d, got %d", raw, parsed, got)
			}
			return
		}

		if got != fallback {
			t.Fatalf("invalid value %q should return fallback %d, got %d", raw, fallback, got)
		}
	})
}

func FuzzIsHopByHopHeader(f *testing.F) {
	f.Add("Connection")
	f.Add("content-type")
	f.Add("X-Custom")

	f.Fuzz(func(t *testing.T, key string) {
		_ = IsHopByHopHeader(key)
	})
}
