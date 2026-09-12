// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package apishared

import (
	"melovian/internal/httputil"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimiter is a fixed-window per-key limiter for credential endpoints and
// public write endpoints. Counters live in process memory, which matches the
// single-process deployment model. Keys are caller chosen (client IP, share
// token, username) so one abusive client cannot exhaust a global budget.
type RateLimiter struct {
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

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		entries: make(map[string]rateLimitEntry),
		max:     max,
		window:  window,
		now:     time.Now,
	}
}

// Blocked reports whether any key already exhausted its window budget.
// Callers pass several keys (client IP, username, share token) so spoofing
// one dimension does not dodge the limit.
func (l *RateLimiter) Blocked(keys ...string) bool {
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

// Record counts one attempt for each key. Expired entries are evicted lazily
// when the map grows so an attacker cannot grow memory without bound.
func (l *RateLimiter) Record(keys ...string) {
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

// Reset clears the counter for each key after a successful attempt.
func (l *RateLimiter) Reset(keys ...string) {
	l.mu.Lock()
	for _, key := range keys {
		delete(l.entries, key)
	}
	l.mu.Unlock()
}

// RetryAfterSeconds returns the longest remaining lockout for the keys.
func (l *RateLimiter) RetryAfterSeconds(keys ...string) int {
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

func WriteRateLimited(w http.ResponseWriter, retryAfterSec int) {
	if retryAfterSec < 1 {
		retryAfterSec = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(retryAfterSec))
	httputil.WriteError(w, http.StatusTooManyRequests, "too_many_requests", "too many requests")
}
