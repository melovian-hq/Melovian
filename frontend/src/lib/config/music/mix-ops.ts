// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import * as musicApi from "$lib/music/api";
import { findMix } from "$lib/music/mixes";
import { slimPersonalMix } from "$lib/music/mix-storage";
import {
  buildPersonalMixes,
  buildMixContextInput,
  buildSingleMix,
  sortMixesByPriority,
  upsertMix,
} from "$lib/music/mix-generator";
import {
  shouldRebuildMixes,
  loadMixCache,
  recentMixTrackIds,
  saveMixCache,
  clearMixCache,
} from "$lib/music/mix-cache";
import {
  mergeMixSettings,
  saveMixSettings,
  type MixSettings,
} from "$lib/music/mix-settings";
import { buildTasteProfile, rankAlbumsByTaste } from "$lib/music/taste-score";
import type { SubsonicAlbum, SubsonicSong } from "$lib/subsonic";
import type {
  ListenEntry,
  ListenStats,
  SubsonicConfig,
} from "$lib/subsonic/types";
import { toast } from "$lib/ui/toast.svelte";
import { tasks } from "$lib/tasks/tasks.svelte";
import { createMixFetchers, shuffleTracks } from "./helpers";
import type { PersonalMix } from "./types";

export interface MusicMixContext {
  config: SubsonicConfig;
  personalMixes: PersonalMix[];
  mixSettings: MixSettings;
  mixesRegenerating: boolean;
  mixesRefreshInFlight: Promise<void> | null;
  personalizationFetchedAt: number;
  connected: boolean;
  stats: ListenStats | null;
  listenHistory: ListenEntry[];
  frequentAlbums: SubsonicAlbum[];
  recommendations: SubsonicAlbum[];
  library: MusicLibraryAdapter;
  shuffle: boolean;
  entryToSong(entry: ListenEntry): SubsonicSong;
  resolveTracksForRestore(ids: string[]): Promise<SubsonicSong[]>;
  playTracks(tracks: SubsonicSong[], startIndex?: number): void;
  refreshMixes(): Promise<void>;
  doRefreshMixes(force?: boolean): Promise<void>;
  refreshRecommendations(): Promise<void>;
  hydrateMixTracks(mix: PersonalMix): Promise<PersonalMix>;
  replaceMixWithHydrated(hydrated: PersonalMix): PersonalMix[];
}

export function restoreCachedMixes(ctx: MusicMixContext) {
  const cached = loadMixCache(ctx.config.serverUrl);
  if (!cached) return;
  ctx.personalMixes = sortMixesByPriority(cached.mixes);
}

export async function refreshPersonalization(ctx: MusicMixContext) {
  await ctx.refreshMixes();
  void ctx.refreshRecommendations().catch(() => {});
  ctx.personalizationFetchedAt = Date.now();
}

export async function refreshMixes(ctx: MusicMixContext) {
  if (ctx.mixesRefreshInFlight) return ctx.mixesRefreshInFlight;

  ctx.mixesRefreshInFlight = ctx.doRefreshMixes().finally(() => {
    ctx.mixesRefreshInFlight = null;
  });
  return ctx.mixesRefreshInFlight;
}

export async function doRefreshMixes(ctx: MusicMixContext, force = false) {
  const date = new Date().toISOString().slice(0, 10);
  const daySeed = `${date}:${ctx.mixSettings.mixSeedSuffix || "0"}`;
  const cached = loadMixCache(ctx.config.serverUrl);
  if (!shouldRebuildMixes(force, cached, daySeed)) {
    if (cached && ctx.personalMixes.length === 0) {
      ctx.personalMixes = sortMixesByPriority(cached.mixes);
    }
    return;
  }

  const stats =
    ctx.stats && ctx.stats.topTracks.length >= 12
      ? ctx.stats
      : ((await musicApi.getListenStats(12).catch(() => null)) ?? ctx.stats);
  if (stats) ctx.stats = stats;

  const history =
    ctx.listenHistory.length >= 100
      ? ctx.listenHistory
      : await musicApi.getListenHistory(100).catch(() => ctx.listenHistory);
  if (history.length > ctx.listenHistory.length) {
    ctx.listenHistory = history;
  }
  const fetchers = createMixFetchers(ctx.library);
  const settings = mergeMixSettings(ctx.mixSettings);
  const recentIds = recentMixTrackIds(ctx.config.serverUrl);
  const mixInput = buildMixContextInput(
    stats,
    history,
    ctx.frequentAlbums,
    daySeed,
    settings,
    (entry) => ctx.entryToSong(entry),
    recentIds,
  );

  const mixes = await buildPersonalMixes(mixInput, fetchers, (mix) => {
    ctx.personalMixes = upsertMix(ctx.personalMixes, slimPersonalMix(mix));
  });

  ctx.personalMixes = mixes.map((mix) => slimPersonalMix(mix));
  saveMixCache(ctx.config.serverUrl, daySeed, mixes);
}

