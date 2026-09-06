// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { createBoundedSet } from "$lib/core/bounded-cache";
import * as musicApi from "$lib/music/api";
import { prefetchNowPlayingCoverArt } from "$lib/music/cover-art-prefetch";
import { LOAD_SUPERSEDED } from "$lib/music/native-audio-engine";
import type { PlaybackEngine } from "$lib/music/playback-engine";
import {
  crossfadeActive,
  type PlaybackSettings,
} from "$lib/music/playback-settings";
import { savePlayback } from "$lib/music/prefs";
import {
  buildStreamOptions,
  type TranscodingSettings,
} from "$lib/music/transcoding-settings";
import type { ImmersiveAudioSettings } from "$lib/music/immersive-audio-settings";
import {
  downloadStreamUrl,
  isInternetRadioTrack,
  radioStreamUrl,
  streamUrl,
  type QueueTrack,
  type SubsonicSong,
} from "$lib/subsonic";
import type { ListenEntry, SubsonicConfig } from "$lib/subsonic/types";
import { resolveMediaUrl } from "$lib/config/runtime";
import { ApiPaths } from "$lib/core/http/api-paths";
import type { ContinuousMode } from "$lib/music/continuous-pool";
import { pauseVideoForMusic } from "$lib/video/playback-gate.svelte";

export interface MusicPlaybackCoreContext {
  playbackEpoch: number;
  resumeOnPlay: boolean;
  playCurrentTail: Promise<void>;
  pendingStartAt: number | null;
  pendingStartPaused: boolean;
  transcodedTrackIds: Set<string>;
  failedTrackSkips: number;
  playing: boolean;
  currentTrack: QueueTrack | null;
  currentTime: number;
  duration: number;
  smoothProgress: number;
  engine: PlaybackEngine | null;
  nextTrackPrepared: boolean;
  playbackSettings: PlaybackSettings;
  nativePlayback: boolean;
  error: string | null;
  resumeTracks: ListenEntry[];
  listenHistory: ListenEntry[];
  downloadedIds: Set<string>;
  config: SubsonicConfig;
  transcodingSettings: TranscodingSettings;
  immersiveAudioSettings: ImmersiveAudioSettings;
  lyricsOpen: boolean;
  queue: QueueTrack[];
  queueIndex: number;
  shuffle: boolean;
  autoplay: boolean;
  continuousMode: ContinuousMode;
  lastSavedPosition: number;
  syncMediaSession(): void;
  initEngine(): Promise<void>;
  loadTrackSource(token: number, track: QueueTrack, url: string): Promise<void>;
  handlePlaybackNetworkFailure(
    token: number,
    options?: { resume?: boolean },
  ): Promise<void>;
  skipFailedTrack(token: number, msg?: string): void;
  prefetchAround(): void;
  startProgressTracking(): void;
  startSmoothProgress(): void;
  recordNowPlaying(track: QueueTrack): Promise<void>;
  enrichCurrentTrack(trackId: string, token: number): Promise<void>;
  maybeCacheTrack(track: QueueTrack): void;
  loadCurrentLyrics(): Promise<void>;
}

export function requestPlayCurrent(ctx: MusicPlaybackCoreContext) {
  const token = ++ctx.playbackEpoch;
  const resume = ctx.resumeOnPlay;
  ctx.resumeOnPlay = false;
  void playCurrent(ctx, token, resume);
}

export function markTrackTranscoded(
  ctx: MusicPlaybackCoreContext,
  trackId: string,
) {
  const next = createBoundedSet<string>(64);
  for (const id of ctx.transcodedTrackIds) {
    next.add(id);
  }
  next.add(trackId);
  ctx.transcodedTrackIds = next;
}

export async function playCurrent(
  ctx: MusicPlaybackCoreContext,
  epoch?: number,
  resume = false,
) {
  const token = epoch ?? ++ctx.playbackEpoch;
  const run = async () => {
    if (token !== ctx.playbackEpoch) return;
    const startAt = ctx.pendingStartAt;
    const startPaused = ctx.pendingStartPaused;
    ctx.pendingStartAt = null;
    ctx.pendingStartPaused = false;
    await playCurrentWork(ctx, token, resume, startAt, startPaused);
  };
  ctx.playCurrentTail = ctx.playCurrentTail.then(run, run);
  return ctx.playCurrentTail;
}

export function isSupersededPlaybackError(
  ctx: MusicPlaybackCoreContext,
  message: string,
): boolean {
  return message === LOAD_SUPERSEDED;
}

export function onPlaybackStarted(
  ctx: MusicPlaybackCoreContext,
  track: QueueTrack,
  token: number,
  options: { incrementPlay?: boolean } = {},
) {
  ctx.failedTrackSkips = 0;
  ctx.playing = true;
  pauseVideoForMusic();
  ctx.startProgressTracking();
  ctx.startSmoothProgress();
  ctx.syncMediaSession();
  void ctx.recordNowPlaying(track);
  if (options.incrementPlay) {
    void saveProgress(ctx, 0, { incrementPlay: true });
  }
  void ctx.enrichCurrentTrack(track.id, token);
  if (!isInternetRadioTrack(track)) {
    void ctx.maybeCacheTrack(track);
  }
  if (ctx.lyricsOpen) void ctx.loadCurrentLyrics();
}

