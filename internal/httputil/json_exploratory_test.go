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

func TestQueryIntExploratory(t *testing.T) {
	inputs := []string{
		"",
		" ",
		"\t",
		"0",
		"1",
		"-1",
		"2147483647",
		"-2147483648",
		"999999999999999999999",
		"1e3",
		"0x10",
		"08",
		"+5",
		"5.0",
		"5,000",
		"nan",
		"inf",
		"null",
		"true",
		" 42 ",
		"\n7\r",
		strings.Repeat("9", 40),
		strings.Repeat(" ", 20),
		"😀",
		"١٢٣",
		"1\x00２",
	}
	fallbacks := []int{0, -1, 1, 42, 100}

	for _, raw := range inputs {
		for _, fallback := range fallbacks {
			values := url.Values{}
			values.Set("q", raw)
			req := httptest.NewRequest("GET", "/?"+values.Encode(), nil)
			got := QueryInt(req, "q", fallback)

			trimmed := strings.TrimSpace(raw)
			if trimmed == "" {
				if got != fallback {
					t.Fatalf("empty input %q fallback %d: got %d", raw, fallback, got)
				}
				continue
			}
			if parsed, err := strconv.Atoi(trimmed); err == nil {
				if got != parsed {
					t.Fatalf("valid input %q: got %d want %d", raw, got, parsed)
				}
				continue
			}
			if got != fallback {
				t.Fatalf("invalid input %q fallback %d: got %d", raw, fallback, got)
			}
		}
	}
}
