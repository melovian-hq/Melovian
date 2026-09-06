// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type {
  ListenEntry,
  ListenStats,
  SubsonicSong,
} from "$lib/subsonic/types";

export interface TasteProfile {
  artistAffinity: Map<string, number>;
  albumAffinity: Map<string, number>;
  genreAffinity: Map<string, number>;
  decadeAffinity: Map<number, number>;
  /** Artist transitions heard within a listen session. Key is "from|to". */
  followArtistAffinity: Map<string, number>;
  playCountByTrack: Map<string, number>;
  listenedMsByTrack: Map<string, number>;
  durationMsByTrack: Map<string, number>;
  lastPlayedAt: Map<string, number>;
  skippedTrackIds: ReadonlySet<string>;
  topArtistKeys: string[];
}

/** Recent completions that steer the next pick toward the current mood. */
export interface SessionMood {
  artistWeights: ReadonlyMap<string, number>;
  genreWeights: ReadonlyMap<string, number>;
  albumWeights: ReadonlyMap<string, number>;
  strength: number;
}

export interface TasteScoreOptions {
  nowMs?: number;
  recentMs?: number;
  exploreBonus?: number;
  recentMixTrackIds?: ReadonlySet<string>;
  starredTrackIds?: ReadonlySet<string>;
  sessionMood?: SessionMood | null;
}

export interface SessionMoodTrack {
  id?: string;
  artist?: string;
  album?: string;
  albumId?: string;
  genre?: string;
}

/** Max gap between listens still treated as one listening session. */
const CO_PLAY_WINDOW_MS = 90 * 60 * 1000;

export function decadeFromYear(year: number): number {
  return Math.floor(year / 10) * 10;
}

function normalizeKey(value: string | undefined | null): string {
  return value?.trim().toLowerCase() ?? "";
}

/** Log-scaled listen seconds used for affinity when listenedMs is present. */
export function listenAffinityWeight(entry: ListenEntry): number {
  const listened = Math.max(0, entry.listenedMs ?? 0);
  const plays = Math.max(0, entry.playCount ?? 0);
  // Combine real listen time with a soft play-count prior so tiny
  // listened_ms values never drop below the no-telemetry estimate.
  const listenPart = listened > 0 ? Math.log1p(listened / 1000) : 0;
  const playPart = Math.log1p(plays * 30) * 0.35;
  return Math.max(0.25, listenPart + playPart);
}

export function completionRatio(entry: ListenEntry): number {
  const duration = entry.durationMs ?? 0;
  const listened = entry.listenedMs ?? 0;
  if (duration > 0 && listened > 0) {
    return Math.min(2, listened / duration);
  }
  if (entry.played) return 1;
  if (duration > 0 && entry.positionMs > 0) {
    return Math.min(1, entry.positionMs / duration);
  }
  return 0;
}

/** Half-life for how fast older listens lose affinity weight. */
const AFFINITY_HALF_LIFE_MS = 45 * 86_400_000;

function recencyDecay(playedAtMs: number, nowMs: number): number {
  if (!Number.isFinite(playedAtMs) || playedAtMs <= 0) return 0.55;
  const age = Math.max(0, nowMs - playedAtMs);
  return 0.3 + 0.7 * Math.exp((-Math.LN2 * age) / AFFINITY_HALF_LIFE_MS);
}

function rankedAffinityKeys(
  affinity: ReadonlyMap<string, number>,
  limit: number,
): string[] {
  return [...affinity.entries()]
    .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
    .slice(0, limit)
    .map(([key]) => key);
}

export function rankedDecadeKeys(
  affinity: ReadonlyMap<number, number>,
  limit: number,
): number[] {
  return [...affinity.entries()]
    .sort((a, b) => b[1] - a[1] || a[0] - b[0])
    .slice(0, limit)
    .map(([key]) => key);
}

/** Rank affinity map keys by weight descending. */
export function topAffinityKeys(
  affinity: ReadonlyMap<string, number>,
  limit: number,
): string[] {
  return rankedAffinityKeys(affinity, limit);
}