export function updateMixSettings(ctx: MusicMixContext, settings: MixSettings) {
  ctx.mixSettings = mergeMixSettings(settings);
  saveMixSettings(ctx.mixSettings);
}

export async function regenerateMixes(ctx: MusicMixContext) {
  if (ctx.mixesRegenerating || !ctx.connected) return;
  ctx.mixesRegenerating = true;
  tasks.beginMixRegeneration("Refreshing mixes");
  try {
    const nextSettings = mergeMixSettings({
      ...ctx.mixSettings,
      mixSeedSuffix: String(Date.now()),
    });
    ctx.mixSettings = nextSettings;
    saveMixSettings(nextSettings);
    clearMixCache(ctx.config.serverUrl);
    ctx.personalMixes = [];
    ctx.personalizationFetchedAt = 0;
    await ctx.doRefreshMixes(true);
    ctx.personalizationFetchedAt = Date.now();
    tasks.endMixRegeneration(true);
  } catch (err) {
    tasks.endMixRegeneration(
      false,
      err instanceof Error ? err.message : "Mix refresh failed",
    );
    throw err;
  } finally {
    ctx.mixesRegenerating = false;
  }
}

export async function regenerateMix(ctx: MusicMixContext, mixId: string) {
  if (ctx.mixesRegenerating || !ctx.connected || !mixId) return;
  ctx.mixesRegenerating = true;
  tasks.beginMixRegeneration("Refreshing mix");
  toast.info("Refreshing mix…");
  try {
    const date = new Date().toISOString().slice(0, 10);
    const daySeed = `${date}:${ctx.mixSettings.mixSeedSuffix || "0"}:solo:${mixId}:${Date.now()}`;
    const stats =
      ctx.stats && ctx.stats.topTracks.length >= 12
        ? ctx.stats
        : ((await musicApi.getListenStats(12).catch(() => null)) ?? ctx.stats);
    if (stats) ctx.stats = stats;

    const history =
      ctx.listenHistory.length >= 100
        ? ctx.listenHistory
        : await musicApi.getListenHistory(100).catch(() => ctx.listenHistory);
    if (history.length > ctx.listenHistory.length) {
      ctx.listenHistory = history;
    }

    const fetchers = createMixFetchers(ctx.library);
    const settings = mergeMixSettings(ctx.mixSettings);
    const mixInput = buildMixContextInput(
      stats,
      history,
      ctx.frequentAlbums,
      daySeed,
      settings,
      (entry) => ctx.entryToSong(entry),
      recentMixTrackIds(ctx.config.serverUrl),
    );
    const mix = await buildSingleMix(mixId, mixInput, fetchers);
    if (!mix) {
      toast.warning("Could not rebuild that mix");
      return;
    }
    ctx.personalMixes = upsertMix(ctx.personalMixes, slimPersonalMix(mix));
    const cached = loadMixCache(ctx.config.serverUrl);
    const cacheSeed =
      cached?.daySeed ?? `${date}:${ctx.mixSettings.mixSeedSuffix || "0"}`;
    saveMixCache(ctx.config.serverUrl, cacheSeed, ctx.personalMixes);
    toast.success(`Refreshed ${mix.title}`);
    tasks.endMixRegeneration(true, `Refreshed ${mix.title}`);
  } catch (err) {
    const message =
      err instanceof Error ? err.message : "Failed to refresh mix";
    tasks.endMixRegeneration(false, message);
    toast.error(message);
  } finally {
    ctx.mixesRegenerating = false;
  }
}

