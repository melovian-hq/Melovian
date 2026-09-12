// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// rateLimiter is a fixed-window per-key limiter for credential endpoints and
// public write endpoints. Counters live in process memory, which matches the
// single-process deployment model. Keys are caller chosen (client IP, share
// token, username) so one abusive client cannot exhaust a global budget.
type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]rateLimitEntry
	max     int
	window  time.Duration
	now     func() time.Time
}

type rateLimitEntry struct {
	count   int
	resetAt time.Time
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		entries: make(map[string]rateLimitEntry),
		max:     max,
		window:  window,
		now:     time.Now,
	}
}

// blocked reports whether any key already exhausted its window budget.
// Callers pass several keys (client IP, username, share token) so spoofing
// one dimension does not dodge the limit.
func (l *rateLimiter) blocked(keys ...string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for _, key := range keys {
		e, ok := l.entries[key]
		if ok && now.Before(e.resetAt) && e.count >= l.max {
			return true
		}
	}
	return false
}

// record counts one attempt for each key. Expired entries are evicted lazily
// when the map grows so an attacker cannot grow memory without bound.
func (l *rateLimiter) record(keys ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for _, key := range keys {
		e, ok := l.entries[key]
		if !ok || !now.Before(e.resetAt) {
			e = rateLimitEntry{resetAt: now.Add(l.window)}
		}
		e.count++
		l.entries[key] = e
	}
	if len(l.entries) > 8192 {
		for k, v := range l.entries {
			if !now.Before(v.resetAt) {
				delete(l.entries, k)
			}
		}
	}
}

// reset clears the counter for each key after a successful attempt.
func (l *rateLimiter) reset(keys ...string) {
	l.mu.Lock()
	for _, key := range keys {
		delete(l.entries, key)
	}
	l.mu.Unlock()
}

// retryAfterSeconds returns the longest remaining lockout for the keys.
func (l *rateLimiter) retryAfterSeconds(keys ...string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	var longest time.Duration
	for _, key := range keys {
		e, ok := l.entries[key]
		if !ok {
			continue
		}
		if remaining := e.resetAt.Sub(l.now()); remaining > longest {
			longest = remaining
		}
	}
	if longest < 0 {
		return 0
	}
	return int(longest.Seconds()) + 1
}

func writeRateLimited(w http.ResponseWriter, retryAfterSec int) {
	if retryAfterSec < 1 {
		retryAfterSec = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(retryAfterSec))
	http.Error(w, "too many requests", http.StatusTooManyRequests)
}
