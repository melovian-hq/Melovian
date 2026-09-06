// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package localmusic

import (
	"sync"

	"melovian/internal/store"
)

const maxCatalogCacheEntries = 2

type CatalogCache struct {
	mu      sync.RWMutex
	entries map[string]cachedCatalog
	order   []string
}

type cachedCatalog struct {
	version     uint64
	catalog     Catalog
	approxBytes int
}

func NewCatalogCache() *CatalogCache {
	return &CatalogCache{
		entries: make(map[string]cachedCatalog, maxCatalogCacheEntries),
	}
}

func CatalogVersion(lib store.LocalLibrary) uint64 {
	return uint64(max(lib.UpdatedAt.Unix(), 0))<<32 | //#nosec G115 -- unix timestamps are non-negative
		uint64(max(lib.TrackCount, 0))<<16 | //#nosec G115 -- catalog counts are non-negative
		uint64(max(lib.MissingCount, 0))<<8 | //#nosec G115 -- catalog counts are non-negative
		uint64(max(lib.DuplicateCount, 0)) //#nosec G115 -- catalog counts are non-negative
}

func estimateCatalogBytes(catalog Catalog) int {
	return len(catalog.Songs)*256 + len(catalog.Albums)*192 + len(catalog.Artists)*128
}

func (c *CatalogCache) Get(libraryID string, version uint64) (Catalog, bool) {
	c.mu.RLock()
	entry, ok := c.entries[libraryID]
	c.mu.RUnlock()
	if !ok || entry.version != version {
		return Catalog{}, false
	}
	return entry.catalog, true
}

func (c *CatalogCache) Set(libraryID string, version uint64, catalog Catalog) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.entries[libraryID]; exists {
		c.removeLocked(libraryID)
	}

	c.entries[libraryID] = cachedCatalog{
		version:     version,
		catalog:     catalog,
		approxBytes: estimateCatalogBytes(catalog),
	}
	c.order = append(c.order, libraryID)

	for len(c.entries) > maxCatalogCacheEntries && len(c.order) > 0 {
		oldest := c.order[0]
		c.order = c.order[1:]
		c.removeLocked(oldest)
	}
}

func (c *CatalogCache) Invalidate(libraryID string) {
	c.mu.Lock()
	c.removeLocked(libraryID)
	c.mu.Unlock()
}

func (c *CatalogCache) Stats() (entries, totalBytes int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, entry := range c.entries {
		totalBytes += entry.approxBytes
	}
	return len(c.entries), totalBytes
}

func (c *CatalogCache) removeLocked(libraryID string) {
	if _, ok := c.entries[libraryID]; !ok {
		return
	}
	delete(c.entries, libraryID)
	for i, id := range c.order {
		if id == libraryID {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
}
