// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as musicApi from "$lib/music/api";
import {
  shouldAwaitConnectInFlight,
  shouldSkipMusicConnect,
} from "$lib/music/connect-session";
import { connection } from "$lib/music/connection.svelte";
import { shouldRestorePlayback } from "$lib/music/playback-session";
import { setMusicSourceKind } from "$lib/music/source.svelte";
import { clearMetadataEnhancementMemoryCache } from "$lib/music/metadata-enhancement";
import { clearRelatedTracksCache } from "$lib/music/related-tracks-cache";
import { sources } from "$lib/features/sources/store.svelte";
import { isInternetRadioTrack, type QueueTrack } from "$lib/subsonic";
import { resetSubsonicDetailCaches } from "$lib/subsonic/detail-cache";
import type { MusicStatus } from "$lib/subsonic/types";
import type { PlaybackEngine } from "$lib/music/playback-engine";

export interface MusicConnectContext {
  connectInFlight: Promise<boolean> | null;
  connected: boolean;
  loading: boolean;
  error: string | null;
  status: MusicStatus;
  libraryWarmup: boolean;
  playbackRestored: boolean;
  playing: boolean;
  queue: QueueTrack[];
  queueIndex: number;
  reconnectResumePending: boolean;
  reconnectPositionMs: number;
  currentTime: number;
  currentTrack: QueueTrack | null;
  engine: PlaybackEngine | null;
  playbackEpoch: number;
  homeCoreFetchedAt: number;
  homeStaleMs: number;
  restoreCachedMixes(): void;
  initEngine(): Promise<void>;
  loadEqSettings(authEnabled: boolean): Promise<void>;
  refreshHomeCore(): Promise<void>;
  loadCacheSettings(): Promise<void>;
  loadLyricsSettings(): Promise<void>;
  restorePlayback(): Promise<void>;
  startLibraryWatch(): void;
  stopLibraryWatch(): void;
  refreshHistory(limit: number): Promise<void>;
  refreshStats(limit: number): Promise<void>;
  refreshPlaylists(): Promise<void>;
  refreshServerPlaylists(): Promise<void>;
  refreshInternetRadios(): Promise<void>;
  refreshFavorites(): Promise<void>;
  refreshLibraryStats(): Promise<void>;
  loadDownloads(): Promise<void>;
  schedulePersonalizationRefresh(force: boolean): void;
  runCacheStrategy(): Promise<void>;
  isDownloaded(trackId: string): boolean;
  loadTrackSource(token: number, track: QueueTrack, url: string): Promise<void>;
  trackStreamUrl(track: QueueTrack): string;
  startProgressTracking(): void;
  startSmoothProgress(): void;
  stopSmoothProgress(): void;
  syncMediaSession(): void;
  handlePlaybackNetworkFailure(
    token: number,
    options?: { resume?: boolean },
  ): Promise<void>;
  playCurrent(): Promise<void>;
  seek(seconds: number): void;
}

export async function connect(
  ctx: MusicConnectContext,
  options: { quiet?: boolean; authEnabled?: boolean; force?: boolean } = {},
): Promise<boolean> {
  if (shouldAwaitConnectInFlight(ctx.connectInFlight, options.force)) {
    return ctx.connectInFlight;
  }

  if (shouldSkipMusicConnect(ctx.connected, options.force)) {
    if (!options.quiet) ctx.loading = false;
    return true;
  }

  ctx.connectInFlight = doConnect(ctx, options).finally(() => {
    ctx.connectInFlight = null;
  });
  return ctx.connectInFlight;
}

