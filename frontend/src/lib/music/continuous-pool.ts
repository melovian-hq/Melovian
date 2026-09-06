// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  isQueueUnlimited,
  remainingQueueSlots,
} from "$lib/music/queue-settings";
import type { SubsonicAlbum, SubsonicSong } from "$lib/subsonic/types";

export type ContinuousMode = "off" | "random" | "library" | "personal";

export const CONTINUOUS_MODES: readonly ContinuousMode[] = [
  "off",
  "random",
  "library",
  "personal",
] as const;

export const CONTINUOUS_MODE_LABELS: Record<ContinuousMode, string> = {
  off: "Off",
  random: "Random radio",
  library: "Library shuffle",
  personal: "Personal radio",
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
    value === "personal"
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
): number {
  const slots = remainingQueueSlots(queueLength, maxQueueSize);
  if (slots <= 0) return 0;

  if (mode === "random" || mode === "personal") {
    const belowTarget = Math.max(0, CONTINUOUS_SOFT_TARGET - queueLength);
    return Math.min(
      Math.max(belowTarget, CONTINUOUS_REFILL_BATCH),
      slots,
      Number.isFinite(slots) ? slots : CONTINUOUS_REFILL_BATCH * 2,
    );
  }

  if (mode === "library") {
    const target = continuousQueueTarget(maxQueueSize);
    const belowTarget = Math.max(0, target - queueLength);
    return Math.min(
      Math.max(belowTarget, CONTINUOUS_REFILL_BATCH),
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
