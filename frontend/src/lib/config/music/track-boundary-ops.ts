// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { connection } from "$lib/music/connection.svelte";
import * as musicApi from "$lib/music/api";
import { toRockskyTrack } from "$lib/music/rocksky";
import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import type { PlaybackEngine } from "$lib/music/playback-engine";
import {
  CONTINUOUS_REFILL_BATCH,
  CONTINUOUS_REFILL_THRESHOLD,
  continuousRefillCount,
  filterUniqueTracks,
  migrateContinuousMode,
  pullLibraryTracks,
  type ContinuousMode,
  type LibraryPoolState,
} from "$lib/music/continuous-pool";
import {
  buildPersonalProfile,
  seedPersonalRadioTracks,
  type PersonalRadioOptions,
  type PersonalRadioState,
} from "$lib/music/personal-radio";
import { loadSavedPlayback } from "$lib/music/prefs";
import {
  enforceQueueLimit,
  remainingQueueSlots,
  truncateTrackList,
  type QueueSettings,
} from "$lib/music/queue-settings";
import {
  shouldStopAtQueueEnd,
  shuffleNewIndices,
} from "$lib/music/playback-queue";
import {
  trackHasFullMetadata,
  trackNeedsLinkMetadata,
} from "$lib/music/track-metadata";
import {
  isInternetRadioTrack,
  type QueueTrack,
  type SubsonicSong,
} from "$lib/subsonic";
import type {
  InternetRadioStation,
  ListenEntry,
  ListenStats,
} from "$lib/subsonic/types";
import type { PlayerLayout } from "./types";

const MAX_FAILED_TRACK_SKIPS = 32;

export interface MusicTrackBoundaryContext {
  trackBoundaryBusy: boolean;
  crossfadeHandled: boolean;
  currentTrack: QueueTrack | null;
  playing: boolean;
  engine: PlaybackEngine | null;
  currentTime: number;
  playbackEpoch: number;
  smoothProgress: number;
  lastSavedPosition: number;
  playbackRestored: boolean;
  queue: QueueTrack[];
  queueIndex: number;
  shuffle: boolean;
  autoplay: boolean;
  continuousMode: ContinuousMode;
  repeat: "off" | "all" | "one";
  shuffleUpcoming: number[];
  shuffleHistory: number[];
  queueSettings: QueueSettings;
  continuousRefillInFlight: Promise<boolean> | null;
  libraryPool: LibraryPoolState;
  personalRadio: PersonalRadioState;
  listenHistory: ListenEntry[];
  stats: ListenStats | null;
  library: MusicLibraryAdapter;
  nativePlayback: boolean;
  transcodedTrackIds: Set<string>;
  failedTrackSkips: number;
  error: string | null;
  playerLayout: PlayerLayout;
  internetRadios: InternetRadioStation[];
  isDownloaded(trackId: string): boolean;
  trackStreamUrl(track: QueueTrack): string;
  loadTrackSource(token: number, track: QueueTrack, url: string): Promise<void>;
  startProgressTracking(): void;
  startSmoothProgress(): void;
  stopSmoothProgress(): void;
  syncMediaSession(): void;
  suspendForReconnect(): void;
  isSupersededPlaybackError(message: string): boolean;
  markTrackTranscoded(trackId: string): void;
  maybeRefillContinuousQueue(): Promise<boolean>;
  refillLibraryQueue(count: number): Promise<boolean>;
  refillPersonalQueue(count: number): Promise<boolean>;
  appendRandomSongsToQueue(count: number): Promise<boolean>;
  appendTracksToQueue(tracks: SubsonicSong[]): boolean;
  advanceTrack(): boolean;
  stopAtQueueEnd(): void;
  recordPlayCompletion(track: SubsonicSong): Promise<void>;
  personalRadioOptions(coldStart?: boolean): PersonalRadioOptions;
  entryToSong(entry: ListenEntry): SubsonicSong;
  persistPlaybackState(): void;
  seedShuffleUpcoming(): void;
  initEngine(): Promise<void>;
  resolveTracksForRestore(trackIds: string[]): Promise<SubsonicSong[]>;
  refreshInternetRadios(): Promise<void>;
}