export async function doConnect(
  ctx: MusicConnectContext,
  options: { quiet?: boolean; authEnabled?: boolean } = {},
): Promise<boolean> {
  if (!options.quiet) {
    connection.cancelScheduledReconnect();
    ctx.loading = true;
  }
  ctx.error = null;
  void bootstrapOfflinePlayback(ctx);

  try {
    setMusicSourceKind(
      sources.hasUnifiedMode
        ? "unified"
        : sources.hasLocalActive
          ? "local"
          : "subsonic",
    );
    const status = await musicApi.getMusicStatus();
    ctx.status = status;
    if (!status.enabled) {
      ctx.connected = false;
      await bootstrapOfflinePlayback(ctx);
      return false;
    }
    if (!status.connected) {
      throw new Error(status.error ?? "Server is unreachable");
    }

    ctx.connected = true;
    ctx.libraryWarmup = true;
    ctx.restoreCachedMixes();
    await ctx.initEngine();
    await ctx.loadEqSettings(options.authEnabled ?? false).catch(() => {});

    void finishConnectSetup(ctx, options).finally(() => {
      ctx.libraryWarmup = false;
      connection.onServerConnected();
    });

    return true;
  } catch (err) {
    ctx.libraryWarmup = false;
    ctx.connected = false;
    ctx.error =
      err instanceof Error ? err.message : "Failed to connect to music";
    if (ctx.status.enabled) {
      connection.onServerDisconnected(ctx.error);
    } else {
      connection.onServerDisconnected();
    }
    await bootstrapOfflinePlayback(ctx);
    return false;
  } finally {
    if (!options.quiet) ctx.loading = false;
  }
}

export async function finishConnectSetup(
  ctx: MusicConnectContext,
  options: {
    quiet?: boolean;
    authEnabled?: boolean;
  },
) {
  await Promise.all([
    ctx.refreshHomeCore().catch(() => {}),
    ctx.loadCacheSettings(),
    ctx.loadLyricsSettings(),
  ]);
  if (
    shouldRestorePlayback({
      playbackRestored: ctx.playbackRestored,
      playing: ctx.playing,
      queueLength: ctx.queue.length,
      queueIndex: ctx.queueIndex,
    })
  ) {
    await ctx.restorePlayback().catch(() => {});
  }
  await resumeAfterReconnect(ctx);
  ctx.startLibraryWatch();
  scheduleBackgroundHydration(ctx);
  void options;
}

export function scheduleBackgroundHydration(ctx: MusicConnectContext) {
  const run = () => {
    void Promise.all([
      ctx.refreshHistory(100).catch(() => {}),
      ctx.refreshStats(12).catch(() => {}),
      ctx.refreshPlaylists().catch(() => {}),
      ctx.refreshServerPlaylists(),
      ctx.refreshInternetRadios(),
      ctx.refreshFavorites(),
      ctx.refreshLibraryStats().catch(() => {}),
      ctx.loadDownloads(),
    ])
      .catch(() => {})
      .finally(() => {
        ctx.schedulePersonalizationRefresh(false);
      });
    void ctx.runCacheStrategy();
  };
  if (typeof requestIdleCallback !== "undefined") {
    requestIdleCallback(run, { timeout: 8000 });
  } else {
    setTimeout(run, 3000);
  }
}

export async function pingServer(ctx: MusicConnectContext): Promise<boolean> {
  try {
    setMusicSourceKind(
      sources.hasUnifiedMode
        ? "unified"
        : sources.hasLocalActive
          ? "local"
          : "subsonic",
    );
    const status = await musicApi.getMusicStatus();
    ctx.status = status;
    const ok = status.enabled && status.connected;
    if (ok) {
      ctx.connected = true;
      if (!connection.serverOnline) {
        connection.onServerConnected();
      }
      return true;
    }
    ctx.connected = false;
    if (!ctx.connectInFlight && !ctx.libraryWarmup) {
      handleServerReachabilityLoss(ctx);
    }
    return false;
  } catch {
    ctx.connected = false;
    if (!ctx.connectInFlight && !ctx.libraryWarmup) {
      handleServerReachabilityLoss(ctx);
    }
    return false;
  }
}

export function handleServerReachabilityLoss(ctx: MusicConnectContext) {
  const playingOffline = ctx.playing && canPlayCurrentOffline(ctx);
  if (playingOffline) {
    void ensureOfflinePlaybackSource(ctx, ctx.playbackEpoch);
    connection.onServerDisconnected(undefined, { silent: true });
    return;
  }
  if (ctx.playing) {
    suspendForReconnect(ctx);
  }
  connection.onServerDisconnected();
}

