// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as musicApi from "$lib/music/api";
import { sources } from "$lib/features/sources/store.svelte";
import { resetSubsonicDetailCaches } from "$lib/subsonic/detail-cache";
import { clearRelatedTracksCache } from "$lib/music/related-tracks-cache";
import { toast } from "$lib/ui/toast.svelte";
import type { LibraryStats } from "$lib/subsonic/types";

const LIBRARY_WATCH_IDLE_MS = 30_000;
const LIBRARY_WATCH_SCAN_MS = 5_000;

export interface MusicLibraryRefreshContext {
  libraryRefreshing: boolean;
  connected: boolean;
  homeCoreFetchedAt: number;
  personalizationFetchedAt: number;
  libraryStats: LibraryStats | null;
  libraryRevision: number;
  lastLibrarySongCount: number;
  lastLibraryScanning: boolean;
  libraryWatchTimer: ReturnType<typeof setTimeout> | undefined;
  invalidateArtistIndex(): void;
  refreshHomeCore(): Promise<void>;
  refreshLibraryStats(options?: { bypassCache?: boolean }): Promise<void>;
  refreshFavorites(): Promise<void>;
  refreshServerPlaylists(): Promise<void>;
  loadArtists(options?: { force?: boolean }): Promise<unknown>;
  schedulePersonalizationRefresh(force: boolean): void;
  refreshLibrary(options?: { quiet?: boolean }): Promise<void>;
  stopLibraryWatch(): void;
  scheduleLibraryWatch(delayMs: number): void;
  pollLibraryChanges(): Promise<void>;
}

export async function refreshLibrary(
  ctx: MusicLibraryRefreshContext,
  options: { quiet?: boolean } = {},
) {
  if (ctx.libraryRefreshing) return;
  ctx.libraryRefreshing = true;
  try {
    if (sources.hasSubsonicActive) {
      await musicApi.bustLibraryCache();
    }
    resetSubsonicDetailCaches();
    clearRelatedTracksCache();
    ctx.invalidateArtistIndex();
    ctx.homeCoreFetchedAt = 0;
    ctx.personalizationFetchedAt = 0;

    if (ctx.connected) {
      await Promise.all([
        ctx.refreshHomeCore(),
        ctx.refreshLibraryStats({ bypassCache: true }),
        ctx.refreshFavorites(),
        ctx.refreshServerPlaylists(),
        sources.hasSubsonicActive
          ? ctx.loadArtists({ force: true })
          : Promise.resolve(),
      ]);
      ctx.schedulePersonalizationRefresh(true);
      const stats = ctx.libraryStats;
      if (stats) {
        ctx.lastLibrarySongCount = stats.songCount ?? 0;
        ctx.lastLibraryScanning = stats.scanning ?? false;
      }
    }

    ctx.libraryRevision += 1;
    if (!options.quiet) {
      toast.success("Library refreshed");
    }
  } catch (err) {
    if (!options.quiet) {
      toast.error(
        err instanceof Error ? err.message : "Failed to refresh library",
      );
    }
  } finally {
    ctx.libraryRefreshing = false;
  }
}

export function startLibraryWatch(ctx: MusicLibraryRefreshContext) {
  ctx.stopLibraryWatch();
  if (typeof window === "undefined") return;

  ctx.lastLibrarySongCount = ctx.libraryStats?.songCount ?? 0;
  ctx.lastLibraryScanning = ctx.libraryStats?.scanning ?? false;
  ctx.scheduleLibraryWatch(LIBRARY_WATCH_IDLE_MS);
}

export function stopLibraryWatch(ctx: MusicLibraryRefreshContext) {
  if (ctx.libraryWatchTimer !== undefined) {
    clearTimeout(ctx.libraryWatchTimer);
    ctx.libraryWatchTimer = undefined;
  }
}

export function scheduleLibraryWatch(
  ctx: MusicLibraryRefreshContext,
  delayMs: number,
) {
  ctx.libraryWatchTimer = setTimeout(() => {
    void ctx.pollLibraryChanges();
  }, delayMs);
}

export async function pollLibraryChanges(ctx: MusicLibraryRefreshContext) {
  if (!ctx.connected) return;

  let nextDelay = LIBRARY_WATCH_IDLE_MS;
  try {
    const stats = await musicApi.getLibraryStats({ bypassCache: true });
    if (stats) {
      const songCount = stats.songCount ?? 0;
      const scanning = stats.scanning ?? false;
      const changed =
        (ctx.lastLibrarySongCount > 0 &&
          songCount !== ctx.lastLibrarySongCount) ||
        (ctx.lastLibraryScanning && !scanning);

      if (changed) {
        await ctx.refreshLibrary({ quiet: true });
      } else {
        ctx.libraryStats = stats;
        ctx.lastLibrarySongCount = songCount;
        ctx.lastLibraryScanning = scanning;
      }

      nextDelay = scanning ? LIBRARY_WATCH_SCAN_MS : LIBRARY_WATCH_IDLE_MS;
    }
  } catch {
    /* polling is best-effort */
  }

  if (ctx.connected) {
    ctx.scheduleLibraryWatch(nextDelay);
  }
}

export function createLibraryRefreshOps(ctx: MusicLibraryRefreshContext) {
  return {
    refreshLibrary: (options?: { quiet?: boolean }) =>
      refreshLibrary(ctx, options),
    startLibraryWatch: () => startLibraryWatch(ctx),
    stopLibraryWatch: () => stopLibraryWatch(ctx),
    scheduleLibraryWatch: (delayMs: number) =>
      scheduleLibraryWatch(ctx, delayMs),
    pollLibraryChanges: () => pollLibraryChanges(ctx),
  };
}
