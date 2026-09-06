// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { ContinuousMode } from "$lib/music/continuous-pool";
import type { PlaybackEngine } from "$lib/music/playback-engine";
import {
  playNextShuffleUpcoming,
  queueIndexAfterRemove,
  shiftIndicesForInsert,
  shuffleNewIndices,
} from "$lib/music/playback-queue";
import {
  remainingQueueSlots,
  type QueueSettings,
} from "$lib/music/queue-settings";
import { clearMediaSession } from "$lib/music/media-session";
import type { QueueTrack, SubsonicSong } from "$lib/subsonic";
import { toast } from "$lib/ui/toast.svelte";
import type { PlayerLayout } from "./types";

export interface MusicQueueContext {
  queue: QueueTrack[];
  queueIndex: number;
  shuffle: boolean;
  shuffleUpcoming: number[];
  shuffleHistory: number[];
  queueSettings: QueueSettings;
  engine: PlaybackEngine | null;
  playing: boolean;
  continuousMode: ContinuousMode;
  queueOpen: boolean;
  currentTrack: QueueTrack | null;
  playerLayout: PlayerLayout;
  pendingStartAt: number | null;
  pendingStartPaused: boolean;
  playTracks(tracks: SubsonicSong[], startIndex?: number): void;
  prefetchAround(): void;
  persistPlaybackState(): void;
  requestPlayCurrent(): void;
  seedShuffleUpcoming(): void;
  stopSmoothProgress(): void;
}

export function removeFromQueue(ctx: MusicQueueContext, index: number) {
  if (index < 0 || index >= ctx.queue.length) return;
  ctx.queue = ctx.queue.filter((_, i) => i !== index);
  const adjusted = queueIndexAfterRemove(
    ctx.queueIndex,
    index,
    ctx.queue.length,
  );
  ctx.queueIndex = adjusted.queueIndex;
  if (ctx.queue.length === 0) {
    ctx.shuffleUpcoming = [];
    ctx.shuffleHistory = [];
    ctx.engine?.pause();
    ctx.playing = false;
    ctx.persistPlaybackState();
    return;
  }
  if (ctx.shuffle) {
    ctx.shuffleHistory = [];
    ctx.seedShuffleUpcoming();
  }
  if (adjusted.restart) {
    ctx.requestPlayCurrent();
  }
  ctx.prefetchAround();
  ctx.persistPlaybackState();
}

/** playNext inserts a track right after the current one. */
export function playNext(ctx: MusicQueueContext, track: SubsonicSong) {
  const item: QueueTrack = { ...track };
  if (ctx.queue.length === 0 || ctx.queueIndex < 0) {
    ctx.playTracks([track], 0);
    return;
  }
  if (
    remainingQueueSlots(ctx.queue.length, ctx.queueSettings.maxQueueSize) < 1
  ) {
    toast.warning("Queue is full");
    return;
  }
  const insertAt = ctx.queueIndex + 1;
  ctx.queue = [
    ...ctx.queue.slice(0, insertAt),
    item,
    ...ctx.queue.slice(insertAt),
  ];
  if (ctx.shuffle) {
    ctx.shuffleHistory = shiftIndicesForInsert(ctx.shuffleHistory, insertAt, 1);
    ctx.shuffleUpcoming = playNextShuffleUpcoming(
      ctx.shuffleUpcoming,
      insertAt,
      1,
    );
  }
  ctx.prefetchAround();
  ctx.persistPlaybackState();
  toast.success(`Playing next: ${track.title}`);
}

/** addToQueue appends a track to the end of the queue. */
export function addToQueue(ctx: MusicQueueContext, track: SubsonicSong) {
  if (ctx.queue.length === 0 || ctx.queueIndex < 0) {
    ctx.playTracks([track], 0);
    return;
  }
  if (
    remainingQueueSlots(ctx.queue.length, ctx.queueSettings.maxQueueSize) < 1
  ) {
    toast.warning("Queue is full");
    return;
  }
  const prevLength = ctx.queue.length;
  ctx.queue = [...ctx.queue, { ...track }];
  if (ctx.shuffle) {
    ctx.shuffleUpcoming.push(
      ...shuffleNewIndices(prevLength, ctx.queue.length, ctx.queueIndex),
    );
  }
  ctx.prefetchAround();
  ctx.persistPlaybackState();
  toast.success(`Added to queue: ${track.title}`);
}

/** addTracksToQueue appends multiple tracks to the end of the queue. */
export function addTracksToQueue(
  ctx: MusicQueueContext,
  tracks: SubsonicSong[],
) {
  if (tracks.length === 0) return;
  if (ctx.queue.length === 0 || ctx.queueIndex < 0) {
    ctx.playTracks(tracks, 0);
    return;
  }
  const slots = remainingQueueSlots(
    ctx.queue.length,
    ctx.queueSettings.maxQueueSize,
  );
  if (slots < 1) {
    toast.warning("Queue is full");
    return;
  }
  const toAdd = tracks.slice(0, slots);
  const prevLength = ctx.queue.length;
  ctx.queue = [...ctx.queue, ...toAdd.map((track) => ({ ...track }))];
  if (ctx.shuffle) {
    ctx.shuffleUpcoming.push(
      ...shuffleNewIndices(prevLength, ctx.queue.length, ctx.queueIndex),
    );
  }
  ctx.prefetchAround();
  ctx.persistPlaybackState();
  if (toAdd.length === 1) {
    toast.success(`Added to queue: ${toAdd[0].title}`);
  } else {
    toast.success(`Added ${toAdd.length} tracks to queue`);
  }
  if (toAdd.length < tracks.length) {
    toast.warning("Queue is full");
  }
}

