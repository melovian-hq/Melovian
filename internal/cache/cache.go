// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	MaxEntries    = 2048
	MaxBodyBytes  = 4 << 20
	MaxTotalBytes = 64 << 20
)

type Entry struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	ExpiresAt  time.Time
}

type ResponseCache struct {
	mu         sync.Mutex
	entries    map[string]Entry
	order      []string
	totalBytes int
}

func NewResponseCache() *ResponseCache {
	return &ResponseCache{
		entries: make(map[string]Entry, 256),
	}
}

func (c *ResponseCache) Get(key string) (Entry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return Entry{}, false
	}
	if time.Now().After(entry.ExpiresAt) {
		c.removeLocked(key)
		return Entry{}, false
	}
	return entry, true
}

func (c *ResponseCache) Set(key string, entry Entry) {
	if len(entry.Body) > MaxBodyBytes {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.entries[key]; exists {
		c.removeLocked(key)
	}

	c.entries[key] = entry
	c.order = append(c.order, key)
	c.totalBytes += len(entry.Body)

	if len(c.entries) > MaxEntries || c.totalBytes > MaxTotalBytes {
		c.evictExpiredLocked(time.Now())
	}
	for (len(c.entries) > MaxEntries || c.totalBytes > MaxTotalBytes) && len(c.order) > 0 {
		c.evictOldestLocked()
	}
}

// removeLocked deletes key and keeps totalBytes/order in sync. Caller holds c.mu.
func (c *ResponseCache) removeLocked(key string) {
	entry, ok := c.entries[key]
	if !ok {
		return
	}
	delete(c.entries, key)
	c.totalBytes -= len(entry.Body)
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
}

func (c *ResponseCache) evictExpiredLocked(now time.Time) {
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			c.removeLocked(key)
		}
	}
}

func (c *ResponseCache) Stats() (entries, totalBytes int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries), c.totalBytes
}

func (c *ResponseCache) InvalidatePrefix(prefix string) int {
	if prefix == "" {
		return 0
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	for key := range c.entries {
		if strings.HasPrefix(key, prefix) {
			c.removeLocked(key)
			removed++
		}
	}
	return removed
}

func (c *ResponseCache) evictOldestLocked() {
	if len(c.order) == 0 {
		return
	}
	oldest := c.order[0]
	c.order = c.order[1:]
	if entry, ok := c.entries[oldest]; ok {
		delete(c.entries, oldest)
		c.totalBytes -= len(entry.Body)
	}
}

func Key(method, path, rawQuery string) string {
	n := len(method) + 1 + len(path)
	if rawQuery != "" {
		n += 1 + len(rawQuery)
	}
	b := make([]byte, 0, n)
	b = append(b, method...)
	b = append(b, ' ')
	b = append(b, path...)
	if rawQuery != "" {
		b = append(b, '?')
		b = append(b, rawQuery...)
	}
	return string(b)
}

func TTL(path string) time.Duration {
	switch {
	case strings.Contains(path, "/Images/"):
		return 24 * time.Hour
	case strings.HasPrefix(path, "/System/Info"):
		return 15 * time.Minute
	case strings.Contains(path, "/Items/") && strings.HasSuffix(path, "/Similar"):
		return 10 * time.Minute
	case strings.HasPrefix(path, "/Persons"):
		return 10 * time.Minute
	case strings.Contains(path, "/Views"):
		return 10 * time.Minute
	case strings.HasPrefix(path, "/Items"):
		return 5 * time.Minute
	case strings.HasPrefix(path, "/Shows/"):
		return 5 * time.Minute
	case strings.HasPrefix(path, "/Search/"):
		return 90 * time.Second
	case strings.Contains(path, "/Latest"):
		return 60 * time.Second
	case strings.Contains(path, "/Resume"):
		return 45 * time.Second
	case strings.HasSuffix(path, ".vtt"):
		return 30 * time.Minute
	default:
		return 2 * time.Minute
	}
}

func ShouldCacheRequest(method, path string) bool {
	if method != http.MethodGet {
		return false
	}

	lower := strings.ToLower(path)
	if strings.HasPrefix(lower, "/videos/") {
		return false
	}
	if strings.HasPrefix(path, "/Sessions/") {
		return false
	}
	if strings.Contains(path, "/PlaybackInfo") {
		return false
	}
	if strings.Contains(path, "/stream") {
		return false
	}
	if strings.HasSuffix(lower, ".m3u8") || strings.HasSuffix(lower, ".ts") {
		return false
	}

	return true
}

func CloneHeader(header http.Header) http.Header {
	cloned := make(http.Header, len(header))
	for key, values := range header {
		copied := make([]string, len(values))
		copy(copied, values)
		cloned[key] = copied
	}
	return cloned
}

func WriteCachedResponse(w http.ResponseWriter, entry Entry, cacheStatus string) {
	for key, values := range entry.Header {
		if len(values) == 0 {
			continue
		}
		w.Header().Set(key, values[0])
		for _, value := range values[1:] {
			w.Header().Add(key, value)
		}
	}
	w.Header().Set("X-Cache", cacheStatus)
	w.WriteHeader(entry.StatusCode)
	_, _ = w.Write(entry.Body)
}
