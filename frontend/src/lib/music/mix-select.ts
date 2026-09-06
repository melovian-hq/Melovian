// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  decadeFromYear,
  scoreTrackTaste,
  type TasteProfile,
  type TasteScoreOptions,
} from "./taste-score";
import type { MixSettings } from "./mix-settings";
import type { SubsonicSong } from "$lib/subsonic/types";

export interface FlowOptions {
  durationPacing?: boolean;
  albumLookback?: number;
  artistFocused?: boolean;
}

export interface MixSeedProfile {
  artistKeys: ReadonlySet<string>;
  genreKeys: ReadonlySet<string>;
  decades: ReadonlySet<number>;
  artistFocused: boolean;
}

export interface MixSelectContext {
  playCountByTrack: ReadonlyMap<string, number>;
  listenedMsByTrack: ReadonlyMap<string, number>;
  skippedTrackIds: ReadonlySet<string>;
  recentlyPlayedIds: ReadonlySet<string>;
  recentMixTrackIds: ReadonlySet<string>;
  settings: MixSettings;
  tasteProfile: TasteProfile;
  starredTrackIds: ReadonlySet<string>;
}

export interface MixSelectPolicy {
  targetCount: number;
  maxPerArtist: number;
  maxPerAlbum: number;
  familiarRatio: number;
  discoverRatio: number;
  excludeSkipped: boolean;
  penalizeRecent: boolean;
  exploreBonus: number;
}

export interface ArtistFeature {
  label: string;
  key: string;
  weight: number;
  genres: string[];
  decades: number[];
}

export interface ArtistCluster {
  artists: ArtistFeature[];
  genreKeys: string[];
  decades: number[];
}

export type MixLane = "familiar" | "discover" | "deep";

function hashString(value: string): number {
  let hash = 2166136261;
  for (let i = 0; i < value.length; i++) {
    hash ^= value.charCodeAt(i);
    hash = Math.imul(hash, 16777619);
  }
  return hash >>> 0;
}

export function createSeededRandom(seed: string): () => number {
  let state = hashString(seed) || 1;
  return () => {
    state = Math.imul(state ^ (state >>> 15), state | 1);
    state ^= state + Math.imul(state ^ (state >>> 7), state | 61);
    return ((state ^ (state >>> 14)) >>> 0) / 4294967296;
  };
}

export function normalizeMixKey(value: string | undefined | null): string {
  return value?.trim().toLowerCase() ?? "";
}

export function trackArtistKey(track: SubsonicSong): string {
  return normalizeMixKey(track.artist) || track.id;
}

export function trackAlbumKey(track: SubsonicSong): string {
  return track.albumId ?? (normalizeMixKey(track.album) || track.id);
}

export function trackGenreKey(track: SubsonicSong): string {
  return normalizeMixKey(track.genre);
}

export function emptyMixSeed(): MixSeedProfile {
  return {
    artistKeys: new Set(),
    genreKeys: new Set(),
    decades: new Set(),
    artistFocused: false,
  };
}

export function seedFromCluster(cluster: ArtistCluster): MixSeedProfile {
  return {
    artistKeys: new Set(cluster.artists.map((artist) => artist.key)),
    genreKeys: new Set(cluster.genreKeys),
    decades: new Set(cluster.decades),
    artistFocused: cluster.artists.length <= 1,
  };
}

export function defaultMixPolicy(
  targetCount: number,
  artistFocused = false,
): MixSelectPolicy {
  return {
    targetCount,
    maxPerArtist: artistFocused ? 10 : 4,
    maxPerAlbum: artistFocused ? 4 : 2,
    familiarRatio: 0.55,
    discoverRatio: 0.3,
    excludeSkipped: true,
    penalizeRecent: true,
    exploreBonus: 0.4,
  };
}

