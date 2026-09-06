// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { genrePalette } from "../genre-art";
import {
  detectTrackLanguage,
  languageMatchScore,
  topLanguages,
} from "../mix-language";
import type { MixSettings } from "../mix-settings";
import {
  createSeededRandom,
  dedupeTracks,
  defaultMixPolicy,
  emptyMixSeed,
  normalizeMixKey,
  orderForFlow,
  selectMixTracks,
  shuffleWithSeed,
  type ArtistCluster,
  type MixSeedProfile,
  type MixSelectPolicy,
} from "../mix-select";
import {
  decadeFromYear,
  listenAffinityWeight,
  rankedDecadeKeys,
} from "../taste-score";
import type {
  ListenEntry,
  ListenStats,
  SubsonicAlbum,
  SubsonicSong,
} from "$lib/subsonic/types";
import { selectContext } from "./context";
import type { GeneratedMix, MixBuildContext, MixBuildState } from "./types";

export function hashString(value: string): number {
  let hash = 2166136261;
  for (let i = 0; i < value.length; i++) {
    hash ^= value.charCodeAt(i);
    hash = Math.imul(hash, 16777619);
  }
  return hash >>> 0;
}

export function gradientFromGenre(name: string): string {
  const [bg, accent, mid] = genrePalette(name);
  return `linear-gradient(135deg, ${bg} 0%, ${mid} 50%, ${accent} 100%)`;
}

export function gradientFromSeed(seed: string): string {
  const hash = hashString(seed);
  const hue = hash % 360;
  return `linear-gradient(135deg, hsl(${hue} 35% 8%) 0%, hsl(${(hue + 30) % 360} 45% 22%) 50%, hsl(${(hue + 60) % 360} 55% 38%) 100%)`;
}

/** Prefer unused artwork so mix cards do not all share one album cover. */
export function pickUniqueMixCover(
  tracks: readonly SubsonicSong[],
  preferred: string | undefined,
  usedCoverArtIds: Set<string>,
  seed: string,
): string | undefined {
  const candidates: string[] = [];
  const push = (value: string | undefined) => {
    const id = value?.trim();
    if (!id || candidates.includes(id)) return;
    candidates.push(id);
  };
  push(preferred);
  for (const track of tracks) {
    push(track.coverArt);
    push(track.albumId);
  }
  if (candidates.length === 0) return undefined;

  const unused = candidates.filter((id) => !usedCoverArtIds.has(id));
  const pool = unused.length > 0 ? unused : candidates;
  const random = createSeededRandom(seed);
  const pick = pool[Math.floor(random() * pool.length)] ?? pool[0];
  if (pick) usedCoverArtIds.add(pick);
  return pick;
}

export function pickAlbumCover(
  albums: readonly SubsonicAlbum[],
  seed: string,
): string | undefined {
  if (albums.length === 0) return undefined;
  const withArt = albums
    .map((album) => album.coverArt?.trim())
    .filter((id): id is string => !!id);
  if (withArt.length === 0) return undefined;
  const random = createSeededRandom(seed);
  return withArt[Math.floor(random() * withArt.length)] ?? withArt[0];
}

function excludeUsed(
  tracks: readonly SubsonicSong[],
  state: MixBuildState,
  settings: MixSettings,
): SubsonicSong[] {
  if (!settings.crossMixDedup) return [...tracks];
  return tracks.filter((track) => !state.usedTrackIds.has(track.id));
}

function applyLanguagePool(
  tracks: readonly SubsonicSong[],
  ctx: MixBuildContext,
  state: MixBuildState,
  minNeeded: number,
): SubsonicSong[] {
  if (ctx.settings.languageBias === "off" || tracks.length === 0) {
    return [...tracks];
  }

  const scored = tracks.map((track) => ({
    track,
    score: languageMatchScore(track, state.languageWeights),
  }));
  scored.sort((a, b) => b.score - a.score);

  if (ctx.settings.languageBias === "strict") {
    const langs = topLanguages(state.languageWeights, 2);
    const strict = scored
      .filter(({ track, score }) => {
        if (score > 0) return true;
        const lang = detectTrackLanguage(track);
        return langs.includes(lang);
      })
      .map(({ track }) => track);
    if (strict.length >= minNeeded) return strict;
  }

  return scored.map(({ track }) => track);
}

function reserveTracks(
  tracks: readonly SubsonicSong[],
  state: MixBuildState,
  settings: MixSettings,
) {
  if (!settings.crossMixDedup) return;
  for (const track of tracks) {
    state.usedTrackIds.add(track.id);
  }
}