export async function refreshRecommendations(ctx: MusicMixContext) {
  const stats =
    ctx.stats ?? (await musicApi.getListenStats().catch(() => null));
  const albums: SubsonicAlbum[] = [];
  const seen = new Set<string>();

  const addAlbums = (items: SubsonicAlbum[]) => {
    for (const album of items) {
      if (!seen.has(album.id)) {
        seen.add(album.id);
        albums.push(album);
      }
    }
  };

  if (stats && stats.topArtists.length > 0) {
    const artistAlbums = await Promise.all(
      stats.topArtists
        .slice(0, 3)
        .map((artist) =>
          ctx.library.searchArtistAlbums(artist.label, 4).catch(() => []),
        ),
    );
    for (const found of artistAlbums) {
      addAlbums(found);
    }
  }

  const [randomAlbums, newest] = await Promise.all([
    ctx.library.getAlbumList2("random", 6),
    ctx.library.getAlbumList2("newest", 6),
  ]);
  addAlbums(randomAlbums);
  addAlbums(newest);

  const history = ctx.listenHistory;
  const playCountByTrack = new Map<string, number>();
  const listenedMsByTrack = new Map<string, number>();
  for (const entry of history) {
    playCountByTrack.set(entry.trackId, entry.playCount);
    if ((entry.listenedMs ?? 0) > 0) {
      listenedMsByTrack.set(entry.trackId, entry.listenedMs);
    }
  }
  const profile = buildTasteProfile(
    stats,
    history,
    playCountByTrack,
    new Set(),
    albums,
    listenedMsByTrack,
  );
  ctx.recommendations = rankAlbumsByTaste(albums, profile).slice(0, 12);
}

export function getMix(
  ctx: MusicMixContext,
  id: string,
): PersonalMix | undefined {
  return findMix(ctx.personalMixes, id);
}

export async function ensureMix(
  ctx: MusicMixContext,
  id: string,
): Promise<PersonalMix | undefined> {
  let existing = findMix(ctx.personalMixes, id);
  if (!existing) {
    await ctx.refreshMixes();
    existing = findMix(ctx.personalMixes, id);
  }
  if (!existing) return undefined;
  if (existing.tracks.length > 0) return existing;
  const hydrated = await ctx.hydrateMixTracks(existing);
  ctx.personalMixes = ctx.replaceMixWithHydrated(hydrated);
  return hydrated;
}

export function replaceMixWithHydrated(
  ctx: MusicMixContext,
  hydrated: PersonalMix,
): PersonalMix[] {
  return ctx.personalMixes.map((mix) => {
    if (mix.id === hydrated.id) return hydrated;
    if (mix.tracks.length > 0) return slimPersonalMix(mix);
    return mix;
  });
}

export async function hydrateMixTracks(
  ctx: MusicMixContext,
  mix: PersonalMix,
): Promise<PersonalMix> {
  if (mix.tracks.length > 0) return mix;
  const ids = mix.trackIds ?? [];
  if (ids.length === 0) return mix;
  const tracks = await ctx.resolveTracksForRestore(ids);
  return { ...mix, tracks, trackIds: undefined };
}

export async function playMix(ctx: MusicMixContext, mix: PersonalMix) {
  const hydrated = await ctx.hydrateMixTracks(mix);
  if (hydrated.tracks.length === 0) return;
  ctx.shuffle = true;
  ctx.playTracks(shuffleTracks(hydrated.tracks), 0);
  // Tracks are already in the queue. Keep the store entry slim so we do not
  // hold a second full copy of the mix in memory for the session.
  ctx.personalMixes = ctx.replaceMixWithHydrated(slimPersonalMix(hydrated));
}

export function slimStoredMix(ctx: MusicMixContext, mixId: string) {
  ctx.personalMixes = ctx.personalMixes.map((mix) =>
    mix.id === mixId && mix.tracks.length > 0 ? slimPersonalMix(mix) : mix,
  );
}

export function createMixOps(ctx: MusicMixContext) {
  return {
    restoreCachedMixes: () => restoreCachedMixes(ctx),
    refreshPersonalization: () => refreshPersonalization(ctx),
    refreshMixes: () => refreshMixes(ctx),
    doRefreshMixes: (force?: boolean) => doRefreshMixes(ctx, force),
    updateMixSettings: (settings: MixSettings) =>
      updateMixSettings(ctx, settings),
    regenerateMixes: () => regenerateMixes(ctx),
    regenerateMix: (mixId: string) => regenerateMix(ctx, mixId),
    refreshRecommendations: () => refreshRecommendations(ctx),
    getMix: (id: string) => getMix(ctx, id),
    ensureMix: (id: string) => ensureMix(ctx, id),
    replaceMixWithHydrated: (hydrated: PersonalMix) =>
      replaceMixWithHydrated(ctx, hydrated),
    hydrateMixTracks: (mix: PersonalMix) => hydrateMixTracks(ctx, mix),
    playMix: (mix: PersonalMix) => playMix(ctx, mix),
    slimStoredMix: (mixId: string) => slimStoredMix(ctx, mixId),
  };
}