export function dedupeTracks(tracks: readonly SubsonicSong[]): SubsonicSong[] {
  const seen = new Set<string>();
  const result: SubsonicSong[] = [];
  for (const track of tracks) {
    if (!track.id || seen.has(track.id)) continue;
    seen.add(track.id);
    result.push(track);
  }
  return result;
}

export function shuffleWithSeed<T>(items: readonly T[], seed: string): T[] {
  const copy = [...items];
  const random = createSeededRandom(seed);
  for (let i = copy.length - 1; i > 0; i--) {
    const j = Math.floor(random() * (i + 1));
    [copy[i], copy[j]] = [copy[j], copy[i]];
  }
  return copy;
}

export function interleaveByArtist(
  tracks: readonly SubsonicSong[],
  seed?: string,
): SubsonicSong[] {
  const buckets = new Map<string, SubsonicSong[]>();
  for (const track of tracks) {
    const key = trackArtistKey(track);
    const bucket = buckets.get(key) ?? [];
    bucket.push(track);
    buckets.set(key, bucket);
  }

  const keys = [...buckets.keys()];
  const queueKeys = seed ? shuffleWithSeed(keys, seed) : keys.sort();
  const queues = queueKeys.map((key) => buckets.get(key)!);
  const result: SubsonicSong[] = [];
  let added = true;
  while (added) {
    added = false;
    for (const queue of queues) {
      const next = queue.shift();
      if (next) {
        result.push(next);
        added = true;
      }
    }
  }
  return result;
}

export function orderForFlow(
  tracks: readonly SubsonicSong[],
  seed = "flow",
  options: FlowOptions = {},
): SubsonicSong[] {
  if (tracks.length <= 2) return [...tracks];

  const remaining = dedupeTracks(tracks);
  const random = createSeededRandom(seed);
  const ordered: SubsonicSong[] = [];
  const albumLookback = options.albumLookback ?? 4;
  const artistFocused = options.artistFocused ?? false;
  const artistLookback = artistFocused ? 1 : 3;

  const pickBest = (candidates: SubsonicSong[]): SubsonicSong => {
    const last = ordered.at(-1);
    const lastArtist = last ? trackArtistKey(last) : "";
    const lastGenre = last ? trackGenreKey(last) : "";
    const lastDecade =
      last && typeof last.year === "number" && last.year >= 1900
        ? decadeFromYear(last.year)
        : null;

    let pool = candidates;
    if (!artistFocused && lastArtist) {
      const differentArtist = pool.filter(
        (track) => trackArtistKey(track) !== lastArtist,
      );
      if (differentArtist.length > 0) pool = differentArtist;
    }

    if (ordered.length > 0) {
      const recentAlbums = new Set<string>();
      for (let i = 1; i <= Math.min(2, ordered.length); i++) {
        const prev = ordered.at(-i);
        if (prev) recentAlbums.add(trackAlbumKey(prev));
      }
      const differentAlbum = pool.filter(
        (track) => !recentAlbums.has(trackAlbumKey(track)),
      );
      if (differentAlbum.length > 0) pool = differentAlbum;
    }

    let best = pool[0];
    let bestScore = Number.NEGATIVE_INFINITY;
    for (const candidate of pool) {
      let score = random() * 0.01;
      const artist = trackArtistKey(candidate);
      const album = trackAlbumKey(candidate);
      const genre = trackGenreKey(candidate);
      const secondLast = ordered.at(-2);

      for (let i = 1; i <= Math.min(artistLookback, ordered.length); i++) {
        const prev = ordered.at(-i);
        if (!prev || !artist) continue;
        if (artist === trackArtistKey(prev)) {
          score -= 5 / i;
        }
      }
      if (secondLast && artist && artist === trackArtistKey(secondLast)) {
        score -= 2.5;
      }

      for (let i = 1; i <= albumLookback; i++) {
        const prev = ordered.at(-i);
        if (prev && album && album === trackAlbumKey(prev)) {
          score -= 3.2 / i;
        }
      }

      if (options.durationPacing && last?.duration && candidate.duration) {
        const ratio = candidate.duration / last.duration;
        if (ratio > 1.35 || ratio < 0.72) score += 0.75;
        else score -= 0.35;

        // Avoid three near-identical lengths in a row.
        if (secondLast?.duration) {
          const prevRatio = last.duration / secondLast.duration;
          const sameBand =
            prevRatio >= 0.85 &&
            prevRatio <= 1.18 &&
            ratio >= 0.85 &&
            ratio <= 1.18;
          if (sameBand) score -= 0.9;
        }
      }

      if (genre && lastGenre) {
        if (genresRelated(genre, lastGenre)) score += 0.7;
        else score -= 0.35;
      }
      if (
        lastDecade !== null &&
        typeof candidate.year === "number" &&
        candidate.year >= 1900
      ) {
        const decadeGap = Math.abs(decadeFromYear(candidate.year) - lastDecade);
        if (decadeGap === 0) score += 0.35;
        else if (decadeGap <= 10) score += 0.1;
        else score -= 0.2;
      }

      // Prefer opening with a familiar-feeling middle duration when possible.
      if (ordered.length === 0 && candidate.duration) {
        if (candidate.duration >= 160 && candidate.duration <= 280)
          score += 0.2;
      }

      if (score > bestScore) {
        bestScore = score;
        best = candidate;
      }
    }
    return best;
  };

  const starterIndex = Math.floor(random() * remaining.length);
  const starter = remaining[starterIndex];
  ordered.push(starter);
  remaining.splice(remaining.indexOf(starter), 1);

  while (remaining.length > 0) {
    const next = pickBest(remaining);
    ordered.push(next);
    remaining.splice(remaining.indexOf(next), 1);
  }

  return ordered;
}