/**
 * Build a mood vector from newest-first recent completions.
 * Strength scales all weights (use a lower value during cold start).
 */
export function buildSessionMood(
  recentNewestFirst: readonly SessionMoodTrack[],
  strength = 1,
): SessionMood {
  const artistWeights = new Map<string, number>();
  const genreWeights = new Map<string, number>();
  const albumWeights = new Map<string, number>();
  const clamped = Math.max(0, Math.min(1.5, strength));
  if (clamped <= 0 || recentNewestFirst.length === 0) {
    return { artistWeights, genreWeights, albumWeights, strength: 0 };
  }

  recentNewestFirst.slice(0, 8).forEach((track, index) => {
    const decay = Math.pow(0.72, index) * clamped;
    const artist = normalizeKey(track.artist);
    if (artist) {
      artistWeights.set(artist, (artistWeights.get(artist) ?? 0) + decay * 3);
    }
    const genre = normalizeKey(track.genre);
    if (genre) {
      genreWeights.set(genre, (genreWeights.get(genre) ?? 0) + decay * 2);
    }
    const album =
      normalizeKey(track.album) || track.albumId?.trim().toLowerCase() || "";
    if (album) {
      albumWeights.set(album, (albumWeights.get(album) ?? 0) + decay * 1.4);
    }
  });

  return {
    artistWeights,
    genreWeights,
    albumWeights,
    strength: clamped,
  };
}

function buildFollowArtistAffinity(
  history: readonly ListenEntry[],
): Map<string, number> {
  const followArtistAffinity = new Map<string, number>();
  if (history.length < 2) return followArtistAffinity;

  const chronological = [...history].sort((a, b) => {
    const aAt = new Date(a.lastPlayedAt).getTime();
    const bAt = new Date(b.lastPlayedAt).getTime();
    return aAt - bAt || a.trackId.localeCompare(b.trackId);
  });

  for (let i = 1; i < chronological.length; i++) {
    const prev = chronological[i - 1];
    const curr = chronological[i];
    const prevAt = new Date(prev.lastPlayedAt).getTime();
    const currAt = new Date(curr.lastPlayedAt).getTime();
    if (!Number.isFinite(prevAt) || !Number.isFinite(currAt)) continue;
    const gap = currAt - prevAt;
    if (gap < 0 || gap > CO_PLAY_WINDOW_MS) continue;

    const from = normalizeKey(prev.artistName);
    const to = normalizeKey(curr.artistName);
    if (!from || !to || from === to) continue;

    const prevRatio = completionRatio(prev);
    const currRatio = completionRatio(curr);
    if (prevRatio < 0.4 || currRatio < 0.35) continue;

    const key = `${from}|${to}`;
    const weight = Math.min(prevRatio, 1.25) * Math.min(currRatio, 1.25);
    followArtistAffinity.set(
      key,
      (followArtistAffinity.get(key) ?? 0) + weight,
    );
  }

  return followArtistAffinity;
}

/**
 * Prefer tracks the listener actually finished, not raw play-count spam.
 * Optional preferFirst (last completed) is always kept at the front when set.
 */
export function pickQualitySeedIds(
  history: readonly ListenEntry[],
  stats: ListenStats | null,
  limit: number,
  preferFirst: string | null = null,
  nowMs: number = Date.now(),
): string[] {
  if (limit <= 0) return [];

  const scored = new Map<string, number>();
  for (const entry of history) {
    const ratio = completionRatio(entry);
    const listened = entry.listenedMs ?? 0;
    if (ratio < 0.35 && listened < 45_000) continue;
    const playedAt = new Date(entry.lastPlayedAt).getTime();
    const weight =
      listenAffinityWeight(entry) *
      (0.5 + Math.min(1.5, ratio)) *
      recencyDecay(playedAt, nowMs);
    scored.set(entry.trackId, Math.max(scored.get(entry.trackId) ?? 0, weight));
  }

  for (const [index, track] of (stats?.topTracks ?? []).entries()) {
    if (!track.key) continue;
    const prior = Math.max(0.5, 7 - index) * 0.4;
    scored.set(track.key, (scored.get(track.key) ?? 0) + prior);
  }

  const ranked = [...scored.entries()]
    .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
    .map(([id]) => id);

  const out: string[] = [];
  const prefer = preferFirst?.trim() || "";
  if (prefer) out.push(prefer);
  for (const id of ranked) {
    if (out.length >= limit) break;
    if (!out.includes(id)) out.push(id);
  }
  return out.slice(0, limit);
}

