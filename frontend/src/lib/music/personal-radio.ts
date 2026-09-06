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
}

async function getCachedSimilarSongs(
  state: PersonalRadioState,
  fetchers: PersonalRadioFetchers,
  trackId: string,
  count: number,
): Promise<SubsonicSong[]> {
  const cached = state.similarCache.get(trackId);
  if (cached) return cached;
  const songs = await fetchers.getSimilarSongs(trackId, count).catch(() => []);
  state.similarCache.set(trackId, songs);
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

  const random = seededRandom(state.sessionSeed);
  const pool: SubsonicSong[] = [];
  const nowMs = Date.now();
  const coldStart = options.coldStart;

  const seedIds = pickQualitySeedIds(
    history,
    stats,
    6,
    state.lastCompletedId,
    nowMs,
  );

  state.seedTrackIds = seedIds.filter(Boolean);

  const similarSeedIds = [
    ...new Set(
      [
        ...(state.lastCompletedId ? [state.lastCompletedId] : []),
        ...seedIds.slice(0, 5),
      ].filter(Boolean),
    ),
  ].slice(0, 5);

  const topArtists = stats?.topArtists.slice(0, coldStart ? 2 : 5) ?? [];
  const randomCount = coldStart ? Math.max(count, 40) : Math.max(count, 20);

  const moodTracks =
    state.recentCompletions.length > 0
      ? state.recentCompletions
      : history.slice(0, 5).map(entryToSong);
  const sessionMood = buildSessionMood(moodTracks, coldStart ? 0.35 : 1);

  const [similarGroups, artistSongs, randomSongs] = await Promise.all([
    Promise.all(
      similarSeedIds.map((id) =>
        getCachedSimilarSongs(
          state,
          fetchers,
          id,
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

  const filtered = pool.filter((track) => {
    const id = track.id?.trim();
    if (!id || exclude.has(track.id) || exclude.has(id)) return false;
    if (recentIds.has(id) && id !== state.lastCompletedId && !coldStart) {
      return false;
    }
    const key = artistKey(track);
    if (key && (state.skipArtistKeys.get(key) ?? 0) >= 3) return false;
    return true;
  });

  const tasteOptions = coldStart
    ? {
        exploreBonus: Math.max(1, options.exploreBonus + 0.4),
        sessionMood,
      }
    : { exploreBonus: options.exploreBonus, sessionMood };

  const sampled = weightedSampleTracks(
    filtered,
    profile,
    count,
    random,
    tasteOptions,
    exclude,
  );

  const interleaved = interleaveByArtist(
    sampled,
    `${state.sessionSeed}:interleave`,
  );
  const ordered = orderForFlow(interleaved, `${state.sessionSeed}:flow`, {
    albumLookback: options.albumLookback,
  });

  return ordered.slice(0, count);
}

export function wasTrackSkipped(
  positionMs: number,
  durationMs: number,
): boolean {
  if (durationMs <= 0) return positionMs < 30_000;
  return positionMs / durationMs < 0.5;
}
