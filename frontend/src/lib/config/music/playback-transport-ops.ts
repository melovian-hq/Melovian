// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { ContinuousMode } from "$lib/music/continuous-pool";
import type { PlaybackEngine } from "$lib/music/playback-engine";
import { updateMediaSessionPosition } from "$lib/music/media-session";
import {
  canAdvanceQueue,
  nextSequentialIndex,
  popShuffleHistory,
} from "$lib/music/playback-queue";
import {
  notePersonalSkip,
  wasTrackSkipped,
  type PersonalRadioState,
} from "$lib/music/personal-radio";
import { saveVolume } from "$lib/music/prefs";
import { isInternetRadioTrack, type QueueTrack } from "$lib/subsonic";
import { pauseVideoForMusic } from "$lib/video/playback-gate.svelte";

export interface MusicPlaybackTransportContext {
  engine: PlaybackEngine | null;
  playing: boolean;
  volume: number;
  repeat: "off" | "all" | "one";
  shuffle: boolean;
  queue: QueueTrack[];
  queueIndex: number;
  currentTrack: QueueTrack | null;
  currentTime: number;
  duration: number;
  smoothProgress: number;
  continuousMode: ContinuousMode;
  shuffleUpcoming: number[];
  shuffleHistory: number[];
  personalRadio: PersonalRadioState;
  lastSavedPosition: number;
  stopSmoothProgress(): void;
  syncMediaSession(): void;
  initEngine(): Promise<void>;
  playCurrent(): Promise<void>;
  requestPlayCurrent(): void;
  advanceShuffleIndex(): boolean;
  stopAtQueueEnd(): void;
  maybeRefillContinuousQueue(): Promise<boolean>;
  persistPlaybackState(): void;
  saveProgress(
    positionMs: number,
    options?: { deltaMs?: number },
  ): Promise<void>;
  prefetchAround(): void;
  startSmoothProgress(): void;
}

export function pause(ctx: MusicPlaybackTransportContext) {
  ctx.playing = false;
  ctx.stopSmoothProgress();
  if (ctx.engine) ctx.engine.pause();
  ctx.syncMediaSession();
}

export async function togglePlay(ctx: MusicPlaybackTransportContext) {
  if (ctx.engine && ctx.currentTrack && ctx.playing) {
    pause(ctx);
    return;
  }

  await ctx.initEngine();
  if (!ctx.engine) return;
  if (!ctx.currentTrack) {
    await ctx.playCurrent();
    return;
  }

  try {
    if (!ctx.engine.canPlay()) {
      await ctx.playCurrent();
      return;
    }
    await ctx.engine.play();
    ctx.playing = true;
    pauseVideoForMusic();
    ctx.startSmoothProgress();
    ctx.syncMediaSession();
  } catch {
    await ctx.playCurrent();
  }
}

export function next(ctx: MusicPlaybackTransportContext) {
  if (ctx.queue.length === 0) return;
  const current = ctx.currentTrack;
  if (
    current &&
    ctx.continuousMode === "personal" &&
    !isInternetRadioTrack(current)
  ) {
    const positionMs = Math.floor(
      (ctx.engine?.currentTime ?? ctx.currentTime) * 1000,
    );
    const durationMs = (current.duration ?? 0) * 1000;
    if (wasTrackSkipped(positionMs, durationMs)) {
      notePersonalSkip(ctx.personalRadio, current);
    }
  }
  if (ctx.shuffle && ctx.queue.length > 1) {
    if (!ctx.advanceShuffleIndex()) {
      void ctx.maybeRefillContinuousQueue().then((extended) => {
        if (extended && ctx.advanceShuffleIndex()) return;
        ctx.stopAtQueueEnd();
      });
    }
    return;
  }
  if (
    !canAdvanceQueue(ctx.queueIndex, ctx.queue.length, ctx.repeat, ctx.shuffle)
  ) {
    void ctx.maybeRefillContinuousQueue().then((extended) => {
      if (!extended) {
        ctx.stopAtQueueEnd();
        return;
      }
      const nextIdx = nextSequentialIndex(
        ctx.queueIndex,
        ctx.queue.length,
        ctx.repeat,
      );
      if (nextIdx === null) {
        ctx.stopAtQueueEnd();
        return;
      }
      ctx.queueIndex = nextIdx;
      ctx.syncMediaSession();
      ctx.requestPlayCurrent();
    });
    return;
  }
  const nextIdx = nextSequentialIndex(
    ctx.queueIndex,
    ctx.queue.length,
    ctx.repeat,
  );
  if (nextIdx === null) {
    ctx.stopAtQueueEnd();
    return;
  }
  ctx.queueIndex = nextIdx;
  ctx.syncMediaSession();
  ctx.requestPlayCurrent();
}

