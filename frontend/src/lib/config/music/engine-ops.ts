// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { createPlaybackEngine } from "$lib/music/create-playback-engine";
import type { PlaybackEngine } from "$lib/music/playback-engine";
import type { ImmersiveAudioSettings } from "$lib/music/immersive-audio-settings";
import type { EqSettings } from "$lib/music/eq";
import {
  saveNativePlaybackPref,
  saveNativeBackendPref,
  loadNativeBackendPref,
  type NativeBackendPref,
} from "$lib/music/prefs";
import { nativeDesktopAvailable } from "$lib/config/runtime";
import { updateMediaSessionPosition } from "$lib/music/media-session";
import { isInternetRadioTrack, type QueueTrack } from "$lib/subsonic";
import { logger } from "$lib/core/logger";
import { toast } from "$lib/ui/toast.svelte";

const MOBILE_MEDIA_SYNC_MS = 1000;

export interface MusicEngineContext {
  engine: PlaybackEngine | null;
  engineInitPromise: Promise<void> | null;
  nativePlayback: boolean;
  nativeAvailable: boolean;
  mpvAvailable: boolean;
  vlcAvailable: boolean;
  nativeBackend: string;
  nativeInitError: string;
  mpvLoadError: string;
  volume: number;
  immersiveAudioSettings: ImmersiveAudioSettings;
  eq: EqSettings;
  eqOpen: boolean;
  cleanupListeners: (() => void)[];
  currentTime: number;
  duration: number;
  currentTrack: QueueTrack | null;
  playing: boolean;
  lastSavedPosition: number;
  playbackEpoch: number;
  nextTrackPrepared: boolean;
  nativeFallbackAttempted: boolean;
  transcodedTrackIds: Set<string>;
  onTrackEnded(): void;
  maybePrepareNext(): void;
  maybeStartCrossfade(): void;
  saveProgress(
    positionMs: number,
    options?: { deltaMs?: number },
  ): Promise<void>;
  persistPlaybackState(): void;
  handlePlaybackNetworkFailure(token: number): Promise<void>;
  skipFailedTrack(epoch: number, reason?: string): void;
  markTrackTranscoded(trackId: string): void;
  playCurrent(epoch?: number, resume?: boolean): Promise<void>;
  stopSmoothProgress(): void;
  syncMediaSession(): void;
  trackStreamUrl(track: QueueTrack): string;
  loadTrackSource(token: number, track: QueueTrack, url: string): Promise<void>;
  prefetchAround(): void;
  startProgressTracking(): void;
  startSmoothProgress(): void;
}

export function applyNativeCaps(
  ctx: MusicEngineContext,
  caps: {
    nativeAvailable: boolean;
    backend: string;
    mpvAvailable: boolean;
    vlcAvailable: boolean;
    mpvLoadError?: string;
    initError?: string;
  },
) {
  ctx.nativeAvailable = caps.nativeAvailable;
  ctx.mpvAvailable = caps.mpvAvailable;
  ctx.vlcAvailable = caps.vlcAvailable;
  ctx.nativeBackend = caps.backend;
  ctx.nativeInitError = caps.initError ?? "";
  ctx.mpvLoadError = caps.mpvLoadError ?? "";
}

export function initEngine(ctx: MusicEngineContext): Promise<void> {
  if (typeof window === "undefined") return Promise.resolve();
  if (ctx.engine) return Promise.resolve();
  if (!ctx.engineInitPromise) {
    ctx.engineInitPromise = setupEngine(ctx);
  }
  return ctx.engineInitPromise;
}

export async function refreshNativeCapabilities(
  ctx: MusicEngineContext,
): Promise<void> {
  await syncNativeCaps(ctx);
}

export async function setupEngine(ctx: MusicEngineContext) {
  await buildEngine(ctx);
}

export async function buildEngine(
  ctx: MusicEngineContext,
  force?: "web" | "native",
) {
  const { engine, native } = await createPlaybackEngine(force);
  ctx.engine = engine;
  ctx.nativePlayback = native;
  await syncNativeCaps(ctx);
  if (ctx.nativePlayback) {
    ctx.eqOpen = false;
  }
  ctx.engine.setVolume(ctx.volume);
  ctx.engine.applyImmersiveAudio(ctx.immersiveAudioSettings);
  if (!ctx.nativePlayback && ctx.eq.enabled) {
    ctx.engine.applyEq(ctx.eq);
  }
  attachEngineListeners(ctx);
}