function uniqueStrings(values: readonly string[]): string[] {
  const seen = new Set<string>();
  const result: string[] = [];
  for (const value of values) {
    const key = normalizeMixKey(value);
    if (!key || seen.has(key)) continue;
    seen.add(key);
    result.push(key);
  }
  return result;
}

function genreTokens(genre: string): Set<string> {
  return new Set(
    genre
      .split(/[\s/&,+-]+/)
      .map((token) => token.trim())
      .filter((token) => token.length > 2),
  );
}

/** Soft genre match: exact, containment, or shared tokens (indie rock / rock). */
export function genresRelated(a: string, b: string): boolean {
  const left = normalizeMixKey(a);
  const right = normalizeMixKey(b);
  if (!left || !right) return false;
  if (left === right) return true;
  if (left.includes(right) || right.includes(left)) return true;
  const tokensA = genreTokens(left);
  const tokensB = genreTokens(right);
  for (const token of tokensA) {
    if (tokensB.has(token)) return true;
  }
  return false;
}

export function artistSimilarity(a: ArtistFeature, b: ArtistFeature): number {
  if (a.key === b.key) return 0;
  let score = 0;
  const genresA = uniqueStrings(a.genres);
  const genresB = uniqueStrings(b.genres);
  let genreOverlap = 0;
  let softOverlap = 0;
  for (const genreA of genresA) {
    for (const genreB of genresB) {
      if (genreA === genreB) genreOverlap += 1;
      else if (genresRelated(genreA, genreB)) softOverlap += 1;
    }
  }
  const genreUnion = new Set([...genresA, ...genresB]).size;
  if (genreUnion > 0) {
    score += (genreOverlap / genreUnion) * 6;
    score += (softOverlap / genreUnion) * 2.5;
  } else {
    score += 0.4;
  }

  const decadesA = [...new Set(a.decades)];
  const decadesB = [...new Set(b.decades)];
  let minDecadeGap = Number.POSITIVE_INFINITY;
  for (const decadeA of decadesA) {
    for (const decadeB of decadesB) {
      minDecadeGap = Math.min(minDecadeGap, Math.abs(decadeA - decadeB));
    }
  }
  if (Number.isFinite(minDecadeGap)) {
    if (minDecadeGap === 0) score += 2.2;
    else if (minDecadeGap <= 10) score += 1.1;
    else if (minDecadeGap <= 20) score += 0.35;
  }

  // Weight proximity keeps heavy favorites from isolating alone.
  const weightRatio =
    Math.min(a.weight, b.weight) / Math.max(a.weight, b.weight, 1);
  score += weightRatio * 0.4;

  if (
    genreOverlap === 0 &&
    softOverlap === 0 &&
    genresA.length > 0 &&
    genresB.length > 0
  ) {
    return score * 0.45;
  }
  return score;
}