export async function playCurrentWork(
  ctx: MusicPlaybackCoreContext,
  token: number,
  resume = false,
  startAt: number | null = null,
  startPaused = false,
) {
  const track = ctx.currentTrack;
  if (!track) return;

  // Stop video as soon as a track load begins, not only after audio starts.
  pauseVideoForMusic();
  ctx.syncMediaSession();

  prefetchNowPlayingCoverArt(ctx.config, track);

  await ctx.initEngine();
  if (!ctx.engine) return;

  const url = trackStreamUrl(ctx, track);
  const seekBeforePlay =
    startAt != null && startAt > 0.05 && !isInternetRadioTrack(track);

  if (ctx.engine && !ctx.engine.hasPrepared(url)) {
    ctx.nextTrackPrepared = false;
    ctx.engine.prepareNext("");
  }

  let startedGapless = false;
  const crossfadeSec = crossfadeActive(ctx.playbackSettings, ctx.nativePlayback)
    ? ctx.playbackSettings.crossfadeDurationSec
    : 0;
  if (!seekBeforePlay && !startPaused) {
    try {
      startedGapless = await ctx.engine.activatePrepared(url, crossfadeSec);
    } catch {
      /* fall back to a fresh load below */
    }
  }

  if (token !== ctx.playbackEpoch) return;

  if (!startedGapless) {
    try {
      await ctx.loadTrackSource(token, track, url);
    } catch (err) {
      if (token !== ctx.playbackEpoch) return;
      const msg = err instanceof Error ? err.message : "Failed to load track";
      if (isSupersededPlaybackError(ctx, msg)) return;
      if (ctx.engine.isNetworkError()) {
        void ctx.handlePlaybackNetworkFailure(token, { resume: true });
        return;
      }
      ctx.playing = false;
      if (isInternetRadioTrack(track)) {
        ctx.error = msg;
        return;
      }
      ctx.skipFailedTrack(token, msg);
      return;
    }
  }

  if (token !== ctx.playbackEpoch) return;

  ctx.nextTrackPrepared = false;
  ctx.engine?.prepareNext("");

  if (startedGapless) {
    ctx.prefetchAround();
    onPlaybackStarted(ctx, track, token, { incrementPlay: true });
    return;
  }

  if (seekBeforePlay && ctx.engine && startAt != null) {
    ctx.engine.currentTime = startAt;
    ctx.currentTime = startAt;
    ctx.smoothProgress = ctx.duration > 0 ? (startAt / ctx.duration) * 100 : 0;
  } else if (resume && !isInternetRadioTrack(track)) {
    const current = ctx.currentTrack;
    const entry =
      current &&
      (ctx.resumeTracks.find((e) => e.trackId === current.id) ??
        ctx.listenHistory.find((e) => e.trackId === current.id));
    if (entry && entry.positionMs > 5000 && !entry.played) {
      ctx.engine.currentTime = entry.positionMs / 1000;
    }
  }

  if (startPaused) {
    ctx.engine.pause();
    ctx.playing = false;
    ctx.syncMediaSession();
    return;
  }

  try {
    await ctx.engine.play();
    if (token !== ctx.playbackEpoch) return;
    ctx.prefetchAround();
    const startPos = ctx.engine.currentTime * 1000;
    onPlaybackStarted(ctx, track, token, {
      incrementPlay: startPos < 5000 && !isInternetRadioTrack(track),
    });
  } catch (err) {
    if (token !== ctx.playbackEpoch) return;
    const msg = err instanceof Error ? err.message : "Playback failed";
    if (isSupersededPlaybackError(ctx, msg)) return;
    if (ctx.engine.isNetworkError()) {
      void ctx.handlePlaybackNetworkFailure(token, { resume: true });
      return;
    }
    ctx.playing = false;
    if (
      !isInternetRadioTrack(track) &&
      !ctx.nativePlayback &&
      ctx.engine.isDecodeError() &&
      !ctx.transcodedTrackIds.has(track.id)
    ) {
      markTrackTranscoded(ctx, track.id);
      try {
        await ctx.loadTrackSource(token, track, trackStreamUrl(ctx, track));
        if (seekBeforePlay && ctx.engine && startAt != null) {
          ctx.engine.currentTime = startAt;
          ctx.currentTime = startAt;
        }
        await ctx.engine.play();
        if (token !== ctx.playbackEpoch) return;
        ctx.prefetchAround();
        const startPos = ctx.engine.currentTime * 1000;
        onPlaybackStarted(ctx, track, token, {
          incrementPlay: startPos < 5000 && !isInternetRadioTrack(track),
        });
        return;
      } catch (retryErr) {
        const retryMsg =
          retryErr instanceof Error ? retryErr.message : "Playback failed";
        if (isInternetRadioTrack(track)) {
          ctx.error = retryMsg;
          return;
        }
        ctx.skipFailedTrack(token, retryMsg);
        return;
      }
    }
    if (isInternetRadioTrack(track)) {
      ctx.error = msg;
      return;
    }
    ctx.skipFailedTrack(token, msg);
  }
}

