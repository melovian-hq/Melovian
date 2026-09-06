// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import * as musicApi from "$lib/music/api";
import {
  mergeCacheSettings,
  type CacheSettings,
} from "$lib/music/cache-settings";
import {
  isRemoteCacheableTrack,
  selectTracksForOfflineDownload,
} from "$lib/music/track-cache";
import type { QueueTrack, SubsonicAlbum, SubsonicSong } from "$lib/subsonic";
import type { ListenEntry, MusicPlaylist } from "$lib/subsonic/types";

export interface MusicCacheContext {
  downloadedIds: Set<string>;
  cacheSettings: CacheSettings;
  cacheUsedBytes: number;
  cacheTrackCount: number;
  cacheInFlight: Set<string>;
  cacheStrategyRunning: boolean;
  connected: boolean;
  shuffle: boolean;
  repeat: "off" | "all" | "one";
  queue: QueueTrack[];
  listenHistory: ListenEntry[];
  frequentAlbums: SubsonicAlbum[];
  playlists: MusicPlaylist[];
  library: MusicLibraryAdapter;
  offlineDownloadProgress: {
    total: number;
    completed: number;
    failed: number;
    active: boolean;
  } | null;
  offlineDownloadAbort: AbortController | null;
  sequentialNextIndex(): number;
  runCacheStrategy(): Promise<void>;
  downloadCurrentOr(track: SubsonicSong): Promise<void>;
  loadCacheSettings(): Promise<void>;
  collectStrategyTracks(): Promise<SubsonicSong[]>;
  prefetchCache(track: SubsonicSong): void;
  removeDownload(trackId: string): Promise<void>;
  cancelOfflineDownload(): void;
}

export function isDownloaded(ctx: MusicCacheContext, trackId: string): boolean {
  return ctx.downloadedIds.has(trackId);
}

export async function loadDownloads(ctx: MusicCacheContext) {
  try {
    const items = await musicApi.listDownloads();
    ctx.downloadedIds = new Set(items.map((d) => d.trackId));
  } catch {
    /* offline cache listing is best-effort */
  }
}

export async function loadCacheSettings(ctx: MusicCacheContext) {
  try {
    const remote = await musicApi.getCacheSettings();
    if (remote) {
      ctx.cacheSettings = mergeCacheSettings(remote);
      ctx.cacheUsedBytes = remote.usedBytes ?? 0;
      ctx.cacheTrackCount = remote.trackCount ?? 0;
    }
  } catch {
    /* cache settings are best-effort */
  }
}

export async function updateCacheSettings(
  ctx: MusicCacheContext,
  settings: CacheSettings,
) {
  const remote = await musicApi.saveCacheSettingsRemote(settings);
  ctx.cacheSettings = mergeCacheSettings(remote);
  ctx.cacheUsedBytes = remote.usedBytes ?? 0;
  ctx.cacheTrackCount = remote.trackCount ?? 0;
  void ctx.runCacheStrategy();
}

export async function clearDownloadCache(ctx: MusicCacheContext) {
  await musicApi.clearDownloadCache();
  ctx.downloadedIds = new Set();
  ctx.cacheUsedBytes = 0;
  ctx.cacheTrackCount = 0;
}

export function maybeCacheTrack(ctx: MusicCacheContext, track: SubsonicSong) {
  if (!ctx.cacheSettings.enabled) return;
  if (!isRemoteCacheableTrack(track)) return;
  if (ctx.downloadedIds.has(track.id) || ctx.cacheInFlight.has(track.id)) {
    return;
  }
  ctx.cacheInFlight.add(track.id);
  void ctx
    .downloadCurrentOr(track)
    .then(() => ctx.loadCacheSettings())
    .catch(() => {})
    .finally(() => {
      ctx.cacheInFlight.delete(track.id);
    });

  if (
    ctx.cacheSettings.strategy === "playback" &&
    !ctx.shuffle &&
    ctx.repeat !== "one"
  ) {
    const nextIdx = ctx.sequentialNextIndex();
    const nextTrack = nextIdx >= 0 ? ctx.queue[nextIdx] : undefined;
    if (nextTrack) void ctx.prefetchCache(nextTrack);
  }
}