interface FinalizeMixOptions {
  maxTracks?: number;
  skipCrossMixDedup?: boolean;
  seed?: MixSeedProfile;
  policy?: Partial<MixSelectPolicy>;
}

export function finalizeMix(
  mix: Omit<GeneratedMix, "coverArtId" | "tracks"> & { coverArtId?: string },
  tracks: SubsonicSong[],
  flowSeed: string,
  ctx: MixBuildContext,
  state: MixBuildState,
  options?: FinalizeMixOptions,
): GeneratedMix | null {
  const minTracks = ctx.settings.minTracksPerMix;
  const maxTracks = options?.maxTracks ?? ctx.settings.maxTracksPerMix;
  const skipCrossMixDedup = options?.skipCrossMixDedup ?? false;
  const seed = options?.seed ?? emptyMixSeed();
  const policy = {
    ...defaultMixPolicy(maxTracks, seed.artistFocused),
    ...options?.policy,
    targetCount: maxTracks,
  };

  let pool = skipCrossMixDedup
    ? dedupeTracks(tracks)
    : excludeUsed(tracks, state, ctx.settings);
  pool = applyLanguagePool(pool, ctx, state, minTracks);
  pool = selectMixTracks(
    pool,
    selectContext(ctx, state),
    seed,
    policy,
    `${flowSeed}:select`,
  );
  if (pool.length < minTracks) return null;

  const ordered = orderForFlow(pool, flowSeed, {
    durationPacing: ctx.settings.durationPacing,
    albumLookback: ctx.settings.flowAlbumLookback,
    artistFocused: seed.artistFocused,
  });
  if (ordered.length < minTracks) return null;

  const finalTracks = ordered.slice(0, maxTracks);
  if (!skipCrossMixDedup) {
    reserveTracks(finalTracks, state, ctx.settings);
  }

  return {
    ...mix,
    tracks: finalTracks,
    coverArtId: pickUniqueMixCover(
      finalTracks,
      mix.coverArtId,
      state.usedCoverArtIds,
      `${flowSeed}:cover`,
    ),
  };
}

export function historyByListenWeight(
  history: readonly ListenEntry[],
): ListenEntry[] {
  const byTrack = new Map<string, ListenEntry>();
  for (const entry of history) {
    const existing = byTrack.get(entry.trackId);
    const weight = listenAffinityWeight(entry);
    const existingWeight = existing ? listenAffinityWeight(existing) : -1;
    if (!existing || weight > existingWeight) {
      byTrack.set(entry.trackId, entry);
    }
  }
  return [...byTrack.values()].sort(
    (a, b) => listenAffinityWeight(b) - listenAffinityWeight(a),
  );
}

export function weightedSimilarSeeds(
  stats: ListenStats | null,
  history: readonly ListenEntry[],
  seed: string,
  count: number,
): string[] {
  const ranked =
    stats?.topTracks.slice(0, 10).map((track) => ({
      id: track.key,
      weight: Math.max(1, track.count),
    })) ??
    historyByListenWeight(history)
      .slice(0, 10)
      .map((entry) => ({
        id: entry.trackId,
        weight: Math.max(1, listenAffinityWeight(entry)),
      }));

  if (ranked.length === 0) return [];

  const random = createSeededRandom(seed);
  const picks: string[] = [];
  const pool = [...ranked];

  while (picks.length < count && pool.length > 0) {
    const total = pool.reduce((sum, item) => sum + item.weight, 0);
    let roll = random() * total;
    let index = 0;
    for (let i = 0; i < pool.length; i++) {
      roll -= pool[i].weight;
      if (roll <= 0) {
        index = i;
        break;
      }
    }
    picks.push(pool[index].id);
    pool.splice(index, 1);
  }

  return picks;
}

export function isDiscoverCandidate(
  track: SubsonicSong,
  ctx: MixBuildContext,
): boolean {
  if (ctx.skippedTrackIds.has(track.id)) return false;
  if (ctx.recentlyPlayedIds.has(track.id)) return false;
  if (ctx.recentMixTrackIds.has(track.id)) return false;
  const plays = ctx.playCountByTrack.get(track.id) ?? 0;
  const listenedMs = ctx.listenedMsByTrack.get(track.id) ?? 0;
  return plays <= 2 && listenedMs < 120_000;
}

export function dominantDecade(ctx: MixBuildContext): number | null {
  return rankedDecades(ctx, 1)[0] ?? null;
}

