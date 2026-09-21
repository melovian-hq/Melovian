// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  isQueueUnlimited,
  remainingQueueSlots,
} from "$lib/music/queue-settings";
import type { SubsonicAlbum, SubsonicSong } from "$lib/subsonic/types";

export type ContinuousMode = "off" | "random" | "library" | "personal" | "all";

export const CONTINUOUS_MODES: readonly ContinuousMode[] = [
  "off",
  "random",
  "library",
  "personal",
  "all",
] as const;

export const CONTINUOUS_MODE_LABELS: Record<ContinuousMode, string> = {
  off: "Off",
  random: "Random radio",
  library: "Library shuffle",
  personal: "Personal radio",
  all: "Playing everything",
};

export const CONTINUOUS_REFILL_BATCH = 8;
export const CONTINUOUS_REFILL_THRESHOLD = 4;
export const CONTINUOUS_SOFT_TARGET = 25;
export const LIBRARY_SOFT_TARGET_WHEN_UNLIMITED = 500;
export const LIBRARY_ALBUM_PAGE_SIZE = 20;

export function parseContinuousMode(value: unknown): ContinuousMode {
  if (
    value === "off" ||
    value === "random" ||
    value === "library" ||
    value === "personal" ||
    value === "all"
  ) {
    return value;
  }
  return "off";
}

export function migrateContinuousMode(input: {
  continuousMode?: unknown;
  randomRadio?: unknown;
}): ContinuousMode {
  const parsed = parseContinuousMode(input.continuousMode);
  if (parsed !== "off") return parsed;
  if (input.randomRadio === true) return "random";
  return "off";
}

export function continuousQueueTarget(maxQueueSize: number): number {
  if (isQueueUnlimited(maxQueueSize)) return LIBRARY_SOFT_TARGET_WHEN_UNLIMITED;
  return Math.max(1, maxQueueSize);
}

export function continuousRefillCount(
  queueLength: number,
  maxQueueSize: number,
  mode: ContinuousMode,
  upcoming = queueLength,
): number {
  const slots = remainingQueueSlots(queueLength, maxQueueSize);
  if (slots <= 0) return 0;

  if (mode === "random" || mode === "personal") {
    const deficit = Math.max(0, CONTINUOUS_SOFT_TARGET - upcoming);
    if (deficit <= 0) return 0;
    return Math.min(
      Math.max(deficit, CONTINUOUS_REFILL_BATCH),
      slots,
      Number.isFinite(slots) ? slots : CONTINUOUS_REFILL_BATCH * 2,
    );
  }

  if (mode === "library" || mode === "all") {
    const target = continuousQueueTarget(maxQueueSize);
    const deficit = Math.max(0, target - upcoming);
    if (deficit <= 0) return 0;
    return Math.min(
      Math.max(deficit, CONTINUOUS_REFILL_BATCH),
      slots,
      Number.isFinite(slots) ? slots : LIBRARY_ALBUM_PAGE_SIZE * 4,
    );
  }

  return 0;
}

export interface LibraryPoolState {
  albumOffset: number;
  seenTrackIds: Set<string>;
  pendingTracks: SubsonicSong[];
  exhaustedPasses: number;
}

export function createLibraryPoolState(): LibraryPoolState {
  return {
    albumOffset: 0,
    seenTrackIds: new Set(),
    pendingTracks: [],
    exhaustedPasses: 0,
  };
}

export function resetLibraryPool(state: LibraryPoolState): void {
  state.albumOffset = 0;
  state.seenTrackIds.clear();
  state.pendingTracks = [];
  state.exhaustedPasses += 1;
}

function shuffleInPlace<T>(items: T[], random: () => number): void {
  for (let i = items.length - 1; i > 0; i--) {
    const j = Math.floor(random() * (i + 1));
    [items[i], items[j]] = [items[j], items[i]];
  }
}

export interface LibraryPoolFetchers {
  getAlbumList2(
    type: "random" | "alphabeticalByName",
    size: number,
    offset?: number,
  ): Promise<SubsonicAlbum[]>;
  getAlbumSongs(albumId: string): Promise<SubsonicSong[]>;
}