export async function syncNativeCaps(ctx: MusicEngineContext) {
  if (!nativeDesktopAvailable()) {
    ctx.nativeAvailable = false;
    ctx.mpvAvailable = false;
    ctx.vlcAvailable = false;
    ctx.nativeBackend = "";
    ctx.nativeInitError = "";
    ctx.mpvLoadError = "";
    return;
  }
  try {
    const { AudioService } =
      await import("@bindings/melovian/services/index.js");
    const caps = await AudioService.Capabilities();
    applyNativeCaps(ctx, caps);
    if (!caps.nativeAvailable && ctx.nativePlayback) {
      saveNativePlaybackPref(false);
      ctx.nativePlayback = false;
    }
  } catch {
    ctx.nativeAvailable = false;
    ctx.mpvAvailable = false;
    ctx.vlcAvailable = false;
    ctx.nativeBackend = "";
    ctx.nativeInitError = "";
    ctx.mpvLoadError = "";
  }
}

export function attachEngineListeners(ctx: MusicEngineContext) {
  if (!ctx.engine) return;
  let lastMobileMediaSyncAt = 0;
  ctx.cleanupListeners.push(
    ctx.engine.onEnded(() => {
      void ctx.onTrackEnded();
    }),
  );
  ctx.cleanupListeners.push(
    ctx.engine.onTimeUpdate(() => {
      if (!ctx.engine) return;
      ctx.currentTime = ctx.engine.currentTime;
      ctx.duration = ctx.engine.duration || (ctx.currentTrack?.duration ?? 0);
      updateMediaSessionPosition(ctx.duration, ctx.currentTime);
      const now = Date.now();
      if (now - lastMobileMediaSyncAt >= MOBILE_MEDIA_SYNC_MS) {
        lastMobileMediaSyncAt = now;
        ctx.syncMediaSession();
      }
      ctx.maybePrepareNext();
      ctx.maybeStartCrossfade();
      const pos = Math.floor(ctx.engine.currentTime * 1000);
      if (Math.abs(pos - ctx.lastSavedPosition) > 5000) {
        const deltaMs = Math.max(
          0,
          Math.min(15000, pos - ctx.lastSavedPosition),
        );
        ctx.lastSavedPosition = pos;
        void ctx.saveProgress(pos, { deltaMs });
        ctx.persistPlaybackState();
      }
    }),
  );
  ctx.cleanupListeners.push(
    ctx.engine.onLoadedMetadata(() => {
      if (ctx.engine) {
        ctx.duration = ctx.engine.duration || (ctx.currentTrack?.duration ?? 0);
        updateMediaSessionPosition(ctx.duration, ctx.currentTime);
        ctx.syncMediaSession();
      }
    }),
  );
  ctx.cleanupListeners.push(
    ctx.engine.onError(() => {
      if (!ctx.currentTrack) return;
      if (ctx.engine?.isNetworkError()) {
        if (ctx.playing) {
          void ctx.handlePlaybackNetworkFailure(ctx.playbackEpoch);
        }
        return;
      }
      if (ctx.nativePlayback) {
        const reason = ctx.engine?.getLastError() || "native playback failed";
        const epoch = ctx.playbackEpoch;
        void fallbackToWebPlayback(ctx, reason).then((switched) => {
          if (!switched) ctx.skipFailedTrack(epoch, reason);
        });
        return;
      }
      if (
        !isInternetRadioTrack(ctx.currentTrack) &&
        !ctx.nativePlayback &&
        ctx.engine?.isDecodeError() &&
        !ctx.transcodedTrackIds.has(ctx.currentTrack.id)
      ) {
        ctx.markTrackTranscoded(ctx.currentTrack.id);
        const epoch = ctx.playbackEpoch;
        void ctx.playCurrent(epoch);
        return;
      }
      if (!ctx.playing) return;
      const epoch = ctx.playbackEpoch;
      ctx.playing = false;
      ctx.stopSmoothProgress();
      ctx.syncMediaSession();
      ctx.skipFailedTrack(epoch);
    }),
  );
}

export function teardownEngine(ctx: MusicEngineContext) {
  for (const cleanup of ctx.cleanupListeners) cleanup();
  ctx.cleanupListeners = [];
  ctx.engine?.destroy();
  ctx.engine = null;
  ctx.engineInitPromise = null;
}

export function recordPlaybackError(ctx: MusicEngineContext, message: string) {
  if (!message) return;
  ctx.nativeInitError = message;
  logger.error(message, undefined, undefined, "playback");
}

export async function fallbackToWebPlayback(
  ctx: MusicEngineContext,
  reason: string,
): Promise<boolean> {
  if (!ctx.nativePlayback || ctx.nativeFallbackAttempted) return false;
  ctx.nativeFallbackAttempted = true;
  recordPlaybackError(ctx, `Native playback failed: ${reason}`);
  toast.error("Native playback failed, switched to web player");

  const wasPlaying = ctx.playing;
  const positionSec = ctx.engine?.currentTime ?? ctx.currentTime;
  ctx.playing = false;
  ctx.stopSmoothProgress();

  teardownEngine(ctx);
  await buildEngine(ctx, "web");
  if (ctx.currentTrack) {
    await resumeCurrentAt(ctx, positionSec, wasPlaying);
  }
  return true;
}

