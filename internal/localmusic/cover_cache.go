// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package localmusic

import (
	"sync"
)

const (
	maxCoverCacheEntries = 256
	maxCoverCacheBytes   = 64 << 20
)

type CoverCache struct {
	mu         sync.Mutex
	entries    map[string]coverEntry
	order      []string
	totalBytes int
}

type coverEntry struct {
	data []byte
	mime string
}

func NewCoverCache() *CoverCache {
	return &CoverCache{
		entries: make(map[string]coverEntry, 32),
	}
}

func coverCacheKey(libraryID, trackID string) string {
	return libraryID + ":" + trackID
}

func (c *CoverCache) Get(libraryID, trackID string) ([]byte, string, bool) {
	key := coverCacheKey(libraryID, trackID)
	c.mu.Lock()
	entry, ok := c.entries[key]
	c.mu.Unlock()
	if !ok {
		return nil, "", false
	}
	return entry.data, entry.mime, true
}

func (c *CoverCache) Set(libraryID, trackID string, data []byte, mime string) {
	key := coverCacheKey(libraryID, trackID)
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.entries[key]; exists {
		c.removeLocked(key)
	}

	c.entries[key] = coverEntry{data: data, mime: mime}
	c.order = append(c.order, key)
	c.totalBytes += len(data)

	for (len(c.entries) > maxCoverCacheEntries || c.totalBytes > maxCoverCacheBytes) && len(c.order) > 0 {
		oldest := c.order[0]
		c.order = c.order[1:]
		c.removeLocked(oldest)
	}
}

// removeLocked deletes key from entries/order and keeps totalBytes in sync.
// Caller must hold c.mu.
func (c *CoverCache) removeLocked(key string) {
	entry, ok := c.entries[key]
	if !ok {
		return
	}
	delete(c.entries, key)
	c.totalBytes -= len(entry.data)
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
}

func (c *CoverCache) Stats() (entries, totalBytes int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries), c.totalBytes
}

func (c *CoverCache) InvalidateLibrary(libraryID string) {
	prefix := libraryID + ":"
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.entries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			c.removeLocked(key)
		}
	}
}