export async function handlePlaybackNetworkFailure(
  ctx: MusicTrackBoundaryContext,
  token: number,
  options: { resume?: boolean } = {},
) {
  const track = ctx.currentTrack;
  if (!track || isInternetRadioTrack(track)) {
    if (ctx.playing) {
      ctx.suspendForReconnect();
      connection.onServerDisconnected("Playback stream failed");
    }
    return;
  }

  const shouldResume = options.resume ?? ctx.playing;

  if (ctx.isDownloaded(track.id)) {
    const cacheUrl = ctx.trackStreamUrl(track);
    const positionSec = ctx.engine?.currentTime ?? ctx.currentTime;
    try {
      await ctx.loadTrackSource(token, track, cacheUrl);
      if (token !== ctx.playbackEpoch) return;
      if (positionSec > 1) {
        ctx.engine!.currentTime = positionSec;
        ctx.currentTime = positionSec;
      }
      if (shouldResume) {
        await ctx.engine!.play();
        if (token !== ctx.playbackEpoch) return;
        ctx.playing = true;
        ctx.startProgressTracking();
        ctx.startSmoothProgress();
        ctx.syncMediaSession();
      }
      connection.onServerDisconnected(undefined, { silent: true });
      return;
    } catch {
      /* fall through */
    }
  }

  if (shouldResume) {
    ctx.suspendForReconnect();
    connection.onServerDisconnected("Playback stream failed");
  }
}

export async function loadTrackSource(
  ctx: MusicTrackBoundaryContext,
  token: number,
  track: QueueTrack,
  url: string,
): Promise<void> {
  if (!ctx.engine) return;
  try {
    await ctx.engine.loadSource(url);
  } catch (err) {
    if (token !== ctx.playbackEpoch) throw err;
    const msg = err instanceof Error ? err.message : "Failed to load track";
    if (ctx.isSupersededPlaybackError(msg)) return;
    if (
      !isInternetRadioTrack(track) &&
      !ctx.nativePlayback &&
      ctx.engine.isDecodeError() &&
      !ctx.transcodedTrackIds.has(track.id)
    ) {
      ctx.markTrackTranscoded(track.id);
      await ctx.loadTrackSource(token, track, ctx.trackStreamUrl(track));
      return;
    }
    throw err;
  }
}

export async function enrichCurrentTrack(
  ctx: MusicTrackBoundaryContext,
  trackId: string,
  epoch?: number,
) {
  const idx = ctx.queue.findIndex((t) => t.id === trackId);
  if (idx < 0 || isInternetRadioTrack(ctx.queue[idx])) return;
  if (
    trackHasFullMetadata(ctx.queue[idx]) &&
    !trackNeedsLinkMetadata(ctx.queue[idx])
  )
    return;
  try {
    const detail = await ctx.library.getSong(trackId);
    if (epoch !== undefined && epoch !== ctx.playbackEpoch) return;
    if (!detail || idx < 0 || idx >= ctx.queue.length) return;
    if (ctx.queue[idx]?.id !== trackId) return;
    const next = ctx.queue.slice();
    next[idx] = { ...next[idx], ...detail };
    ctx.queue = next;
  } catch {
    /* metadata enrichment is best-effort */
  }
}

export async function recordNowPlaying(
  ctx: MusicTrackBoundaryContext,
  track: SubsonicSong,
) {
  if (isInternetRadioTrack(track)) return;
  await ctx.library.scrobble(track.id, false).catch(() => {
    /* now-playing notification is best-effort */
  });
  await musicApi.rockskyNowPlaying(toRockskyTrack(track)).catch(() => {
    /* rocksky now playing is best-effort */
  });
  await musicApi.listenbrainzNowPlaying(toRockskyTrack(track)).catch(() => {
    /* listenbrainz now playing is best-effort */
  });
  await musicApi.lastfmNowPlaying(toRockskyTrack(track)).catch(() => {
    /* last.fm now playing is best-effort */
  });
}