function clusterCentroidScore(
  member: ArtistFeature,
  cluster: readonly ArtistFeature[],
): number {
  if (cluster.length === 0) return 0;
  let total = 0;
  for (const other of cluster) {
    total += artistSimilarity(member, other);
  }
  return total / cluster.length;
}

export function clusterArtistFeatures(
  features: readonly ArtistFeature[],
  seed: string,
  clusterCount: number,
  perCluster: number,
): ArtistCluster[] {
  if (features.length === 0 || clusterCount <= 0) return [];

  const random = createSeededRandom(seed);
  const remaining = [...features].sort((a, b) => {
    if (b.weight !== a.weight) return b.weight - a.weight;
    return a.key.localeCompare(b.key);
  });

  const groups: ArtistFeature[][] = [];
  const used = new Set<string>();

  const takeNextSeed = (): ArtistFeature | null => {
    for (const feature of remaining) {
      if (!used.has(feature.key)) return feature;
    }
    return null;
  };

  while (groups.length < clusterCount) {
    const start = takeNextSeed();
    if (!start) break;
    const group: ArtistFeature[] = [start];
    used.add(start.key);

    while (group.length < perCluster) {
      let best: ArtistFeature | null = null;
      let bestScore = Number.NEGATIVE_INFINITY;
      for (const candidate of remaining) {
        if (used.has(candidate.key)) continue;
        const sim = clusterCentroidScore(candidate, group) + random() * 0.02;
        if (sim > bestScore) {
          bestScore = sim;
          best = candidate;
        }
      }
      if (!best) break;
      if (bestScore < 0.85 && group.length >= 2) break;
      group.push(best);
      used.add(best.key);
    }
    groups.push(group);
  }

  return groups.map((artists) => {
    const genreCounts = new Map<string, number>();
    const decadeCounts = new Map<number, number>();
    for (const artist of artists) {
      for (const genre of uniqueStrings(artist.genres)) {
        genreCounts.set(genre, (genreCounts.get(genre) ?? 0) + 1);
      }
      for (const decade of artist.decades) {
        decadeCounts.set(decade, (decadeCounts.get(decade) ?? 0) + 1);
      }
    }
    const genreKeys = [...genreCounts.entries()]
      .sort((a, b) => b[1] - a[1])
      .map(([genre]) => genre);
    const decades = [...decadeCounts.entries()]
      .sort((a, b) => b[1] - a[1])
      .map(([decade]) => decade);
    return { artists, genreKeys, decades };
  });
}

export function classifyMixLane(
  track: SubsonicSong,
  ctx: MixSelectContext,
  seed: MixSeedProfile,
): MixLane {
  const plays = ctx.playCountByTrack.get(track.id) ?? 0;
  const listened = ctx.listenedMsByTrack.get(track.id) ?? 0;
  const artist = normalizeMixKey(track.artist);
  const inCluster = artist.length > 0 && seed.artistKeys.has(artist);
  if (plays === 0 && listened < 90_000) {
    return inCluster ? "deep" : "discover";
  }
  if (plays <= ctx.settings.deepCutMaxPlayCount && inCluster) return "deep";
  return "familiar";
}