export async function ensureOfflinePlaybackSource(
  ctx: MusicConnectContext,
  token: number,
) {
  const track = ctx.currentTrack;
  if (!track || !ctx.isDownloaded(track.id) || !ctx.engine) return;

  const positionSec = ctx.engine.currentTime;
  const wasPlaying = ctx.playing;
  try {
    await ctx.loadTrackSource(token, track, ctx.trackStreamUrl(track));
    if (token !== ctx.playbackEpoch) return;
    if (positionSec > 1) {
      ctx.engine.currentTime = positionSec;
      ctx.currentTime = positionSec;
    }
    if (wasPlaying && !ctx.engine.canPlay()) {
      await ctx.engine.play();
      if (token !== ctx.playbackEpoch) return;
      ctx.playing = true;
      ctx.startProgressTracking();
      ctx.startSmoothProgress();
      ctx.syncMediaSession();
    }
  } catch {
    void ctx.handlePlaybackNetworkFailure(token);
  }
}

export function canPlayCurrentOffline(ctx: MusicConnectContext): boolean {
  const track = ctx.currentTrack;
  if (!track || isInternetRadioTrack(track)) return false;
  return ctx.isDownloaded(track.id);
}

export async function bootstrapOfflinePlayback(ctx: MusicConnectContext) {
  await Promise.all([ctx.loadDownloads(), ctx.initEngine()]).catch(() => {});
}

export function suspendForReconnect(ctx: MusicConnectContext) {
  if (!ctx.currentTrack || !ctx.playing) return;
  markPendingReconnectResume(ctx);
  ctx.engine?.pause();
  ctx.playing = false;
  ctx.stopSmoothProgress();
  ctx.syncMediaSession();
}

export function markPendingReconnectResume(ctx: MusicConnectContext) {
  if (!ctx.currentTrack) return;
  ctx.reconnectResumePending = true;
  ctx.reconnectPositionMs = Math.max(
    ctx.reconnectPositionMs,
    Math.floor(ctx.currentTime * 1000),
  );
}

export async function resumeAfterReconnect(ctx: MusicConnectContext) {
  if (!ctx.reconnectResumePending || !ctx.currentTrack) return;
  ctx.reconnectResumePending = false;
  const positionMs = ctx.reconnectPositionMs;
  ctx.reconnectPositionMs = 0;
  try {
    await ctx.playCurrent();
    if (positionMs > 1000) {
      ctx.seek(positionMs / 1000);
    }
  } catch {
    ctx.error = "Failed to resume playback";
  }
}

export async function selfHeal(ctx: MusicConnectContext) {
  if (!ctx.connected) return;
  await Promise.all([
    ctx.refreshFavorites(),
    ctx.refreshPlaylists(),
    ctx.refreshServerPlaylists(),
    ctx.refreshInternetRadios(),
    ctx.loadDownloads(),
  ]).catch(() => {});
  if (Date.now() - ctx.homeCoreFetchedAt > ctx.homeStaleMs) {
    void ctx.refreshHomeCore().catch(() => {});
  }
  void ctx.runCacheStrategy();
}

export function disconnect(ctx: MusicConnectContext) {
  connection.cancelScheduledReconnect();
  ctx.connected = false;
  clearRelatedTracksCache();
  resetSubsonicDetailCaches();
  clearMetadataEnhancementMemoryCache();
  ctx.stopLibraryWatch();
  connection.onServerDisconnected();
}

export function createConnectOps(ctx: MusicConnectContext) {
  return {
    connect: (options?: {
      quiet?: boolean;
      authEnabled?: boolean;
      force?: boolean;
    }) => connect(ctx, options),
    doConnect: (options?: { quiet?: boolean; authEnabled?: boolean }) =>
      doConnect(ctx, options),
    finishConnectSetup: (options: { quiet?: boolean; authEnabled?: boolean }) =>
      finishConnectSetup(ctx, options),
    scheduleBackgroundHydration: () => scheduleBackgroundHydration(ctx),
    pingServer: () => pingServer(ctx),
    handleServerReachabilityLoss: () => handleServerReachabilityLoss(ctx),
    ensureOfflinePlaybackSource: (token: number) =>
      ensureOfflinePlaybackSource(ctx, token),
    canPlayCurrentOffline: () => canPlayCurrentOffline(ctx),
    bootstrapOfflinePlayback: () => bootstrapOfflinePlayback(ctx),
    suspendForReconnect: () => suspendForReconnect(ctx),
    markPendingReconnectResume: () => markPendingReconnectResume(ctx),
    resumeAfterReconnect: () => resumeAfterReconnect(ctx),
    selfHeal: () => selfHeal(ctx),
    disconnect: () => disconnect(ctx),
  };
}