/** Ranked release decades from taste affinity and frequent albums. */
export function rankedDecades(ctx: MixBuildContext, limit: number): number[] {
  const counts = new Map<number, number>();
  for (const [decade, weight] of ctx.tasteProfile.decadeAffinity) {
    counts.set(decade, (counts.get(decade) ?? 0) + weight);
  }
  for (const album of ctx.frequentAlbums) {
    if (typeof album.year !== "number" || album.year < 1900) continue;
    const decade = decadeFromYear(album.year);
    counts.set(decade, (counts.get(decade) ?? 0) + 2);
  }
  return rankedDecadeKeys(counts, limit);
}

export function albumYearIndex(
  albums: readonly SubsonicAlbum[],
): Map<string, number> {
  const years = new Map<string, number>();
  for (const album of albums) {
    if (typeof album.year !== "number" || album.year < 1900) continue;
    if (!album.id) continue;
    years.set(album.id, album.year);
  }
  return years;
}

export function trackMatchesDecade(
  track: SubsonicSong,
  decade: number,
  albumYears: ReadonlyMap<string, number>,
): boolean {
  if (typeof track.year === "number" && track.year >= 1900) {
    return decadeFromYear(track.year) === decade;
  }
  if (track.albumId) {
    const year = albumYears.get(track.albumId);
    if (typeof year === "number") return decadeFromYear(year) === decade;
  }
  return false;
}

export function seedFromArtists(
  labels: readonly string[],
  genres: readonly string[] = [],
  decades: readonly number[] = [],
  artistFocused = false,
): MixSeedProfile {
  return {
    artistKeys: new Set(labels.map(normalizeMixKey).filter(Boolean)),
    genreKeys: new Set(genres.map(normalizeMixKey).filter(Boolean)),
    decades: new Set(decades),
    artistFocused,
  };
}

export function topArtistLabels(ctx: MixBuildContext, limit: number): string[] {
  const fromStats = (ctx.stats?.topArtists ?? [])
    .map((artist) => artist.label)
    .filter((label) => label.trim().length > 0);
  if (fromStats.length >= 2) return fromStats.slice(0, limit);

  const counts = new Map<string, number>();
  for (const entry of ctx.history) {
    const label = entry.artistName?.trim();
    if (!label) continue;
    counts.set(label, (counts.get(label) ?? 0) + Math.max(1, entry.playCount));
  }
  return [...counts.entries()]
    .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
    .slice(0, limit)
    .map(([label]) => label);
}

export function clusterSeedIds(
  ctx: MixBuildContext,
  cluster: ArtistCluster,
  seed: string,
  count: number,
): string[] {
  const artistKeys = new Set(cluster.artists.map((artist) => artist.key));
  const ranked = historyByListenWeight(ctx.history)
    .filter((entry) => artistKeys.has(normalizeMixKey(entry.artistName)))
    .map((entry) => ({
      id: entry.trackId,
      weight: Math.max(1, listenAffinityWeight(entry)),
    }));
  if (ranked.length === 0) {
    return weightedSimilarSeeds(ctx.stats, ctx.history, seed, count);
  }
  const random = createSeededRandom(seed);
  const picks: string[] = [];
  const pool = [...ranked];
  while (picks.length < count && pool.length > 0) {
    const total = pool.reduce((sum, item) => sum + item.weight, 0);
    let roll = random() * total;
    let index = 0;
    for (let i = 0; i < pool.length; i++) {
      roll -= pool[i].weight;
      if (roll <= 0) {
        index = i;
        break;
      }
    }
    picks.push(pool[index].id);
    pool.splice(index, 1);
  }
  return picks;
}
export function pickDeepCutTracks(
  songs: SubsonicSong[],
  ctx: MixBuildContext,
  albumSeed: string,
): SubsonicSong[] {
  const eligible = songs.filter((song) => {
    const plays = ctx.playCountByTrack.get(song.id) ?? 0;
    return plays <= ctx.settings.deepCutMaxPlayCount;
  });
  if (eligible.length === 0) return [];

  const sorted = [...eligible].sort((a, b) => {
    const playA = ctx.playCountByTrack.get(a.id) ?? 0;
    const playB = ctx.playCountByTrack.get(b.id) ?? 0;
    if (playA !== playB) return playA - playB;

    const trackA = a.track ?? 999;
    const trackB = b.track ?? 999;
    const midA = Math.abs(trackA - 4);
    const midB = Math.abs(trackB - 4);
    return midA - midB;
  });

  return shuffleWithSeed(sorted, albumSeed).slice(0, 3);
}
export function dedupeAlbums(
  albums: readonly SubsonicAlbum[],
): SubsonicAlbum[] {
  const seen = new Set<string>();
  const result: SubsonicAlbum[] = [];
  for (const album of albums) {
    if (!album.id || seen.has(album.id)) continue;
    seen.add(album.id);
    result.push(album);
  }
  return result;
}