export function buildTasteProfile(
  stats: ListenStats | null,
  history: readonly ListenEntry[],
  playCountByTrack: ReadonlyMap<string, number>,
  skippedTrackIds: ReadonlySet<string>,
  genreSources: readonly { genre?: string; year?: number }[] = [],
  listenedMsByTrack: ReadonlyMap<string, number> = new Map(),
  nowMs: number = Date.now(),
): TasteProfile {
  const artistAffinity = new Map<string, number>();
  const albumAffinity = new Map<string, number>();
  const genreAffinity = new Map<string, number>();
  const decadeAffinity = new Map<number, number>();
  const lastPlayedAt = new Map<string, number>();
  const durationMsByTrack = new Map<string, number>();
  const listenedMap = new Map(listenedMsByTrack);

  for (const [index, artist] of (stats?.topArtists ?? []).entries()) {
    const key = normalizeKey(artist.label);
    if (!key) continue;
    artistAffinity.set(key, Math.max(artist.count, 12 - index));
  }

  for (const [index, album] of (stats?.topAlbums ?? []).entries()) {
    const key = normalizeKey(album.label) || album.key;
    if (!key) continue;
    albumAffinity.set(key, Math.max(album.count, 10 - index));
  }

  for (const entry of history) {
    const weight = listenAffinityWeight(entry);
    const ratio = completionRatio(entry);
    const qualityBoost = 0.5 + Math.min(1.5, ratio);
    const playedAt = new Date(entry.lastPlayedAt).getTime();
    const decay = recencyDecay(playedAt, nowMs);
    const contribution = weight * qualityBoost * decay;

    const artistKey = normalizeKey(entry.artistName);
    if (artistKey) {
      artistAffinity.set(
        artistKey,
        (artistAffinity.get(artistKey) ?? 0) + contribution,
      );
    }
    const albumKey = normalizeKey(entry.albumTitle) || entry.albumId || "";
    if (albumKey) {
      albumAffinity.set(
        albumKey,
        (albumAffinity.get(albumKey) ?? 0) + contribution,
      );
    }
    if (Number.isFinite(playedAt)) {
      const existing = lastPlayedAt.get(entry.trackId) ?? 0;
      if (playedAt > existing) lastPlayedAt.set(entry.trackId, playedAt);
    }
    if ((entry.listenedMs ?? 0) > 0) {
      listenedMap.set(entry.trackId, entry.listenedMs);
    }
    if (entry.durationMs > 0) {
      durationMsByTrack.set(entry.trackId, entry.durationMs);
    }
  }

  for (const source of genreSources) {
    const genreKey = normalizeKey(source.genre);
    if (genreKey) {
      genreAffinity.set(genreKey, (genreAffinity.get(genreKey) ?? 0) + 2);
    }
    if (typeof source.year === "number" && source.year >= 1900) {
      const decade = decadeFromYear(source.year);
      decadeAffinity.set(decade, (decadeAffinity.get(decade) ?? 0) + 2);
    }
  }

  return {
    artistAffinity,
    albumAffinity,
    genreAffinity,
    decadeAffinity,
    followArtistAffinity: buildFollowArtistAffinity(history),
    playCountByTrack: new Map(playCountByTrack),
    listenedMsByTrack: listenedMap,
    durationMsByTrack,
    lastPlayedAt,
    skippedTrackIds,
    topArtistKeys: rankedAffinityKeys(artistAffinity, 12),
  };
}

/**
 * Higher scores mean a better fit for For You / personal radio.
 * Real listen time and completion quality outweigh raw play counts.
 * Skips and very recent plays are penalized. Mild exploration favors
 * lightly-heard tracks from favored artists.
 */