/**
 * Pull the next batch of library tracks without loading the whole catalog.
 * Uses random album pages first, then alphabetical pagination, then resets.
 */
export async function pullLibraryTracks(
  state: LibraryPoolState,
  fetchers: LibraryPoolFetchers,
  count: number,
  excludeIds: ReadonlySet<string>,
  random: () => number = Math.random,
): Promise<SubsonicSong[]> {
  if (count <= 0) return [];

  const out: SubsonicSong[] = [];

  const takePending = () => {
    while (out.length < count && state.pendingTracks.length > 0) {
      const track = state.pendingTracks.shift();
      if (!track?.id) continue;
      if (excludeIds.has(track.id) || state.seenTrackIds.has(track.id))
        continue;
      state.seenTrackIds.add(track.id);
      out.push(track);
    }
  };

  takePending();
  if (out.length >= count) return out;

  let emptyPages = 0;
  while (out.length < count && emptyPages < 4) {
    const albums = await fetchers
      .getAlbumList2("random", LIBRARY_ALBUM_PAGE_SIZE, state.albumOffset)
      .catch(() => [] as SubsonicAlbum[]);

    if (albums.length === 0) {
      const alpha = await fetchers
        .getAlbumList2(
          "alphabeticalByName",
          LIBRARY_ALBUM_PAGE_SIZE,
          state.albumOffset,
        )
        .catch(() => [] as SubsonicAlbum[]);

      if (alpha.length === 0) {
        if (state.seenTrackIds.size === 0 && state.pendingTracks.length === 0) {
          break;
        }
        resetLibraryPool(state);
        emptyPages += 1;
        continue;
      }

      await enqueueAlbumSongs(state, fetchers, alpha, random);
      state.albumOffset += alpha.length;
    } else {
      await enqueueAlbumSongs(state, fetchers, albums, random);
      state.albumOffset += albums.length;
    }

    const before = out.length;
    takePending();
    if (out.length === before) emptyPages += 1;
    else emptyPages = 0;
  }

  return out;
}

async function enqueueAlbumSongs(
  state: LibraryPoolState,
  fetchers: LibraryPoolFetchers,
  albums: readonly SubsonicAlbum[],
  random: () => number,
): Promise<void> {
  const songLists = await Promise.all(
    albums.map((album) =>
      fetchers.getAlbumSongs(album.id).catch(() => [] as SubsonicSong[]),
    ),
  );
  const batch: SubsonicSong[] = [];
  for (const songs of songLists) {
    for (const song of songs) {
      if (!song.id) continue;
      if (state.seenTrackIds.has(song.id)) continue;
      batch.push(song);
    }
  }
  shuffleInPlace(batch, random);
  state.pendingTracks.push(...batch);
}

export function filterUniqueTracks(
  tracks: readonly SubsonicSong[],
  existingIds: ReadonlySet<string>,
): SubsonicSong[] {
  const seen = new Set(existingIds);
  const out: SubsonicSong[] = [];
  for (const track of tracks) {
    if (!track.id || seen.has(track.id)) continue;
    seen.add(track.id);
    out.push(track);
  }
  return out;
}

export const FOREVER_ENUM_PAGE_SIZE = 500;
export const FOREVER_ALBUMS_PER_ROUND = 6;
export const FOREVER_PENDING_CAP = 200;

/**
 * Pool for the play-everything mode. The whole library is enumerated once as
 * a shuffled list of album ids, then walked in small album batches. Because
 * each album is visited once per pass, tracks can never repeat inside a
 * pass, so there is no per-track seen set to grow with library size. Memory
 * stays bounded at the album id list plus a capped pending buffer.
 */
export interface ForeverPoolState {
  albumIds: string[];
  albumCursor: number;
  enumerated: boolean;
  pendingTracks: SubsonicSong[];
  pendingIds: Set<string>;
  pass: number;
}

export function createForeverPoolState(): ForeverPoolState {
  return {
    albumIds: [],
    albumCursor: 0,
    enumerated: false,
    pendingTracks: [],
    pendingIds: new Set(),
    pass: 0,
  };
}