export async function onTrackEnded(ctx: MusicTrackBoundaryContext) {
  if (ctx.trackBoundaryBusy) return;
  ctx.trackBoundaryBusy = true;
  try {
    if (ctx.crossfadeHandled) {
      ctx.crossfadeHandled = false;
      return;
    }

    const finished = ctx.currentTrack;

    if (ctx.repeat === "one") {
      if (ctx.engine) {
        ctx.engine.currentTime = 0;
        void ctx.engine.play();
      }
      if (finished) void ctx.recordPlayCompletion(finished);
      return;
    }

    const atEnd = ctx.queueIndex >= ctx.queue.length - 1;
    const shuffleNeedsRefill =
      ctx.shuffle &&
      ctx.autoplay &&
      ctx.shuffleUpcoming.length <= CONTINUOUS_REFILL_THRESHOLD;
    let queueExtended = false;

    if (
      ctx.continuousMode !== "off" &&
      finished &&
      !isInternetRadioTrack(finished)
    ) {
      queueExtended = await ctx.maybeRefillContinuousQueue();
    } else if (
      (atEnd || shuffleNeedsRefill) &&
      ctx.repeat === "off" &&
      ctx.autoplay &&
      finished &&
      !isInternetRadioTrack(finished)
    ) {
      queueExtended = await ctx.appendRandomSongsToQueue(
        CONTINUOUS_REFILL_BATCH,
      );
    }

    const playbackExhausted = ctx.shuffle
      ? ctx.shuffleUpcoming.length === 0 && !queueExtended
      : atEnd;

    if (shouldStopAtQueueEnd(playbackExhausted, ctx.repeat, queueExtended)) {
      ctx.stopAtQueueEnd();
      if (finished) void ctx.recordPlayCompletion(finished);
      return;
    }

    if (!ctx.advanceTrack()) {
      ctx.stopAtQueueEnd();
    }
    if (finished) void ctx.recordPlayCompletion(finished);
  } finally {
    ctx.trackBoundaryBusy = false;
  }
}

export async function maybeRefillContinuousQueue(
  ctx: MusicTrackBoundaryContext,
): Promise<boolean> {
  if (ctx.continuousMode === "off" || !ctx.autoplay) return false;
  const count = continuousRefillCount(
    ctx.queue.length,
    ctx.queueSettings.maxQueueSize,
    ctx.continuousMode,
  );
  if (count <= 0) return false;

  if (ctx.continuousMode === "library") {
    return ctx.refillLibraryQueue(count);
  }
  if (ctx.continuousMode === "personal") {
    return ctx.refillPersonalQueue(count);
  }
  return ctx.appendRandomSongsToQueue(count);
}

export async function refillLibraryQueue(
  ctx: MusicTrackBoundaryContext,
  count: number,
): Promise<boolean> {
  if (ctx.continuousRefillInFlight) {
    return ctx.continuousRefillInFlight;
  }

  const run = async (): Promise<boolean> => {
    const existingIds = new Set(ctx.queue.map((track) => track.id));
    const more = await pullLibraryTracks(
      ctx.libraryPool,
      {
        getAlbumList2: (type, size, offset) =>
          ctx.library.getAlbumList2(type, size, offset),
        getAlbumSongs: async (albumId) => {
          const album = await ctx.library.getAlbum(albumId).catch(() => null);
          return album?.songs ?? [];
        },
      },
      count,
      existingIds,
    );
    return ctx.appendTracksToQueue(more);
  };

  ctx.continuousRefillInFlight = run().finally(() => {
    ctx.continuousRefillInFlight = null;
  });
  return ctx.continuousRefillInFlight;
}