export function scoreTrackTaste(
  track: SubsonicSong,
  profile: TasteProfile,
  options: TasteScoreOptions = {},
): number {
  const now = options.nowMs ?? Date.now();
  const recentMs = options.recentMs ?? 21 * 86_400_000;
  const exploreBonus = options.exploreBonus ?? 0.35;

  if (profile.skippedTrackIds.has(track.id)) return -100;

  const artistKey = normalizeKey(track.artist);
  const albumKey =
    normalizeKey(track.album) || track.albumId?.trim().toLowerCase() || "";
  const genreKey = normalizeKey(track.genre);

  const artistScore = artistKey
    ? (profile.artistAffinity.get(artistKey) ?? 0)
    : 0;
  const albumScore = albumKey ? (profile.albumAffinity.get(albumKey) ?? 0) : 0;
  const genreScore = genreKey ? (profile.genreAffinity.get(genreKey) ?? 0) : 0;
  const plays = profile.playCountByTrack.get(track.id) ?? 0;
  const listenedMs = profile.listenedMsByTrack.get(track.id) ?? 0;
  const durationMs =
    profile.durationMsByTrack.get(track.id) ??
    (track.duration ? track.duration * 1000 : 0);

  const listenSeconds = listenedMs / 1000;
  const listenScore =
    listenSeconds > 0 ? Math.min(10, Math.log1p(listenSeconds) * 1.35) : 0;
  const playScore = Math.min(plays, 8) * 0.35;
  // Compress mega-artists so one heavy act cannot drown the mix.
  const artistFit = Math.log1p(artistScore) * 3.1;
  const albumFit = Math.log1p(albumScore) * 1.7;
  const genreFit = Math.log1p(genreScore) * 2.1;

  let score = artistFit + albumFit + genreFit + listenScore + playScore;

  if (durationMs > 0 && listenedMs > 0) {
    const ratio = listenedMs / durationMs;
    if (ratio >= 0.75) {
      score += 2.5;
    } else if (ratio >= 0.4) {
      score += 1;
    } else if (ratio < 0.2 && plays >= 2) {
      score -= 2.5;
    }
  } else if (listenedMs > 45_000 && plays === 0) {
    score += 1.5;
  }

  const lastPlayed = profile.lastPlayedAt.get(track.id);
  if (lastPlayed && now - lastPlayed < recentMs) {
    const freshness = 1 - (now - lastPlayed) / recentMs;
    const listenFactor =
      listenSeconds > 0 ? Math.min(1.4, 0.6 + listenSeconds / 180) : 1;
    score -= freshness * 8 * listenFactor;
  }

  if (plays === 0 && listenedMs === 0 && (artistScore > 0 || genreScore > 0)) {
    score += exploreBonus * 4;
  } else if (plays <= 2 && listenedMs < 60_000 && artistScore > 0) {
    score += exploreBonus * 2;
  } else if (plays >= 3 && plays <= 10 && artistScore > 0) {
    // Mid-tier sweet spot: known but not worn out.
    score += exploreBonus * 1.15;
  }

  if (artistScore > 20) {
    score -= Math.min(5, (artistScore - 20) * 0.1);
  }

  if (options.recentMixTrackIds?.has(track.id)) {
    score -= 3.5;
  }

  if (options.starredTrackIds?.has(track.id)) {
    score += 3.2;
  }

  if (typeof track.year === "number" && track.year >= 1900) {
    const decadeScore =
      profile.decadeAffinity.get(decadeFromYear(track.year)) ?? 0;
    score += Math.min(3, decadeScore * 0.35);
  }

  if (artistScore === 0 && albumScore === 0 && genreScore === 0) {
    score += exploreBonus;
  }

  const mood = options.sessionMood;
  if (mood && mood.strength > 0) {
    const artistMood = artistKey ? (mood.artistWeights.get(artistKey) ?? 0) : 0;
    const genreMood = genreKey ? (mood.genreWeights.get(genreKey) ?? 0) : 0;
    const albumMood = albumKey ? (mood.albumWeights.get(albumKey) ?? 0) : 0;
    score += artistMood * 2.4 + genreMood * 1.6 + albumMood * 1.2;

    if (artistKey) {
      let followBoost = 0;
      for (const [fromArtist, fromWeight] of mood.artistWeights) {
        if (fromArtist === artistKey) continue;
        const follow =
          profile.followArtistAffinity.get(`${fromArtist}|${artistKey}`) ?? 0;
        if (follow > 0) {
          followBoost += Math.min(3.5, follow * fromWeight * 1.6);
        }
      }
      score += Math.min(5, followBoost);
    }
  } else if (artistKey && profile.followArtistAffinity.size > 0) {
    // Without a live session, lightly prefer artists that often follow favorites.
    let follow = 0;
    for (const top of profile.topArtistKeys.slice(0, 4)) {
      follow += profile.followArtistAffinity.get(`${top}|${artistKey}`) ?? 0;
    }
    if (follow > 0) score += Math.min(2.5, follow * 0.85);
  }

  return score;
}