export async function resumeCurrentAt(
  ctx: MusicEngineContext,
  positionSec: number,
  wasPlaying: boolean,
) {
  const token = ++ctx.playbackEpoch;
  const track = ctx.currentTrack;
  if (!track || !ctx.engine) return;

  const url = ctx.trackStreamUrl(track);
  try {
    await ctx.loadTrackSource(token, track, url);
  } catch (err) {
    if (token !== ctx.playbackEpoch) return;
    ctx.playing = false;
    recordPlaybackError(
      ctx,
      err instanceof Error ? err.message : "Failed to load track",
    );
    return;
  }
  if (token !== ctx.playbackEpoch) return;

  ctx.nextTrackPrepared = false;
  ctx.engine.prepareNext("");
  ctx.prefetchAround();

  if (positionSec > 1 && !isInternetRadioTrack(track)) {
    ctx.engine.currentTime = positionSec;
    ctx.currentTime = positionSec;
  }

  if (!wasPlaying) {
    ctx.playing = false;
    ctx.syncMediaSession();
    return;
  }

  try {
    await ctx.engine.play();
    if (token !== ctx.playbackEpoch) return;
    ctx.playing = true;
    ctx.startProgressTracking();
    ctx.startSmoothProgress();
    ctx.syncMediaSession();
  } catch (err) {
    if (token !== ctx.playbackEpoch) return;
    ctx.playing = false;
    recordPlaybackError(
      ctx,
      err instanceof Error ? err.message : "Playback failed",
    );
  }
}

export async function applyEngineChange(ctx: MusicEngineContext) {
  const wasPlaying = ctx.playing;
  const positionSec = ctx.engine?.currentTime ?? ctx.currentTime;
  teardownEngine(ctx);
  await buildEngine(ctx);
  if (ctx.currentTrack) {
    await resumeCurrentAt(ctx, positionSec, wasPlaying);
  }
}

export async function setNativePlaybackEnabled(
  ctx: MusicEngineContext,
  enabled: boolean,
) {
  saveNativePlaybackPref(enabled);
  ctx.nativeFallbackAttempted = false;
  if (enabled && nativeDesktopAvailable()) {
    try {
      const { AudioService } =
        await import("@bindings/melovian/services/index.js");
      await AudioService.SetPreferredBackend(loadNativeBackendPref());
      const caps = await AudioService.Capabilities();
      applyNativeCaps(ctx, caps);
    } catch {
      ctx.nativeAvailable = false;
      ctx.mpvAvailable = false;
      ctx.vlcAvailable = false;
      ctx.nativeBackend = "";
      ctx.nativeInitError = "";
      ctx.mpvLoadError = "";
    }
  }
  await applyEngineChange(ctx);
}

export async function setNativeBackend(
  ctx: MusicEngineContext,
  backend: NativeBackendPref,
) {
  saveNativeBackendPref(backend);
  ctx.nativeFallbackAttempted = false;
  if (!nativeDesktopAvailable()) {
    ctx.nativeAvailable = false;
    ctx.nativeBackend = "";
    ctx.nativeInitError = "";
    ctx.mpvLoadError = "";
  } else {
    try {
      const { AudioService } =
        await import("@bindings/melovian/services/index.js");
      await AudioService.SetPreferredBackend(backend);
      const caps = await AudioService.Capabilities();
      applyNativeCaps(ctx, caps);
    } catch {
      ctx.nativeAvailable = false;
      ctx.nativeBackend = "";
      ctx.nativeInitError = "";
      ctx.mpvLoadError = "";
    }
  }
  await applyEngineChange(ctx);
}

export function createEngineOps(ctx: MusicEngineContext) {
  return {
    applyNativeCaps: (caps: {
      nativeAvailable: boolean;
      backend: string;
      mpvAvailable: boolean;
      vlcAvailable: boolean;
      mpvLoadError?: string;
      initError?: string;
    }) => applyNativeCaps(ctx, caps),
    initEngine: () => initEngine(ctx),
    refreshNativeCapabilities: () => refreshNativeCapabilities(ctx),
    setupEngine: () => setupEngine(ctx),
    buildEngine: (force?: "web" | "native") => buildEngine(ctx, force),
    syncNativeCaps: () => syncNativeCaps(ctx),
    attachEngineListeners: () => attachEngineListeners(ctx),
    teardownEngine: () => teardownEngine(ctx),
    recordPlaybackError: (message: string) => recordPlaybackError(ctx, message),
    fallbackToWebPlayback: (reason: string) =>
      fallbackToWebPlayback(ctx, reason),
    resumeCurrentAt: (positionSec: number, wasPlaying: boolean) =>
      resumeCurrentAt(ctx, positionSec, wasPlaying),
    applyEngineChange: () => applyEngineChange(ctx),
    setNativePlaybackEnabled: (enabled: boolean) =>
      setNativePlaybackEnabled(ctx, enabled),
    setNativeBackend: (backend: NativeBackendPref) =>
      setNativeBackend(ctx, backend),
  };
}
