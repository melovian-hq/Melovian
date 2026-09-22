// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { interleaveByArtist, orderForFlow } from "$lib/music/mix-generator";
import {
  buildSessionMood,
  buildTasteProfile,
  pickQualitySeedIds,
  weightedSampleTracks,
  type TasteProfile,
} from "$lib/music/taste-score";
import type {
  ListenEntry,
  ListenStats,
  SubsonicSong,
} from "$lib/subsonic/types";

export interface PersonalRadioOptions {
  recencyCooldownMs: number;
  exploreBonus: number;
  albumLookback: number;
  coldStart: boolean;
  starredTrackIds?: ReadonlySet<string>;
}

export const DEFAULT_PERSONAL_RADIO_OPTIONS: PersonalRadioOptions = {
  recencyCooldownMs: 2 * 60 * 60 * 1000,
  exploreBonus: 0.6,
  albumLookback: 4,
  coldStart: false,
};

function seededRandom(seed: string): () => number {
  let state = 2166136261;
  for (let i = 0; i < seed.length; i++) {
    state ^= seed.charCodeAt(i);
    state = Math.imul(state, 16777619);
  }
  state = state >>> 0 || 1;
  return () => {
    state = Math.imul(state ^ (state >>> 15), state | 1);
    state ^= state + Math.imul(state ^ (state >>> 7), state | 61);
    return ((state ^ (state >>> 14)) >>> 0) / 4294967296;
  };
}

export interface PersonalRadioState {
  skipTrackIds: Set<string>;
  skipArtistKeys: Map<string, number>;
  seedTrackIds: string[];
  lastCompletedId: string | null;
  /** Newest-first completions used as the live mood vector. */
  recentCompletions: SubsonicSong[];
  sessionSeed: string;
  similarCache: Map<string, SubsonicSong[]>;
  /**
   * Counts seed batches this session. Each batch re-seeds the RNG and
   * rotates the seed and artist windows, so refills keep exploring instead
   * of re-dealing the same picks.
   */
  batchIndex: number;
  /** Skip timestamps used to adapt exploration to session mood. */
  recentSkipAt: number[];
}

export function createPersonalRadioState(): PersonalRadioState {
  return {
    skipTrackIds: new Set(),
    skipArtistKeys: new Map(),
    seedTrackIds: [],
    lastCompletedId: null,
    recentCompletions: [],
    sessionSeed: `personal-radio:${Date.now()}`,
    similarCache: new Map(),
    batchIndex: 0,
    recentSkipAt: [],
  };
}

function artistKey(track: SubsonicSong): string {
  return track.artist?.trim().toLowerCase() ?? "";
}

export function notePersonalSkip(
  state: PersonalRadioState,
  track: SubsonicSong,
): void {
  if (track.id) state.skipTrackIds.add(track.id);
  state.recentSkipAt.push(Date.now());
  if (state.recentSkipAt.length > 32) {
    state.recentSkipAt.splice(0, state.recentSkipAt.length - 32);
  }
  const key = artistKey(track);
  if (key) {
    state.skipArtistKeys.set(key, (state.skipArtistKeys.get(key) ?? 0) + 1);
  }
}

export function notePersonalComplete(
  state: PersonalRadioState,
  track: SubsonicSong,
): void {
  state.lastCompletedId = track.id || null;
  if (track.id) state.skipTrackIds.delete(track.id);
  const key = artistKey(track);
  if (key) {
    const count = state.skipArtistKeys.get(key) ?? 0;
    if (count <= 1) state.skipArtistKeys.delete(key);
    else state.skipArtistKeys.set(key, count - 1);
  }
  if (track.id) {
    state.recentCompletions = [
      track,
      ...state.recentCompletions.filter((entry) => entry.id !== track.id),
    ].slice(0, 8);
  }
}

export function isPersonalColdStart(
  history: readonly ListenEntry[],
  stats: ListenStats | null,
  minPlays = 5,
): boolean {
  const plays = stats?.totalPlays ?? history.length;
  return plays < minPlays;
}

export function buildPersonalProfile(
  stats: ListenStats | null,
  history: readonly ListenEntry[],
  playCountByTrack: ReadonlyMap<string, number>,
  skippedTrackIds: ReadonlySet<string>,
  sessionSkips: ReadonlySet<string>,
  listenedMsByTrack: ReadonlyMap<string, number> = new Map(),
): TasteProfile {
  const mergedSkips = new Set([...skippedTrackIds, ...sessionSkips]);
  return buildTasteProfile(
    stats,
    history,
    playCountByTrack,
    mergedSkips,
    [],
    listenedMsByTrack,
  );
}