/** playTracksNext inserts multiple tracks right after the current one. */
export function playTracksNext(ctx: MusicQueueContext, tracks: SubsonicSong[]) {
  if (tracks.length === 0) return;
  if (ctx.queue.length === 0 || ctx.queueIndex < 0) {
    ctx.playTracks(tracks, 0);
    return;
  }
  const slots = remainingQueueSlots(
    ctx.queue.length,
    ctx.queueSettings.maxQueueSize,
  );
  if (slots < 1) {
    toast.warning("Queue is full");
    return;
  }
  const toAdd = tracks.slice(0, slots);
  const insertAt = ctx.queueIndex + 1;
  ctx.queue = [
    ...ctx.queue.slice(0, insertAt),
    ...toAdd.map((track) => ({ ...track })),
    ...ctx.queue.slice(insertAt),
  ];
  if (ctx.shuffle) {
    ctx.shuffleHistory = shiftIndicesForInsert(
      ctx.shuffleHistory,
      insertAt,
      toAdd.length,
    );
    ctx.shuffleUpcoming = playNextShuffleUpcoming(
      ctx.shuffleUpcoming,
      insertAt,
      toAdd.length,
    );
  }
  ctx.prefetchAround();
  ctx.persistPlaybackState();
  if (toAdd.length === 1) {
    toast.success(`Playing next: ${toAdd[0].title}`);
  } else {
    toast.success(`Playing ${toAdd.length} tracks next`);
  }
  if (toAdd.length < tracks.length) {
    toast.warning("Queue is full");
  }
}

export function moveInQueue(ctx: MusicQueueContext, from: number, to: number) {
  if (
    from === to ||
    from < 0 ||
    to < 0 ||
    from >= ctx.queue.length ||
    to >= ctx.queue.length
  ) {
    return;
  }
  const currentId = ctx.currentTrack?.id;
  const next = [...ctx.queue];
  const [moved] = next.splice(from, 1);
  next.splice(to, 0, moved);
  ctx.queue = next;
  if (currentId) {
    const idx = next.findIndex((t) => t.id === currentId);
    if (idx >= 0) ctx.queueIndex = idx;
  }
  if (ctx.shuffle) {
    ctx.shuffleHistory = [];
    ctx.seedShuffleUpcoming();
  }
  ctx.prefetchAround();
  ctx.persistPlaybackState();
}

/** clearQueue stops playback and empties the queue. */
export function clearQueue(ctx: MusicQueueContext) {
  ctx.engine?.pause();
  ctx.playing = false;
  ctx.stopSmoothProgress();
  ctx.continuousMode = "off";
  ctx.queue = [];
  ctx.queueIndex = -1;
  ctx.shuffleUpcoming = [];
  ctx.shuffleHistory = [];
  ctx.queueOpen = false;
  clearMediaSession();
  ctx.persistPlaybackState();
}

export function playQueueIndex(ctx: MusicQueueContext, index: number) {
  if (index < 0 || index >= ctx.queue.length) return;
  ctx.queueIndex = index;
  ctx.shuffleHistory = [];
  if (ctx.shuffle) ctx.seedShuffleUpcoming();
  ctx.persistPlaybackState();
  ctx.requestPlayCurrent();
}

export function armStartPosition(
  ctx: MusicQueueContext,
  seconds: number,
  paused = false,
) {
  if (!Number.isFinite(seconds)) return;
  ctx.pendingStartAt = Math.max(0, seconds);
  ctx.pendingStartPaused = paused;
}

export function toggleQueue(ctx: MusicQueueContext) {
  ctx.queueOpen = !ctx.queueOpen;
}

export function createQueueOps(ctx: MusicQueueContext) {
  return {
    removeFromQueue: (index: number) => removeFromQueue(ctx, index),
    playNext: (track: SubsonicSong) => playNext(ctx, track),
    addToQueue: (track: SubsonicSong) => addToQueue(ctx, track),
    addTracksToQueue: (tracks: SubsonicSong[]) => addTracksToQueue(ctx, tracks),
    playTracksNext: (tracks: SubsonicSong[]) => playTracksNext(ctx, tracks),
    moveInQueue: (from: number, to: number) => moveInQueue(ctx, from, to),
    clearQueue: () => clearQueue(ctx),
    playQueueIndex: (index: number) => playQueueIndex(ctx, index),
    armStartPosition: (seconds: number, paused?: boolean) =>
      armStartPosition(ctx, seconds, paused),
    toggleQueue: () => toggleQueue(ctx),
  };
}
