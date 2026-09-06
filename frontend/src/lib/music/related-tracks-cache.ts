// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { createDetailCache } from "$lib/core/detail-cache";
import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import { loadRelatedTracks } from "$lib/music/related-tracks";
import type { SubsonicSong } from "$lib/subsonic";

const RELATED_CACHE_TTL_MS = 5 * 60 * 1000;
const RELATED_CACHE_MAX_ENTRIES = 500;

const relatedCache = createDetailCache<SubsonicSong[]>(
  RELATED_CACHE_TTL_MS,
  RELATED_CACHE_MAX_ENTRIES,
);

const inflight = new Map<string, Promise<SubsonicSong[]>>();

function cacheKey(track: SubsonicSong, limit: number): string {
  return `${track.id ?? "unknown"}:${limit}`;
}

export function peekRelatedTracks(
  track: SubsonicSong,
  limit = 24,
): SubsonicSong[] | null {
  const cached = relatedCache.peek(cacheKey(track, limit));
  return cached?.value ?? null;
}

export async function getRelatedTracksCached(
  library: MusicLibraryAdapter,
  track: SubsonicSong,
  limit = 24,
): Promise<SubsonicSong[]> {
  const key = cacheKey(track, limit);
  const cached = relatedCache.peek(key);
  if (cached && !cached.stale) {
    return cached.value;
  }

  const existing = inflight.get(key);
  if (existing) return existing;

  const promise = loadRelatedTracks(library, track, limit)
    .then((songs) => {
      relatedCache.set(key, songs);
      inflight.delete(key);
      return songs;
    })
    .catch((error) => {
      inflight.delete(key);
      if (cached) return cached.value;
      throw error;
    });

  inflight.set(key, promise);
  return promise;
}

export function prefetchRelatedTracks(
  library: MusicLibraryAdapter,
  track: SubsonicSong | null | undefined,
  limit = 24,
): void {
  if (!track?.id) return;
  const key = cacheKey(track, limit);
  const cached = relatedCache.peek(key);
  if (cached && !cached.stale) return;
  if (inflight.has(key)) return;
  void getRelatedTracksCached(library, track, limit).catch(() => undefined);
}

export function clearRelatedTracksCache(): void {
  relatedCache.clear();
  inflight.clear();
}