export interface PersonalRadioFetchers {
  getSimilarSongs(trackId: string, count: number): Promise<SubsonicSong[]>;
  getRandomSongs(count: number): Promise<SubsonicSong[]>;
  searchArtistSongs(artist: string, limit: number): Promise<SubsonicSong[]>;
  /**
   * Resolve a seed id to a full track when it is neither a recent
   * completion nor in listen history (for example after a restore).
   */
  resolveTrack?(trackId: string): Promise<SubsonicSong | null>;
  /**
   * Richer related-track lookup with artist and album fallbacks. Preferred
   * over getSimilarSongs when available because the Subsonic similar-songs
   * endpoint expects an artist id and silently returns nothing for track
   * ids on several servers.
   */
  getRelatedTracks?(
    track: SubsonicSong,
    count: number,
  ): Promise<SubsonicSong[]>;
}

async function getCachedRelatedSongs(
  state: PersonalRadioState,
  fetchers: PersonalRadioFetchers,
  seed: SubsonicSong,
  count: number,
): Promise<SubsonicSong[]> {
  const key = seed.id ?? "";
  if (key) {
    const cached = state.similarCache.get(key);
    if (cached) return cached;
  }
  const songs = fetchers.getRelatedTracks
    ? await fetchers.getRelatedTracks(seed, count).catch(() => [])
    : key
      ? await fetchers.getSimilarSongs(key, count).catch(() => [])
      : [];
  // Keep the session cache bounded; oldest seeds age out first.
  if (key) {
    if (state.similarCache.size >= 64) {
      const oldest = state.similarCache.keys().next().value;
      if (oldest !== undefined) state.similarCache.delete(oldest);
    }
    state.similarCache.set(key, songs);
  }
  return songs;
}

function recentPlayIds(
  profile: TasteProfile,
  nowMs: number,
  recencyCooldownMs: number,
): Set<string> {
  const recent = new Set<string>();
  for (const [id, at] of profile.lastPlayedAt) {
    if (nowMs - at < recencyCooldownMs) recent.add(id);
  }
  return recent;
}

/** Skips inside this window count toward the restlessness signal. */
const SKIP_ADAPT_WINDOW_MS = 15 * 60 * 1000;
const SKIP_ADAPT_THRESHOLD = 3;
/** Wide seed pool that refill batches rotate through. */
const SEED_POOL_SIZE = 12;
const SEEDS_PER_BATCH = 5;
const ARTIST_POOL_SIZE = 12;
const ARTISTS_PER_BATCH = 5;

function rotateWindow<T>(
  items: readonly T[],
  size: number,
  offset: number,
): T[] {
  if (items.length <= size) return [...items];
  const start = ((offset % items.length) + items.length) % items.length;
  const out: T[] = [];
  for (let i = 0; i < size; i++) {
    out.push(items[(start + i) % items.length]);
  }
  return out;
}

async function resolveSeedTracks(
  state: PersonalRadioState,
  fetchers: PersonalRadioFetchers,
  history: readonly ListenEntry[],
  ids: readonly string[],
  entryToSong: (entry: ListenEntry) => SubsonicSong,
): Promise<Map<string, SubsonicSong>> {
  const resolved = new Map<string, SubsonicSong>();
  for (const track of state.recentCompletions) {
    if (track.id) resolved.set(track.id, track);
  }
  const historyById = new Map(history.map((entry) => [entry.trackId, entry]));
  const missing: string[] = [];
  for (const id of ids) {
    if (resolved.has(id)) continue;
    const entry = historyById.get(id);
    if (entry) resolved.set(id, entryToSong(entry));
    else missing.push(id);
  }
  if (missing.length > 0 && fetchers.resolveTrack) {
    const fetched = await Promise.all(
      missing.map((id) =>
        fetchers.resolveTrack!(id).catch(() => null as SubsonicSong | null),
      ),
    );
    missing.forEach((id, index) => {
      const track = fetched[index];
      if (track?.id) resolved.set(id, track);
    });
  }
  return resolved;
}

function recentSkipCount(state: PersonalRadioState, nowMs: number): number {
  let count = 0;
  for (const at of state.recentSkipAt) {
    if (nowMs - at < SKIP_ADAPT_WINDOW_MS) count += 1;
  }
  return count;
}

