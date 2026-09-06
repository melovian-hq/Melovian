// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func BenchmarkResponseCacheSetEvict(b *testing.B) {
	c := NewResponseCache()
	body := make([]byte, 32<<10)
	now := time.Now()

	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		c.Set(fmt.Sprintf("GET /bench/%d", i), Entry{
			StatusCode: http.StatusOK,
			Body:       body,
			ExpiresAt:  now.Add(time.Hour),
		})
	}
}

func BenchmarkResponseCacheStats(b *testing.B) {
	c := NewResponseCache()
	now := time.Now()
	for i := range 128 {
		c.Set(fmt.Sprintf("GET /s/%d", i), Entry{
			StatusCode: http.StatusOK,
			Body:       []byte("x"),
			ExpiresAt:  now.Add(time.Hour),
		})
	}

	b.ReportAllocs()
	for b.Loop() {
		_, _ = c.Stats()
	}
}
