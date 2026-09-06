// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { inferLanguageWeights } from "../mix-language";
import type { MixSettings } from "../mix-settings";
import type { MixSelectContext } from "../mix-select";
import { buildTasteProfile, listenAffinityWeight } from "../taste-score";
import type {
  ListenEntry,
  ListenStats,
  SubsonicAlbum,
  SubsonicSong,
} from "$lib/subsonic/types";
import type { MixBuildContext, MixBuildState } from "./types";

export function selectContext(
  ctx: MixBuildContext,
  state: MixBuildState,
): MixSelectContext {
  return {
    playCountByTrack: ctx.playCountByTrack,
    listenedMsByTrack: ctx.listenedMsByTrack,
    skippedTrackIds: ctx.skippedTrackIds,
    recentlyPlayedIds: ctx.recentlyPlayedIds,
    recentMixTrackIds: ctx.recentMixTrackIds,
    settings: ctx.settings,
    tasteProfile: ctx.tasteProfile,
    starredTrackIds: state.starredTrackIds,
  };
}
export function createMixBuildState(ctx: MixBuildContext): MixBuildState {
  const languageWeights = inferLanguageWeights(
    ctx.history.map((entry) => ({
      title: entry.trackTitle,
      artist: entry.artistName,
      album: entry.albumTitle,
      weight: Math.max(1, listenAffinityWeight(entry)),
    })),
    ctx.settings.preferredLanguages,
  );

  return {
    usedTrackIds: new Set<string>(),
    usedCoverArtIds: new Set<string>(),
    languageWeights,
    starredTrackIds: new Set<string>(),
    artistClusters: null,
  };
}
export function buildMixContextInput(
  stats: ListenStats | null,
  history: readonly ListenEntry[],
  frequentAlbums: readonly SubsonicAlbum[],
  daySeed: string,
  settings: MixSettings,
  entryToSong: (entry: ListenEntry) => SubsonicSong,
  recentMixTrackIds: ReadonlySet<string> = new Set(),
): MixBuildContext {
  const recentCutoff = Date.now() - settings.discoverRecentDays * 86_400_000;
  const recentlyPlayedIds = new Set<string>();
  const playCountByTrack = new Map<string, number>();
  const listenedMsByTrack = new Map<string, number>();
  const skippedTrackIds = new Set<string>();

  for (const entry of history) {
    playCountByTrack.set(entry.trackId, entry.playCount);
    if ((entry.listenedMs ?? 0) > 0) {
      listenedMsByTrack.set(entry.trackId, entry.listenedMs);
    }
    if (new Date(entry.lastPlayedAt).getTime() >= recentCutoff) {
      recentlyPlayedIds.add(entry.trackId);
    }
    if (isInferredSkip(entry)) {
      skippedTrackIds.add(entry.trackId);
    }
  }

  return {
    stats,
    history,
    frequentAlbums,
    recentlyPlayedIds,
    playCountByTrack,
    listenedMsByTrack,
    skippedTrackIds,
    daySeed,
    settings,
    entryToSong,
    recentMixTrackIds,
    tasteProfile: buildTasteProfile(
      stats,
      history,
      playCountByTrack,
      skippedTrackIds,
      frequentAlbums,
      listenedMsByTrack,
    ),
  };
}

/** Completed plays are never skips. Incomplete listens under 50% are. */
export function isInferredSkip(entry: ListenEntry): boolean {
  if (entry.played) return false;
  const duration = entry.durationMs ?? 0;
  const listened = entry.listenedMs ?? 0;
  if (duration > 0 && listened > 0) {
    return listened / duration < 0.5;
  }
  if (duration > 0) {
    return entry.positionMs / duration < 0.5;
  }
  return true;
}