export async function seedPersonalRadioTracks(
  state: PersonalRadioState,
  fetchers: PersonalRadioFetchers,
  profile: TasteProfile,
  history: readonly ListenEntry[],
  stats: ListenStats | null,
  count: number,
  excludeIds: ReadonlySet<string>,
  entryToSong: (entry: ListenEntry) => SubsonicSong,
  options: PersonalRadioOptions = DEFAULT_PERSONAL_RADIO_OPTIONS,
): Promise<SubsonicSong[]> {
  if (count <= 0) return [];

  const batchIndex = state.batchIndex++;
  const random = seededRandom(`${state.sessionSeed}:${batchIndex}`);
  const pool: SubsonicSong[] = [];
  const nowMs = Date.now();
  const coldStart = options.coldStart;

  const seedIds = pickQualitySeedIds(
    history,
    stats,
    SEED_POOL_SIZE,
    state.lastCompletedId,
    nowMs,
  );

  state.seedTrackIds = seedIds.filter(Boolean);

  // Always keep the last completed track as the anchor seed, then rotate a
  // window over the wider quality pool so each refill leans on a different
  // slice of taste instead of re-seeding from the same five tracks.
  const rotatedSeeds = rotateWindow(
    seedIds.filter((id) => id !== state.lastCompletedId),
    SEEDS_PER_BATCH,
    batchIndex * 3,
  );
  const similarSeedIds = [
    ...new Set(
      [...(state.lastCompletedId ? [state.lastCompletedId] : []), ...rotatedSeeds].filter(
        Boolean,
      ),
    ),
  ].slice(0, SEEDS_PER_BATCH);

  const artistPool = stats?.topArtists.slice(0, coldStart ? 4 : ARTIST_POOL_SIZE) ?? [];
  const topArtists = rotateWindow(
    artistPool,
    coldStart ? 2 : ARTISTS_PER_BATCH,
    batchIndex * 4,
  );
  const randomCount = coldStart
    ? Math.max(count, 40)
    : Math.max(count * 2, 30);

  const moodTracks =
    state.recentCompletions.length > 0
      ? state.recentCompletions
      : history.slice(0, 5).map(entryToSong);
  const sessionMood = buildSessionMood(moodTracks, coldStart ? 0.35 : 1);

  const seedTracks = await resolveSeedTracks(
    state,
    fetchers,
    history,
    similarSeedIds,
    entryToSong,
  );

  const [similarGroups, artistSongs, randomSongs] = await Promise.all([
    Promise.all(
      similarSeedIds.map((id) =>
        getCachedRelatedSongs(
          state,
          fetchers,
          seedTracks.get(id) ?? ({ id } as SubsonicSong),
          Math.max(10, Math.ceil(count / 2)),
        ),
      ),
    ),
    Promise.all(
      topArtists.map((artist) =>
        fetchers
          .searchArtistSongs(artist.label, 8)
          .catch(() => [] as SubsonicSong[]),
      ),
    ),
    fetchers.getRandomSongs(randomCount).catch(() => [] as SubsonicSong[]),
  ]);

  pool.push(...similarGroups.flat());
  pool.push(...artistSongs.flat());
  if (!coldStart) {
    for (const entry of history.slice(0, 20)) {
      pool.push(entryToSong(entry));
    }
  }
  pool.push(...randomSongs);

  const exclude = new Set(excludeIds);
  for (const id of state.skipTrackIds) exclude.add(id);

  const recentIds = recentPlayIds(profile, nowMs, options.recencyCooldownMs);

  const keepTrack = (track: SubsonicSong): boolean => {
    const id = track.id?.trim();
    if (!id || exclude.has(track.id) || exclude.has(id)) return false;
    if (recentIds.has(id) && id !== state.lastCompletedId && !coldStart) {
      return false;
    }
    const key = artistKey(track);
    if (key && (state.skipArtistKeys.get(key) ?? 0) >= 3) return false;
    return true;
  };

  const filtered = pool.filter(keepTrack);

  // When the candidate pool comes back thin (dead similar-songs endpoint,
  // small library, heavy exclusions) top up with one extra random draw so
  // a refill batch still has room to pick.
  if (filtered.length < Math.max(count, 12)) {
    const extra = await fetchers
      .getRandomSongs(Math.max(count * 2, 30))
      .catch(() => [] as SubsonicSong[]);
    for (const track of extra) {
      if (keepTrack(track)) filtered.push(track);
    }
  }

  // Adapt exploration to session mood: a run of skips means the listener
  // wants safer picks, while a streak of completions earns more discovery.
  const skips = recentSkipCount(state, nowMs);
  let exploreBonus = coldStart
    ? Math.max(1, options.exploreBonus + 0.4)
    : options.exploreBonus;
  if (skips >= SKIP_ADAPT_THRESHOLD) {
    exploreBonus = Math.max(0.15, exploreBonus * 0.5);
  } else if (!coldStart && skips === 0 && state.recentCompletions.length >= 4) {
    exploreBonus += 0.15;
  }

  const sampled = weightedSampleTracks(
    filtered,
    profile,
    count,
    random,
    {
      exploreBonus,
      sessionMood,
      starredTrackIds: options.starredTrackIds,
    },
    exclude,
  );

  const interleaved = interleaveByArtist(
    sampled,
    `${state.sessionSeed}:interleave:${batchIndex}`,
  );
  const ordered = orderForFlow(
    interleaved,
    `${state.sessionSeed}:flow:${batchIndex}`,
    {
      albumLookback: options.albumLookback,
    },
  );

  return ordered.slice(0, count);
}

export function wasTrackSkipped(
  positionMs: number,
  durationMs: number,
): boolean {
  if (durationMs <= 0) return positionMs < 30_000;
  return positionMs / durationMs < 0.5;
}