export function mixFitScore(
  track: SubsonicSong,
  ctx: MixSelectContext,
  seed: MixSeedProfile,
  policy: MixSelectPolicy,
): number {
  if (policy.excludeSkipped && ctx.skippedTrackIds.has(track.id)) return -999;

  const options: TasteScoreOptions = {
    exploreBonus: policy.exploreBonus,
    recentMixTrackIds: ctx.recentMixTrackIds,
    starredTrackIds: ctx.starredTrackIds,
  };
  let score = scoreTrackTaste(track, ctx.tasteProfile, options);

  const artist = normalizeMixKey(track.artist);
  if (artist && seed.artistKeys.has(artist)) score += 8;
  else if (artist && seed.artistKeys.size > 0) score += 1.2;

  const genre = trackGenreKey(track);
  if (genre && seed.genreKeys.size > 0) {
    let matched = false;
    let soft = false;
    for (const seedGenre of seed.genreKeys) {
      if (genre === seedGenre) {
        matched = true;
        break;
      }
      if (genresRelated(genre, seedGenre)) soft = true;
    }
    if (matched) score += 5;
    else if (soft) score += 2.4;
    else score -= 2.1;
  }

  if (typeof track.year === "number" && track.year >= 1900) {
    const decade = decadeFromYear(track.year);
    if (seed.decades.has(decade)) score += 2;
    else if (seed.decades.size > 0) {
      let closest = Number.POSITIVE_INFINITY;
      for (const seedDecade of seed.decades) {
        closest = Math.min(closest, Math.abs(seedDecade - decade));
      }
      if (closest <= 10) score += 0.8;
    }
  }

  // Starred already boosts taste score. Keep a light extra nudge only.
  if (ctx.starredTrackIds.has(track.id)) score += 0.8;
  if (policy.penalizeRecent && ctx.recentlyPlayedIds.has(track.id))
    score -= 5.5;
  if (ctx.recentMixTrackIds.has(track.id)) score -= 4;

  return score;
}

function uniqueArtistCount(tracks: readonly SubsonicSong[]): number {
  const keys = new Set<string>();
  for (const track of tracks) keys.add(trackArtistKey(track));
  return keys.size;
}

function adaptiveArtistCap(
  tracks: readonly SubsonicSong[],
  policy: MixSelectPolicy,
): number {
  const artists = uniqueArtistCount(tracks);
  if (artists <= 1) return Math.max(policy.maxPerArtist, policy.targetCount);
  const spread = Math.ceil(policy.targetCount / artists);
  return Math.max(policy.maxPerArtist, spread);
}

function pickSoftmax(
  items: { track: SubsonicSong; weight: number }[],
  random: () => number,
  temperature = 0.9,
): number {
  if (items.length === 1) return 0;
  const maxWeight = Math.max(...items.map((item) => item.weight));
  const scaled = items.map((item) =>
    Math.exp(Math.min(14, (item.weight - maxWeight) / temperature)),
  );
  const total = scaled.reduce((sum, value) => sum + value, 0);
  let roll = random() * total;
  for (let i = 0; i < scaled.length; i++) {
    roll -= scaled[i];
    if (roll <= 0) return i;
  }
  return items.length - 1;
}

/**
 * Taste-weighted sample with artist/album caps and a familiarity mix.
 * Caps relax when the library is too small to fill the target otherwise.
 */