export function trackStreamUrl(
  ctx: MusicPlaybackCoreContext,
  track: QueueTrack,
): string {
  if (isInternetRadioTrack(track) && track.streamUrl) {
    return radioStreamUrl(track);
  }
  if (track.streamUrl?.includes(ApiPaths.partyPrefix)) {
    return resolveMediaUrl(track.streamUrl);
  }
  if (ctx.downloadedIds.has(track.id)) {
    return downloadStreamUrl(track.id);
  }
  const transcode = ctx.transcodedTrackIds.has(track.id);
  const options = buildStreamOptions(ctx.transcodingSettings, transcode, {
    track,
    preserveImmersiveStreams:
      ctx.immersiveAudioSettings.preserveImmersiveStreams,
  });
  return streamUrl(ctx.config, track.id, options);
}

export async function saveProgress(
  ctx: MusicPlaybackCoreContext,
  positionMs: number,
  options: {
    played?: boolean;
    incrementPlay?: boolean;
    deltaMs?: number;
  } = {},
  forTrack?: SubsonicSong,
) {
  const track = forTrack ?? ctx.currentTrack;
  if (!track || isInternetRadioTrack(track)) return;

  try {
    await musicApi.saveListenProgress(track.id, {
      positionMs,
      played: options.played ?? false,
      incrementPlay: options.incrementPlay ?? false,
      deltaMs: options.deltaMs ?? 0,
      trackTitle: track.title,
      artistName: track.artist ?? "",
      albumId: track.albumId ?? "",
      albumTitle: track.album ?? "",
      durationMs: (track.duration ?? 0) * 1000,
      coverArtId: track.coverArt ?? track.albumId ?? track.id,
    });
  } catch {
    /* best effort */
  }
}

export function persistPlaybackSnapshot(ctx: MusicPlaybackCoreContext): void {
  persistPlaybackState(ctx);
}

export async function flushPlaybackState(
  ctx: MusicPlaybackCoreContext,
): Promise<void> {
  if (!ctx.currentTrack || ctx.queueIndex < 0) return;
  const positionMs = Math.floor(
    (ctx.engine?.currentTime ?? ctx.currentTime) * 1000,
  );
  const deltaMs = Math.max(
    0,
    Math.min(15000, positionMs - ctx.lastSavedPosition),
  );
  ctx.lastSavedPosition = positionMs;
  persistPlaybackState(ctx);
  if (!isInternetRadioTrack(ctx.currentTrack)) {
    await saveProgress(ctx, positionMs, { deltaMs });
  }
}

export function persistPlaybackState(ctx: MusicPlaybackCoreContext) {
  if (ctx.queue.length === 0 || ctx.queueIndex < 0) return;
  savePlayback({
    trackIds: ctx.queue.map((t) => t.id),
    queueIndex: ctx.queueIndex,
    positionMs: Math.floor((ctx.engine?.currentTime ?? ctx.currentTime) * 1000),
    shuffle: ctx.shuffle,
    autoplay: ctx.autoplay,
    continuousMode: ctx.continuousMode,
    randomRadio: ctx.continuousMode === "random",
  });
}

export function createPlaybackCoreOps(ctx: MusicPlaybackCoreContext) {
  return {
    requestPlayCurrent: () => requestPlayCurrent(ctx),
    markTrackTranscoded: (trackId: string) => markTrackTranscoded(ctx, trackId),
    playCurrent: (epoch?: number, resume?: boolean) =>
      playCurrent(ctx, epoch, resume),
    isSupersededPlaybackError: (message: string) =>
      isSupersededPlaybackError(ctx, message),
    onPlaybackStarted: (
      track: QueueTrack,
      token: number,
      options?: { incrementPlay?: boolean },
    ) => onPlaybackStarted(ctx, track, token, options),
    playCurrentWork: (
      token: number,
      resume?: boolean,
      startAt?: number | null,
      startPaused?: boolean,
    ) => playCurrentWork(ctx, token, resume, startAt, startPaused),
    trackStreamUrl: (track: QueueTrack) => trackStreamUrl(ctx, track),
    saveProgress: (
      positionMs: number,
      options?: {
        played?: boolean;
        incrementPlay?: boolean;
        deltaMs?: number;
      },
      forTrack?: SubsonicSong,
    ) => saveProgress(ctx, positionMs, options, forTrack),
    persistPlaybackSnapshot: () => persistPlaybackSnapshot(ctx),
    flushPlaybackState: () => flushPlaybackState(ctx),
    persistPlaybackState: () => persistPlaybackState(ctx),
  };
}