export function previous(ctx: MusicPlaybackTransportContext) {
  if (ctx.queue.length === 0) return;
  if (ctx.engine && ctx.engine.currentTime > 3) {
    ctx.engine.currentTime = 0;
    ctx.currentTime = 0;
    ctx.smoothProgress = 0;
    updateMediaSessionPosition(ctx.duration, 0);
    return;
  }
  if (ctx.shuffle && ctx.queue.length > 1) {
    const step = popShuffleHistory(ctx.shuffleHistory, ctx.queueIndex);
    if (!step) return;
    ctx.shuffleHistory = step.history;
    ctx.shuffleUpcoming.unshift(step.prependUpcoming);
    ctx.queueIndex = step.previousIndex;
    ctx.syncMediaSession();
    ctx.requestPlayCurrent();
    return;
  }
  ctx.queueIndex =
    ctx.queueIndex <= 0 ? ctx.queue.length - 1 : ctx.queueIndex - 1;
  ctx.syncMediaSession();
  ctx.requestPlayCurrent();
}

export function cycleRepeat(ctx: MusicPlaybackTransportContext) {
  ctx.repeat =
    ctx.repeat === "off" ? "all" : ctx.repeat === "all" ? "one" : "off";
  ctx.prefetchAround();
}

export function setVolume(ctx: MusicPlaybackTransportContext, value: number) {
  if (!Number.isFinite(value)) return;
  ctx.volume = Math.max(0, Math.min(1, value));
  ctx.engine?.setVolume(ctx.volume);
  saveVolume(ctx.volume);
}

export function adjustVolume(
  ctx: MusicPlaybackTransportContext,
  delta: number,
) {
  setVolume(ctx, ctx.volume + delta);
}

export function seekBy(ctx: MusicPlaybackTransportContext, seconds: number) {
  if (!ctx.engine || !ctx.currentTrack) return;
  const max = ctx.duration || ctx.engine.duration || 0;
  const target = Math.max(
    0,
    Math.min(max || Infinity, ctx.currentTime + seconds),
  );
  seek(ctx, target);
}

export function seek(ctx: MusicPlaybackTransportContext, seconds: number) {
  if (ctx.engine) {
    ctx.engine.currentTime = seconds;
    ctx.currentTime = seconds;
    ctx.smoothProgress = ctx.duration > 0 ? (seconds / ctx.duration) * 100 : 0;
    updateMediaSessionPosition(ctx.duration, seconds);
    ctx.lastSavedPosition = Math.floor(seconds * 1000);
    ctx.persistPlaybackState();
    void ctx.saveProgress(ctx.lastSavedPosition, { deltaMs: 0 });
  }
}

export function createPlaybackTransportOps(ctx: MusicPlaybackTransportContext) {
  return {
    pause: () => pause(ctx),
    togglePlay: () => togglePlay(ctx),
    next: () => next(ctx),
    previous: () => previous(ctx),
    cycleRepeat: () => cycleRepeat(ctx),
    setVolume: (value: number) => setVolume(ctx, value),
    adjustVolume: (delta: number) => adjustVolume(ctx, delta),
    seek: (seconds: number) => seek(ctx, seconds),
    seekBy: (seconds: number) => seekBy(ctx, seconds),
  };
}