export function selectMixTracks(
  tracks: readonly SubsonicSong[],
  ctx: MixSelectContext,
  seed: MixSeedProfile,
  policy: MixSelectPolicy,
  rngSeed: string,
): SubsonicSong[] {
  const pool = dedupeTracks(tracks).filter((track) => {
    if (!track.id) return false;
    if (policy.excludeSkipped && ctx.skippedTrackIds.has(track.id)) {
      return false;
    }
    return true;
  });
  if (pool.length === 0) return [];

  const random = createSeededRandom(rngSeed);
  const artistCap = adaptiveArtistCap(pool, policy);
  const albumCap = policy.maxPerAlbum;
  const target = Math.min(policy.targetCount, pool.length);
  const familiarNeed = Math.round(target * policy.familiarRatio);
  const discoverNeed = Math.round(target * policy.discoverRatio);

  const remaining = pool.map((track) => ({
    track,
    lane: classifyMixLane(track, ctx, seed),
    score: mixFitScore(track, ctx, seed, policy),
  }));

  const picks: SubsonicSong[] = [];
  const artistCounts = new Map<string, number>();
  const albumCounts = new Map<string, number>();
  let familiarPicked = 0;
  let discoverPicked = 0;
  let deepPicked = 0;

  const underCaps = (track: SubsonicSong, relax: 0 | 1 | 2): boolean => {
    const artist = trackArtistKey(track);
    const album = trackAlbumKey(track);
    if (relax < 2 && (artistCounts.get(artist) ?? 0) >= artistCap) {
      return false;
    }
    if (relax < 1 && (albumCounts.get(album) ?? 0) >= albumCap) {
      return false;
    }
    return true;
  };

  while (picks.length < target && remaining.length > 0) {
    let eligible = remaining.filter((item) => underCaps(item.track, 0));
    if (eligible.length === 0) {
      eligible = remaining.filter((item) => underCaps(item.track, 1));
    }
    if (eligible.length === 0) eligible = remaining;
    const use = eligible;

    const needFamiliar = familiarPicked < familiarNeed;
    const needDiscover = discoverPicked < discoverNeed;
    const deepNeed = Math.max(
      0,
      target - familiarNeed - discoverNeed - deepPicked,
    );
    const needDeep = deepNeed > 0;

    const weighted = use.map((item) => {
      let weight = item.score + 2;
      const artist = trackArtistKey(item.track);
      const album = trackAlbumKey(item.track);
      const artistCount = artistCounts.get(artist) ?? 0;
      const albumCount = albumCounts.get(album) ?? 0;
      // Soft diversity on top of hard caps.
      weight *= 1 / (1 + artistCount * 1.4);
      weight *= 1 / (1 + albumCount * 0.95);

      if (needFamiliar && item.lane === "familiar") weight *= 2.5;
      if (needDiscover && item.lane === "discover") weight *= 2.2;
      if (needDeep && item.lane === "deep") weight *= 2;
      if (!needFamiliar && !needDiscover && item.lane === "deep") {
        weight *= 1.55;
      }
      return { track: item.track, weight, lane: item.lane };
    });

    const index = pickSoftmax(weighted, random, 0.88);
    const chosen = weighted[index];
    picks.push(chosen.track);
    if (chosen.lane === "familiar") familiarPicked += 1;
    if (chosen.lane === "discover") discoverPicked += 1;
    if (chosen.lane === "deep") deepPicked += 1;
    const artist = trackArtistKey(chosen.track);
    const album = trackAlbumKey(chosen.track);
    artistCounts.set(artist, (artistCounts.get(artist) ?? 0) + 1);
    albumCounts.set(album, (albumCounts.get(album) ?? 0) + 1);

    const remainingIndex = remaining.findIndex(
      (item) => item.track.id === chosen.track.id,
    );
    if (remainingIndex >= 0) remaining.splice(remainingIndex, 1);
  }

  return picks;
}

export function tracksMatchSeed(
  track: SubsonicSong,
  seed: MixSeedProfile,
): boolean {
  if (seed.artistKeys.size === 0 && seed.genreKeys.size === 0) return true;
  const artist = normalizeMixKey(track.artist);
  if (artist && seed.artistKeys.has(artist)) return true;
  const genre = trackGenreKey(track);
  if (genre && seed.genreKeys.has(genre)) return true;
  if (!artist && !genre) return true;
  return false;
}