export async function refillPersonalQueue(
  ctx: MusicTrackBoundaryContext,
  count: number,
): Promise<boolean> {
  if (ctx.continuousRefillInFlight) {
    return ctx.continuousRefillInFlight;
  }

  const run = async (): Promise<boolean> => {
    const history = ctx.listenHistory;
    const stats = ctx.stats;
    const playCountByTrack = new Map<string, number>();
    const listenedMsByTrack = new Map<string, number>();
    for (const entry of history) {
      playCountByTrack.set(entry.trackId, entry.playCount);
      if ((entry.listenedMs ?? 0) > 0) {
        listenedMsByTrack.set(entry.trackId, entry.listenedMs);
      }
    }
    const profile = buildPersonalProfile(
      stats,
      history,
      playCountByTrack,
      new Set(),
      ctx.personalRadio.skipTrackIds,
      listenedMsByTrack,
    );
    const existingIds = new Set(ctx.queue.map((track) => track.id));
    const more = await seedPersonalRadioTracks(
      ctx.personalRadio,
      {
        getSimilarSongs: (trackId, c) =>
          ctx.library.getSimilarSongs(trackId, c).catch(() => []),
        getRandomSongs: (c) => ctx.library.getRandomSongs(c).catch(() => []),
        searchArtistSongs: async (artist, limit) => {
          const result = await ctx.library
            .search3(artist, limit)
            .catch(() => ({ songs: [] as SubsonicSong[] }));
          return result.songs.filter(
            (song) => song.artist?.toLowerCase() === artist.toLowerCase(),
          );
        },
      },
      profile,
      history,
      stats,
      count,
      existingIds,
      (entry) => ctx.entryToSong(entry),
      ctx.personalRadioOptions(false),
    );
    return ctx.appendTracksToQueue(more);
  };

  ctx.continuousRefillInFlight = run().finally(() => {
    ctx.continuousRefillInFlight = null;
  });
  return ctx.continuousRefillInFlight;
}

export function appendTracksToQueue(
  ctx: MusicTrackBoundaryContext,
  tracks: SubsonicSong[],
): boolean {
  const slots = remainingQueueSlots(
    ctx.queue.length,
    ctx.queueSettings.maxQueueSize,
  );
  if (slots <= 0) return false;

  const existingIds = new Set(ctx.queue.map((track) => track.id));
  const unique = filterUniqueTracks(tracks, existingIds).slice(
    0,
    Number.isFinite(slots) ? slots : tracks.length,
  );
  if (unique.length === 0) return false;

  const prevLength = ctx.queue.length;
  let nextQueue = [...ctx.queue, ...unique.map((t) => ({ ...t }))];
  let nextIndex = ctx.queueIndex;
  const enforced = enforceQueueLimit(
    nextQueue,
    nextIndex,
    ctx.queueSettings.maxQueueSize,
  );
  nextQueue = enforced.queue;
  nextIndex = enforced.queueIndex;
  ctx.queue = nextQueue;
  ctx.queueIndex = nextIndex;
  if (ctx.shuffle) {
    ctx.shuffleUpcoming.push(
      ...shuffleNewIndices(prevLength, ctx.queue.length, ctx.queueIndex),
    );
  }
  ctx.persistPlaybackState();
  return true;
}

export async function appendRandomSongsToQueue(
  ctx: MusicTrackBoundaryContext,
  count: number,
): Promise<boolean> {
  if (ctx.continuousRefillInFlight) {
    return ctx.continuousRefillInFlight;
  }

  const run = async (): Promise<boolean> => {
    const slots = remainingQueueSlots(
      ctx.queue.length,
      ctx.queueSettings.maxQueueSize,
    );
    if (slots <= 0) return false;

    const batch = Math.min(count, slots);
    if (batch <= 0) return false;

    const more = await ctx.library
      .getRandomSongs(batch)
      .catch(() => [] as SubsonicSong[]);
    if (more.length === 0) return false;

    return ctx.appendTracksToQueue(more);
  };

  ctx.continuousRefillInFlight = run().finally(() => {
    ctx.continuousRefillInFlight = null;
  });
  return ctx.continuousRefillInFlight;
}

export function skipFailedTrack(
  ctx: MusicTrackBoundaryContext,
  epoch: number,
  reason?: string,
) {
  if (epoch !== ctx.playbackEpoch) return;
  ctx.failedTrackSkips += 1;
  const limit = Math.max(1, Math.min(ctx.queue.length, MAX_FAILED_TRACK_SKIPS));
  if (ctx.failedTrackSkips >= limit) {
    ctx.failedTrackSkips = 0;
    ctx.error = reason ?? "Unable to play queued tracks";
    ctx.stopAtQueueEnd();
    return;
  }
  if (!ctx.advanceTrack()) {
    ctx.failedTrackSkips = 0;
    ctx.stopAtQueueEnd();
  }
}

