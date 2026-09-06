// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export const DETAIL_CACHE_TTL_MS = 5 * 60 * 1000;
export const DETAIL_CACHE_MAX_ENTRIES = 500;

interface DetailCacheEntry<T> {
  value: T;
  fetchedAt: number;
}

// Long browsing sessions can touch thousands of distinct albums/artists. Without
// a cap this cache would grow without bound for the lifetime of the app.
export function createDetailCache<T>(
  ttlMs = DETAIL_CACHE_TTL_MS,
  maxEntries = DETAIL_CACHE_MAX_ENTRIES,
) {
  const entries = new Map<string, DetailCacheEntry<T>>();

  function touch(id: string, entry: DetailCacheEntry<T>) {
    entries.delete(id);
    entries.set(id, entry);
  }

  function evictOverflow() {
    while (entries.size > maxEntries) {
      const oldest = entries.keys().next().value;
      if (oldest === undefined) break;
      entries.delete(oldest);
    }
  }

  return {
    peek(id: string): { value: T; stale: boolean } | null {
      const entry = entries.get(id);
      if (!entry) return null;
      touch(id, entry);
      return {
        value: entry.value,
        stale: Date.now() - entry.fetchedAt > ttlMs,
      };
    },
    set(id: string, value: T) {
      touch(id, { value, fetchedAt: Date.now() });
      evictOverflow();
    },
    delete(id: string) {
      entries.delete(id);
    },
    clear() {
      entries.clear();
    },
    get size() {
      return entries.size;
    },
  };
}
