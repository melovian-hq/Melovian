// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { music } from "$lib/config/music.svelte";
import { createBoundedMap } from "$lib/core/bounded-cache";

const SERVER_COVER_CACHE_MAX = 48;
const serverCoverCache = createBoundedMap<string, string[]>(
  SERVER_COVER_CACHE_MAX,
);
const serverCoverInflight = new Map<string, Promise<string[]>>();

const MAX_CONCURRENT_COVER_FETCHES = 2;
let activeCoverFetches = 0;
const coverFetchWaiters: Array<() => void> = [];

async function withCoverFetchSlot<T>(fn: () => Promise<T>): Promise<T> {
  while (activeCoverFetches >= MAX_CONCURRENT_COVER_FETCHES) {
    await new Promise<void>((resolve) => {
      coverFetchWaiters.push(resolve);
    });
  }
  activeCoverFetches += 1;
  try {
    return await fn();
  } finally {
    activeCoverFetches -= 1;
    coverFetchWaiters.shift()?.();
  }
}

export function getCachedServerPlaylistCovers(
  playlistId: string,
): string[] | undefined {
  return serverCoverCache.get(playlistId);
}

export function clearServerPlaylistCoverCache(playlistId?: string): void {
  if (playlistId) {
    serverCoverCache.delete(playlistId);
    serverCoverInflight.delete(playlistId);
    return;
  }
  serverCoverCache.clear();
  serverCoverInflight.clear();
}

function coverIdsFromFallback(fallbackCoverArt?: string): string[] {
  const value = fallbackCoverArt?.trim();
  return value ? [value] : [];
}

export async function loadServerPlaylistCoverIds(
  playlistId: string,
  fallbackCoverArt?: string,
): Promise<string[]> {
  const cached = serverCoverCache.get(playlistId);
  if (cached) return cached;

  const inflight = serverCoverInflight.get(playlistId);
  if (inflight) return inflight;

  if (fallbackCoverArt?.trim()) {
    const ids = coverIdsFromFallback(fallbackCoverArt);
    serverCoverCache.set(playlistId, ids);
    return ids;
  }

  const task = withCoverFetchSlot(async () => {
    const result = await music
      .fetchServerPlaylist(playlistId)
      .catch(() => null);
    const ids: string[] = [];
    const seen = new Set<string>();

    const push = (id?: string) => {
      const value = id?.trim();
      if (!value || seen.has(value)) return;
      seen.add(value);
      ids.push(value);
    };

    for (const song of result?.songs ?? []) {
      push(song.coverArt ?? song.albumId ?? song.id);
      if (ids.length >= 3) break;
    }

    serverCoverCache.set(playlistId, ids);
    serverCoverInflight.delete(playlistId);
    return ids;
  });

  serverCoverInflight.set(playlistId, task);
  return task;
}

export async function enrichServerPlaylistCovers(
  playlistId: string,
  seedIds: string[],
): Promise<string[]> {
  const trimmed = seedIds.map((id) => id.trim()).filter(Boolean);
  if (trimmed.length >= 3) return trimmed.slice(0, 3);

  const enrichKey = `${playlistId}:enrich`;
  const cached = serverCoverCache.get(playlistId);
  if (cached && cached.length >= 3) return cached.slice(0, 3);
  if (cached && cached.length > trimmed.length) return cached;

  const inflight = serverCoverInflight.get(enrichKey);
  if (inflight) return inflight;

  const task = withCoverFetchSlot(async () => {
    const result = await music
      .fetchServerPlaylist(playlistId)
      .catch(() => null);
    const seen = new Set(trimmed);
    const ids = [...trimmed];

    for (const song of result?.songs ?? []) {
      const id = (song.coverArt ?? song.albumId ?? song.id)?.trim();
      if (!id || seen.has(id)) continue;
      seen.add(id);
      ids.push(id);
      if (ids.length >= 3) break;
    }

    serverCoverCache.set(playlistId, ids);
    serverCoverInflight.delete(enrichKey);
    return ids;
  });

  serverCoverInflight.set(enrichKey, task);
  return task;
}

export function localPlaylistCoverIds(playlist: {
  coverArtIds?: string[];
  tracks?: { coverArtId: string }[];
}): string[] {
  const fromList = playlist.coverArtIds?.filter(Boolean) ?? [];
  if (fromList.length > 0) return fromList.slice(0, 3);

  const ids: string[] = [];
  const seen = new Set<string>();
  for (const track of playlist.tracks ?? []) {
    const id = track.coverArtId?.trim();
    if (!id || seen.has(id)) continue;
    seen.add(id);
    ids.push(id);
    if (ids.length >= 3) break;
  }
  return ids;
}
