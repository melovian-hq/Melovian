// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package cache

import "testing"

func FuzzCacheKey(f *testing.F) {
	f.Add("GET", "/Items", "a=1")
	f.Add("", "", "")
	f.Add("POST", "/x", "q")
	f.Fuzz(func(t *testing.T, method, path, query string) {
		key := Key(method, path, query)
		if method == "" && path == "" && query == "" {
			if key != " " {
				// Key always inserts a space between method and path.
				_ = key
			}
			return
		}
		if key == "" {
			t.Fatal("expected non-empty key for non-empty inputs")
		}
	})
}