export async function runCacheStrategy(ctx: MusicCacheContext) {
  if (!ctx.connected || !ctx.cacheSettings.enabled) return;
  if (ctx.cacheSettings.strategy === "playback") return;
  if (ctx.cacheStrategyRunning) return;
  if (ctx.cacheUsedBytes >= ctx.cacheSettings.limitBytes) return;

  ctx.cacheStrategyRunning = true;
  try {
    const candidates = await ctx.collectStrategyTracks();
    const pending = candidates.filter(
      (track) =>
        isRemoteCacheableTrack(track) &&
        !ctx.downloadedIds.has(track.id) &&
        !ctx.cacheInFlight.has(track.id),
    );
    const batchSize = 2;
    for (let i = 0; i < pending.length; i += batchSize) {
      if (ctx.cacheUsedBytes >= ctx.cacheSettings.limitBytes) break;
      const batch = pending.slice(i, i + batchSize);
      await Promise.all(
        batch.map(async (track) => {
          if (ctx.cacheInFlight.size >= batchSize) return;
          ctx.cacheInFlight.add(track.id);
          try {
            await ctx.downloadCurrentOr(track);
          } finally {
            ctx.cacheInFlight.delete(track.id);
          }
        }),
      );
      await ctx.loadCacheSettings();
    }
  } catch {
    /* background caching is best-effort */
  } finally {
    ctx.cacheStrategyRunning = false;
  }
}

export async function collectStrategyTracks(
  ctx: MusicCacheContext,
): Promise<SubsonicSong[]> {
  const strategy = ctx.cacheSettings.strategy;
  const tracks: SubsonicSong[] = [];

  if (strategy === "most_listened") {
    for (const entry of ctx.listenHistory.slice(0, 100)) {
      tracks.push({
        id: entry.trackId,
        title: entry.trackTitle,
        artist: entry.artistName,
      });
    }
    for (const album of ctx.frequentAlbums.slice(0, 5)) {
      const detail = await ctx.library.getAlbum(album.id).catch(() => null);
      if (detail) tracks.push(...detail.songs);
    }
    return tracks;
  }

  if (strategy === "playlists") {
    for (const playlist of ctx.playlists) {
      const detail = await musicApi.getPlaylist(playlist.id).catch(() => null);
      if (!detail?.tracks) continue;
      for (const item of detail.tracks) {
        tracks.push({
          id: item.trackId,
          title: item.trackTitle,
          artist: item.artistName,
        });
      }
    }
    return tracks;
  }

  if (strategy === "all") {
    const random = await ctx.library.getRandomSongs(50).catch(() => []);
    tracks.push(...random);
    const albums = await ctx.library
      .getAlbumList2("newest", 20)
      .catch(() => []);
    for (const album of albums.slice(0, 10)) {
      const detail = await ctx.library.getAlbum(album.id).catch(() => null);
      if (detail) tracks.push(...detail.songs);
    }
  }

  return tracks;
}

export function prefetchCache(ctx: MusicCacheContext, track: SubsonicSong) {
  if (!ctx.cacheSettings.enabled) return;
  if (!isRemoteCacheableTrack(track)) return;
  if (ctx.downloadedIds.has(track.id) || ctx.cacheInFlight.has(track.id)) {
    return;
  }
  ctx.cacheInFlight.add(track.id);
  void ctx
    .downloadCurrentOr(track)
    .then(() => ctx.loadCacheSettings())
    .catch(() => {})
    .finally(() => {
      ctx.cacheInFlight.delete(track.id);
    });
}

export async function downloadCurrentOr(
  ctx: MusicCacheContext,
  track: SubsonicSong,
) {
  if (!isRemoteCacheableTrack(track)) return;
  if (ctx.downloadedIds.has(track.id)) return;
  await musicApi.downloadTrack(track.id, {
    title: track.title,
    artist: track.artist ?? "",
  });
  const next = new Set(ctx.downloadedIds);
  next.add(track.id);
  ctx.downloadedIds = next;
}

export async function removeDownload(ctx: MusicCacheContext, trackId: string) {
  await musicApi.deleteDownload(trackId).catch(() => {});
  const next = new Set(ctx.downloadedIds);
  next.delete(trackId);
  ctx.downloadedIds = next;
}