export async function restorePlayback(ctx: MusicTrackBoundaryContext) {
  const saved = loadSavedPlayback();
  if (!saved || saved.trackIds.length === 0) return;

  ctx.playbackRestored = true;
  const targetTrackId = saved.trackIds[saved.queueIndex] ?? saved.trackIds[0];
  const tracks = await ctx.resolveTracksForRestore(saved.trackIds);
  if (tracks.length === 0) return;

  const limited = truncateTrackList(tracks, ctx.queueSettings.maxQueueSize);
  ctx.queue = limited;
  const restoredIndex = targetTrackId
    ? limited.findIndex((track) => track.id === targetTrackId)
    : -1;
  ctx.queueIndex =
    restoredIndex >= 0
      ? restoredIndex
      : Math.min(Math.max(0, saved.queueIndex), limited.length - 1);
  ctx.shuffle = saved.shuffle;
  ctx.autoplay = saved.autoplay;
  ctx.continuousMode = migrateContinuousMode(saved);
  ctx.shuffleHistory = [];
  if (ctx.shuffle) {
    ctx.seedShuffleUpcoming();
  } else {
    ctx.shuffleUpcoming = [];
    ctx.shuffleHistory = [];
  }
  await ctx.initEngine();
  if (!ctx.engine) return;

  const track = ctx.currentTrack;
  if (!track) return;

  try {
    await ctx.engine.loadSource(ctx.trackStreamUrl(track));
  } catch {
    return;
  }

  const progress = await musicApi
    .getListenProgressBatch([track.id])
    .catch(() => new Map<string, ListenEntry>());
  const serverEntry = progress.get(track.id);
  const serverPos =
    serverEntry && !serverEntry.played && serverEntry.positionMs > 1000
      ? serverEntry.positionMs
      : 0;
  const positionMs = Math.max(saved.positionMs, serverPos);

  if (!isInternetRadioTrack(track) && positionMs > 1000) {
    ctx.engine.currentTime = positionMs / 1000;
    ctx.currentTime = positionMs / 1000;
    const duration = track.duration ?? 0;
    ctx.smoothProgress = duration > 0 ? (ctx.currentTime / duration) * 100 : 0;
    ctx.lastSavedPosition = positionMs;
  }
  ctx.persistPlaybackState();
  ctx.playerLayout = "full";
  ctx.syncMediaSession();
}

export function createTrackBoundaryOps(ctx: MusicTrackBoundaryContext) {
  return {
    handlePlaybackNetworkFailure: (
      token: number,
      options?: { resume?: boolean },
    ) => handlePlaybackNetworkFailure(ctx, token, options),
    loadTrackSource: (token: number, track: QueueTrack, url: string) =>
      loadTrackSource(ctx, token, track, url),
    enrichCurrentTrack: (trackId: string, epoch?: number) =>
      enrichCurrentTrack(ctx, trackId, epoch),
    recordNowPlaying: (track: SubsonicSong) => recordNowPlaying(ctx, track),
    onTrackEnded: () => onTrackEnded(ctx),
    maybeRefillContinuousQueue: () => maybeRefillContinuousQueue(ctx),
    refillLibraryQueue: (count: number) => refillLibraryQueue(ctx, count),
    refillPersonalQueue: (count: number) => refillPersonalQueue(ctx, count),
    appendTracksToQueue: (tracks: SubsonicSong[]) =>
      appendTracksToQueue(ctx, tracks),
    appendRandomSongsToQueue: (count: number) =>
      appendRandomSongsToQueue(ctx, count),
    skipFailedTrack: (epoch: number, reason?: string) =>
      skipFailedTrack(ctx, epoch, reason),
    restorePlayback: () => restorePlayback(ctx),
  };
}