/** Rank albums for home shelves using the same affinity maps as tracks. */
export function scoreAlbumTaste(
  album: {
    id: string;
    name?: string;
    artist?: string;
    year?: number;
    genre?: string;
  },
  profile: TasteProfile,
): number {
  if (profile.skippedTrackIds.has(album.id)) return -50;

  const artistKey = normalizeKey(album.artist);
  const albumKey = normalizeKey(album.name) || album.id.trim().toLowerCase();
  const genreKey = normalizeKey(album.genre);

  const artistScore = artistKey
    ? (profile.artistAffinity.get(artistKey) ?? 0)
    : 0;
  const albumScore = albumKey ? (profile.albumAffinity.get(albumKey) ?? 0) : 0;
  const genreScore = genreKey ? (profile.genreAffinity.get(genreKey) ?? 0) : 0;

  let score =
    Math.log1p(artistScore) * 3.2 +
    Math.log1p(albumScore) * 2.4 +
    Math.log1p(genreScore) * 1.8;

  if (typeof album.year === "number" && album.year >= 1900) {
    const decadeScore =
      profile.decadeAffinity.get(decadeFromYear(album.year)) ?? 0;
    score += Math.min(2.5, decadeScore * 0.3);
  }

  if (artistScore === 0 && albumScore === 0 && genreScore === 0) {
    score += 0.4;
  }

  return score;
}

export function rankAlbumsByTaste<
  T extends {
    id: string;
    name?: string;
    artist?: string;
    year?: number;
    genre?: string;
  },
>(albums: readonly T[], profile: TasteProfile): T[] {
  return [...albums].sort(
    (a, b) =>
      scoreAlbumTaste(b, profile) - scoreAlbumTaste(a, profile) ||
      a.id.localeCompare(b.id),
  );
}

export function rankTracksByTaste(
  tracks: readonly SubsonicSong[],
  profile: TasteProfile,
  options: TasteScoreOptions = {},
): SubsonicSong[] {
  return [...tracks].sort(
    (a, b) =>
      scoreTrackTaste(b, profile, options) -
      scoreTrackTaste(a, profile, options),
  );
}

export function weightedSampleTracks(
  tracks: readonly SubsonicSong[],
  profile: TasteProfile,
  count: number,
  random: () => number,
  options: TasteScoreOptions = {},
  excludeIds: ReadonlySet<string> = new Set(),
): SubsonicSong[] {
  const seen = new Set<string>();
  const pool: { track: SubsonicSong; weight: number }[] = [];
  for (const track of tracks) {
    const id = track.id?.trim();
    if (!id || excludeIds.has(track.id) || excludeIds.has(id) || seen.has(id)) {
      continue;
    }
    seen.add(id);
    pool.push({
      track,
      weight: Math.max(0.05, scoreTrackTaste(track, profile, options) + 2),
    });
  }

  const picks: SubsonicSong[] = [];
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
    picks.push(pool[index].track);
    pool.splice(index, 1);
  }
  return picks;
}

export function isTopArtistTrack(
  track: SubsonicSong,
  profile: TasteProfile,
): boolean {
  const key = normalizeKey(track.artist);
  return key.length > 0 && profile.topArtistKeys.includes(key);
}