export async function toggleDownload(
  ctx: MusicCacheContext,
  track: SubsonicSong,
) {
  if (!isRemoteCacheableTrack(track)) return;
  if (ctx.downloadedIds.has(track.id)) {
    await ctx.removeDownload(track.id);
  } else {
    await ctx.downloadCurrentOr(track);
  }
}

export function cancelOfflineDownload(ctx: MusicCacheContext) {
  ctx.offlineDownloadAbort?.abort();
  ctx.offlineDownloadAbort = null;
  if (ctx.offlineDownloadProgress) {
    ctx.offlineDownloadProgress = {
      ...ctx.offlineDownloadProgress,
      active: false,
    };
  }
}

export async function downloadTracks(
  ctx: MusicCacheContext,
  tracks: SubsonicSong[],
  options: { concurrency?: number } = {},
): Promise<{ downloaded: number; skipped: number; failed: number }> {
  if (!ctx.cacheSettings.enabled) {
    throw new Error("Track caching is disabled");
  }
  const pending = selectTracksForOfflineDownload(tracks, ctx.downloadedIds);
  const skipped = tracks.length - pending.length;
  if (pending.length === 0) {
    return { downloaded: 0, skipped, failed: 0 };
  }

  ctx.cancelOfflineDownload();
  const abort = new AbortController();
  ctx.offlineDownloadAbort = abort;
  const concurrency = Math.max(1, Math.min(options.concurrency ?? 3, 6));
  let completed = 0;
  let failed = 0;
  let downloaded = 0;
  ctx.offlineDownloadProgress = {
    total: pending.length,
    completed: 0,
    failed: 0,
    active: true,
  };

  let index = 0;
  const workers = Array.from({ length: concurrency }, async () => {
    while (index < pending.length) {
      if (abort.signal.aborted) return;
      const current = index++;
      const track = pending[current];
      if (!track) return;
      try {
        await musicApi.downloadTrack(
          track.id,
          { title: track.title, artist: track.artist ?? "" },
          abort.signal,
        );
        if (abort.signal.aborted) return;
        const next = new Set(ctx.downloadedIds);
        next.add(track.id);
        ctx.downloadedIds = next;
        downloaded += 1;
      } catch {
        if (abort.signal.aborted) return;
        failed += 1;
      } finally {
        completed += 1;
        ctx.offlineDownloadProgress = {
          total: pending.length,
          completed,
          failed,
          active: !abort.signal.aborted && completed < pending.length,
        };
      }
    }
  });

  await Promise.all(workers);
  const active = !abort.signal.aborted;
  ctx.offlineDownloadProgress = {
    total: pending.length,
    completed,
    failed,
    active: false,
  };
  if (ctx.offlineDownloadAbort === abort) {
    ctx.offlineDownloadAbort = null;
  }
  if (active) {
    await ctx.loadCacheSettings().catch(() => {});
  }
  return { downloaded, skipped, failed };
}

export function createCacheOps(ctx: MusicCacheContext) {
  return {
    isDownloaded: (trackId: string) => isDownloaded(ctx, trackId),
    loadDownloads: () => loadDownloads(ctx),
    loadCacheSettings: () => loadCacheSettings(ctx),
    updateCacheSettings: (settings: CacheSettings) =>
      updateCacheSettings(ctx, settings),
    clearDownloadCache: () => clearDownloadCache(ctx),
    maybeCacheTrack: (track: SubsonicSong) => maybeCacheTrack(ctx, track),
    runCacheStrategy: () => runCacheStrategy(ctx),
    collectStrategyTracks: () => collectStrategyTracks(ctx),
    prefetchCache: (track: SubsonicSong) => prefetchCache(ctx, track),
    downloadCurrentOr: (track: SubsonicSong) => downloadCurrentOr(ctx, track),
    removeDownload: (trackId: string) => removeDownload(ctx, trackId),
    toggleDownload: (track: SubsonicSong) => toggleDownload(ctx, track),
    cancelOfflineDownload: () => cancelOfflineDownload(ctx),
    downloadTracks: (
      tracks: SubsonicSong[],
      options?: { concurrency?: number },
    ) => downloadTracks(ctx, tracks, options),
  };
}
