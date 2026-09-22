// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from "vitest";
import type { QueueTrack, SubsonicSong } from "$lib/subsonic";
import type { ListenEntry } from "$lib/subsonic/types";
import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import {
  appendTracksToQueue,
  maybeRefillContinuousQueue,
  onTrackEnded,
  refillPersonalQueue,
  type MusicTrackBoundaryContext,
} from "./track-boundary-ops";
import { createPersonalRadioState } from "$lib/music/personal-radio";

function track(id: string, artist = `artist-${id}`): QueueTrack {
  return {
    id,
    title: id,
    artist,
    album: "alb",
    duration: 200,
  } as QueueTrack;
}

function historyEntry(trackId: string): ListenEntry {
  return {
    trackId,
    trackTitle: trackId,
    artistName: `Artist ${trackId}`,
    albumId: "al",
    albumTitle: "Album",
    positionMs: 0,
    durationMs: 180_000,
    played: true,
    playCount: 10,
    listenedMs: 500_000,
    lastPlayedAt: new Date(Date.now() - 86_400_000 * 3).toISOString(),
    coverArtId: "",
  };
}

function makeCtx(
  overrides: Record<string, unknown> = {},
): MusicTrackBoundaryContext {
  const library = {
    getSimilarSongs: vi.fn(async (id: string, count: number) =>
      Array.from({ length: count }, (_, i) =>
        track(`sim-${id}-${i}`, `similar-to-${id}`),
      ),
    ),
    getRandomSongs: vi.fn(async (count: number) =>
      Array.from({ length: count }, (_, i) =>
        track(`rand-${Math.random()}-${i}`),
      ),
    ),
    search3: vi.fn(async () => ({ songs: [] as SubsonicSong[] })),
    getSong: vi.fn(async () => null),
  } as unknown as MusicLibraryAdapter;
  const history = [historyEntry("hist-1"), historyEntry("hist-2")];
  const ctx = {
    continuousRefillInFlight: null as Promise<boolean> | null,
    trackBoundaryBusy: false,
    crossfadeHandled: false,
    queue: Array.from({ length: 25 }, (_, i) => track(`q-${i}`)),
    queueIndex: 0,
    currentTrack: track("q-0"),
    playing: true,
    shuffle: false,
    autoplay: true,
    repeat: "off",
    continuousMode: "personal",
    shuffleUpcoming: [] as number[],
    shuffleHistory: [] as number[],
    queueSettings: { maxQueueSize: 500 },
    personalRadio: createPersonalRadioState(),
    listenHistory: history,
    stats: null,
    library,
    preferLowBandwidth: false,
    engine: null,
    playbackEpoch: 1,
    currentTime: 0,
    smoothProgress: 0,
    lastSavedPosition: 0,
    playbackRestored: false,
    nativePlayback: false,
    transcodedTrackIds: new Set<string>(),
    failedTrackSkips: 0,
    error: null,
    playerLayout: "full",
    internetRadios: [],
    isDownloaded: () => false,
    trackStreamUrl: (t: QueueTrack) => `u-${t.id}`,
    loadTrackSource: vi.fn(async () => {}),
    startProgressTracking: vi.fn(),
    startSmoothProgress: vi.fn(),
    stopSmoothProgress: vi.fn(),
    syncMediaSession: vi.fn(),
    suspendForReconnect: vi.fn(),
    isSupersededPlaybackError: () => false,
    markTrackTranscoded: vi.fn(),
    advanceTrack: vi.fn(() => true),
    stopAtQueueEnd: vi.fn(),
    recordPlayCompletion: vi.fn(async () => {}),
    persistPlaybackState: vi.fn(),
    seedShuffleUpcoming: vi.fn(),
    initEngine: vi.fn(async () => {}),
    resolveTracksForRestore: vi.fn(async () => []),
    refreshInternetRadios: vi.fn(async () => {}),
    entryToSong: (e: ListenEntry) =>
      ({
        id: e.trackId,
        title: e.trackTitle,
        artist: e.artistName,
        album: e.albumTitle,
        albumId: e.albumId,
      }) as SubsonicSong,
    personalRadioOptions: () => ({
      recencyCooldownMs: 2 * 60 * 60 * 1000,
      exploreBonus: 0.6,
      albumLookback: 4,
      coldStart: false,
    }),
    ...overrides,
  } as unknown as MusicTrackBoundaryContext;
  ctx.maybeRefillContinuousQueue = () => maybeRefillContinuousQueue(ctx);
  ctx.refillPersonalQueue = (c: number) => refillPersonalQueue(ctx, c);
  ctx.appendTracksToQueue = (t: SubsonicSong[]) => appendTracksToQueue(ctx, t);
  ctx.appendRandomSongsToQueue = vi.fn(async () => false);
  ctx.refillLibraryQueue = vi.fn(async () => false);
  ctx.refillForeverQueue = vi.fn(async () => false);
  return ctx;
}

describe("personal radio refill", () => {
  afterEach(() => vi.useRealTimers());

  it("extends the queue when a track ends in personal mode", async () => {
    const ctx = makeCtx({ queueIndex: 23, currentTrack: track("q-23") });
    const before = ctx.queue.length;

    await onTrackEnded(ctx);
    await vi.waitFor(() => {
      expect(ctx.queue.length).toBeGreaterThan(before);
    });
  });

  it("appends seeded tracks on a direct refill", async () => {
    const ctx = makeCtx();
    const ok = await refillPersonalQueue(ctx, 10);
    expect(ok).toBe(true);
    expect(ctx.queue.length).toBeGreaterThan(25);
  });

  it("falls back to random draws when the seed pool is empty", async () => {
    const ctx = makeCtx({
      library: {
        getSimilarSongs: vi.fn(async () => []),
        getRandomSongs: vi.fn(async (count: number) =>
          Array.from({ length: count }, (_, i) =>
            track(`fresh-${Math.random()}-${i}`),
          ),
        ),
        search3: vi.fn(async () => ({ songs: [] })),
        getSong: vi.fn(async () => null),
      },
      listenHistory: [] as ListenEntry[],
    });

    const ok = await refillPersonalQueue(ctx, 8);

    expect(ok).toBe(true);
    expect(ctx.queue.length).toBeGreaterThan(25);
  });

  it("releases the in-flight guard after the refill timeout", async () => {
    vi.useFakeTimers();
    let calls = 0;
    const hanging = vi.fn(
      () =>
        new Promise<SubsonicSong[]>(() => {
          calls += 1;
        }),
    );
    const ctx = makeCtx({
      library: {
        getSimilarSongs: hanging,
        getRandomSongs: hanging,
        search3: hanging,
        getSong: hanging,
      },
    });

    const first = refillPersonalQueue(ctx, 5);
    const inFlight = ctx.continuousRefillInFlight;
    expect(inFlight).not.toBeNull();
    const second = refillPersonalQueue(ctx, 5);
    expect(ctx.continuousRefillInFlight).toBe(inFlight);
    // The wedged fetch never resolves; the guard must still expire so the
    // next track end can retry instead of silently skipping every refill.
    vi.advanceTimersByTime(25_000);
    expect(await first).toBe(false);
    expect(await second).toBe(false);
    expect(ctx.continuousRefillInFlight).toBeNull();

    const retry = refillPersonalQueue(ctx, 5);
    expect(retry).not.toBe(first);
    vi.advanceTimersByTime(25_000);
    expect(await retry).toBe(false);
    expect(calls).toBeGreaterThan(0);
  });
});
