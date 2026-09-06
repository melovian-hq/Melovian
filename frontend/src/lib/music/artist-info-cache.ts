// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { createDetailCache, DETAIL_CACHE_TTL_MS } from "$lib/core/detail-cache";
import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import { isLocalMusicId } from "$lib/music/library-adapter";
import type { SubsonicArtistInfo } from "$lib/subsonic/types";

const ARTIST_INFO_CACHE_TTL_MS = 30 * 60 * 1000;
const STORAGE_PREFIX = "mel-artist-info:";
const STORAGE_TTL_MS = 24 * 60 * 60 * 1000;

const infoCache = createDetailCache<SubsonicArtistInfo>(
  ARTIST_INFO_CACHE_TTL_MS,
);
const inflight = new Map<string, Promise<SubsonicArtistInfo>>();

interface StoredArtistInfo {
  at: number;
  value: SubsonicArtistInfo;
}

interface PersistedArtistInfo {
  value: SubsonicArtistInfo;
  stale: boolean;
}

function readPersistentArtistInfo(id: string): PersistedArtistInfo | null {
  if (typeof localStorage === "undefined") return null;
  try {
    const raw = localStorage.getItem(`${STORAGE_PREFIX}${id}`);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as StoredArtistInfo;
    if (Date.now() - parsed.at > STORAGE_TTL_MS) {
      localStorage.removeItem(`${STORAGE_PREFIX}${id}`);
      return null;
    }
    return {
      value: parsed.value,
      stale: Date.now() - parsed.at > ARTIST_INFO_CACHE_TTL_MS,
    };
  } catch {
    return null;
  }
}

function writePersistentArtistInfo(
  id: string,
  value: SubsonicArtistInfo,
): void {
  if (typeof localStorage === "undefined") return;
  try {
    const payload: StoredArtistInfo = { at: Date.now(), value };
    localStorage.setItem(`${STORAGE_PREFIX}${id}`, JSON.stringify(payload));
  } catch {
    /* storage full or unavailable */
  }
}

function emptyArtistInfo(): SubsonicArtistInfo {
  return { similarArtists: [] };
}

export async function fetchArtistInfoWithCache(
  library: MusicLibraryAdapter,
  id: string,
  onStale?: (data: SubsonicArtistInfo) => void,
): Promise<SubsonicArtistInfo> {
  if (isLocalMusicId(id)) {
    return emptyArtistInfo();
  }

  const cached = infoCache.peek(id);
  if (cached && !cached.stale) return cached.value;
  if (cached?.stale) {
    onStale?.(cached.value);
    void revalidateArtistInfo(library, id);
    return cached.value;
  }

  const persisted = readPersistentArtistInfo(id);
  if (persisted) {
    infoCache.set(id, persisted.value);
    onStale?.(persisted.value);
    if (persisted.stale) {
      void revalidateArtistInfo(library, id);
    }
    return persisted.value;
  }

  const pending = inflight.get(id);
  if (pending) return pending;

  const promise = (async () => {
    const fresh = await library
      .getArtistInfo(id)
      .catch(() => emptyArtistInfo());
    infoCache.set(id, fresh);
    writePersistentArtistInfo(id, fresh);
    return fresh;
  })();

  inflight.set(id, promise);
  try {
    return await promise;
  } finally {
    inflight.delete(id);
  }
}

async function revalidateArtistInfo(
  library: MusicLibraryAdapter,
  id: string,
): Promise<void> {
  try {
    const fresh = await library.getArtistInfo(id);
    infoCache.set(id, fresh);
    writePersistentArtistInfo(id, fresh);
  } catch {
    /* keep stale cache */
  }
}

export function prefetchArtistInfo(
  library: MusicLibraryAdapter,
  id: string,
): void {
  if (isLocalMusicId(id)) return;
  const cached = infoCache.peek(id);
  if (cached && !cached.stale) return;
  void fetchArtistInfoWithCache(library, id);
}

export function clearArtistInfoMemoryCache(id?: string) {
  if (id) {
    infoCache.delete(id);
    inflight.delete(id);
    return;
  }
  infoCache.clear();
  inflight.clear();
}

export function invalidateArtistInfoCache(id?: string) {
  if (id) {
    infoCache.delete(id);
    inflight.delete(id);
    if (typeof localStorage !== "undefined") {
      localStorage.removeItem(`${STORAGE_PREFIX}${id}`);
    }
    return;
  }
  infoCache.clear();
  inflight.clear();
  if (typeof localStorage === "undefined") return;
  const keys: string[] = [];
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i);
    if (key?.startsWith(STORAGE_PREFIX)) keys.push(key);
  }
  for (const key of keys) localStorage.removeItem(key);
}

export function resetArtistInfoCaches() {
  invalidateArtistInfoCache();
}

export { ARTIST_INFO_CACHE_TTL_MS, DETAIL_CACHE_TTL_MS };