async function enumerateAlbumIds(
  state: ForeverPoolState,
  fetchers: LibraryPoolFetchers,
  random: () => number,
): Promise<void> {
  if (state.enumerated) return;
  // Resume where a previous partial enumeration stopped. Alphabetical
  // pagination is stable, so offset equals the id count collected so far.
  let offset = state.albumIds.length;
  let complete = false;
  while (true) {
    let page: SubsonicAlbum[];
    try {
      page = await fetchers.getAlbumList2(
        "alphabeticalByName",
        FOREVER_ENUM_PAGE_SIZE,
        offset,
      );
    } catch {
      break;
    }
    if (page.length === 0) {
      complete = true;
      break;
    }
    for (const album of page) {
      if (album?.id) state.albumIds.push(album.id);
    }
    offset += page.length;
    if (page.length < FOREVER_ENUM_PAGE_SIZE) {
      complete = true;
      break;
    }
  }
  state.albumIds = [...new Set(state.albumIds)];
  if (complete) {
    shuffleInPlace(state.albumIds, random);
    state.enumerated = true;
  }
}

/**
 * Pull tracks from the shuffled all-library album walk. Reaching the end of
 * the id list starts a fresh pass with a re-shuffle and re-enumeration so
 * library additions join the rotation.
 */
export async function pullForeverTracks(
  state: ForeverPoolState,
  fetchers: LibraryPoolFetchers,
  count: number,
  excludeIds: ReadonlySet<string>,
  random: () => number = Math.random,
): Promise<SubsonicSong[]> {
  if (count <= 0) return [];

  const out: SubsonicSong[] = [];
  // A pass rollover inside one call refetches albums, so tracks handed out
  // already must be tracked here or the same pull can emit duplicates.
  const delivered = new Set<string>();

  const takePending = () => {
    while (out.length < count && state.pendingTracks.length > 0) {
      const track = state.pendingTracks.shift();
      if (!track?.id) continue;
      state.pendingIds.delete(track.id);
      if (excludeIds.has(track.id) || delivered.has(track.id)) continue;
      delivered.add(track.id);
      out.push(track);
    }
  };

  takePending();
  if (out.length >= count) return out;

  if (!state.enumerated) {
    await enumerateAlbumIds(state, fetchers, random);
  }

  let guard = 0;
  while (out.length < count && guard < 8) {
    guard += 1;
    const before = out.length;
    let rolledOver = false;

    if (state.albumCursor >= state.albumIds.length) {
      if (state.albumIds.length === 0) break;
      state.albumCursor = 0;
      state.pass += 1;
      rolledOver = true;
      // Rebuild the id list so a fresh pass picks up library changes.
      state.enumerated = false;
      state.albumIds = [];
      await enumerateAlbumIds(state, fetchers, random);
      if (state.albumIds.length === 0) break;
    }

    const ids = state.albumIds.slice(
      state.albumCursor,
      state.albumCursor + FOREVER_ALBUMS_PER_ROUND,
    );
    state.albumCursor += ids.length;
    if (ids.length === 0) continue;

    const songLists = await Promise.all(
      ids.map((id) =>
        fetchers.getAlbumSongs(id).catch(() => [] as SubsonicSong[]),
      ),
    );
    const batch: SubsonicSong[] = [];
    for (const songs of songLists) {
      for (const song of songs) {
        if (!song.id || state.pendingIds.has(song.id)) continue;
        state.pendingIds.add(song.id);
        batch.push(song);
      }
    }
    shuffleInPlace(batch, random);

    const room = Math.max(0, FOREVER_PENDING_CAP - state.pendingTracks.length);
    if (batch.length > room) {
      for (const track of batch.slice(room)) {
        state.pendingIds.delete(track.id);
      }
      batch.length = room;
    }
    state.pendingTracks.push(...batch);
    takePending();
    // A pass that produced nothing new means the catalog is fully delivered
    // and smaller than the request. Stop instead of re-enumerating again.
    if (rolledOver && out.length === before) break;
  }

  return out;
}
