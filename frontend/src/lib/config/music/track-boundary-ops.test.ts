// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { QueueTrack, SubsonicSong } from "$lib/subsonic";
import { connection } from "$lib/music/connection.svelte";
import { OFFLINE_SUSPECT_SKIP_THRESHOLD } from "$lib/music/offline-gate";
import {
  appendRandomSongsToQueue,
  maybeRefillContinuousQueue,
  onTrackEnded,
  skipFailedTrack,
  type MusicTrackBoundaryContext,
} from "./track-boundary-ops";

function track(id: string): QueueTrack {
  return { id, title: id, artist: "a", album: "b", duration: 1 } as QueueTrack;
}

interface BoundaryMocks {
  advanceTrack: ReturnType<typeof vi.fn>;
  stopAtQueueEnd: ReturnType<typeof vi.fn>;
  suspendForReconnect: ReturnType<typeof vi.fn>;
  persistPlaybackState: ReturnType<typeof vi.fn>;
}

function makeCtx(
  overrides: Record<string, unknown> = {},
): MusicTrackBoundaryContext & BoundaryMocks {
  const mocks: BoundaryMocks = {
    advanceTrack: vi.fn(() => true),
    stopAtQueueEnd: vi.fn(),
    suspendForReconnect: vi.fn(),
    persistPlaybackState: vi.fn(),
  };
  const ctx = {
    playbackEpoch: 7,
    failedTrackSkips: 0,
    queue: [track("a"), track("b"), track("c")],
    queueIndex: 0,
    currentTrack: track("a"),
    playing: false,
    error: null,
    ...mocks,
    ...overrides,
  } as unknown as MusicTrackBoundaryContext & BoundaryMocks;
  return ctx;
}

function resetConnection() {
  connection.dispose();
  connection.managed = false;
  connection.serverOnline = false;
  connection.browserOnline = true;
  connection.phase = "offline";
  connection.reconnectAttempt = 0;
  connection.lastDisconnectedAt = null;
  connection.lastConnectedAt = null;
  connection.nextRetryAt = null;
  connection.history = [];
}

describe("skipFailedTrack", () => {
  beforeEach(resetConnection);
  afterEach(() => connection.dispose());

  it("advances past a failed track while the server is online", () => {
    connection.managed = true;
    connection.serverOnline = true;
    const ctx = makeCtx();

    skipFailedTrack(ctx, 7, "decode failed");

    expect(ctx.advanceTrack).toHaveBeenCalledTimes(1);
    expect(ctx.suspendForReconnect).not.toHaveBeenCalled();
    expect(ctx.failedTrackSkips).toBe(1);
  });

  it("ignores a stale playback epoch", () => {
    connection.managed = true;
    connection.serverOnline = true;
    const ctx = makeCtx();

    skipFailedTrack(ctx, 99, "decode failed");

    expect(ctx.advanceTrack).not.toHaveBeenCalled();
    expect(ctx.failedTrackSkips).toBe(0);
  });

  it("does not hold for unmanaged sources", () => {
    // Local-only and demo sources never init the connection store, so
    // failed tracks keep the classic skip behavior there.
    connection.managed = false;
    connection.serverOnline = false;
    const ctx = makeCtx();

    skipFailedTrack(ctx, 7, "decode failed");

    expect(ctx.advanceTrack).toHaveBeenCalledTimes(1);
    expect(ctx.suspendForReconnect).not.toHaveBeenCalled();
  });

  it("holds the queue when the server is known unreachable", () => {
    connection.managed = true;
    connection.serverOnline = false;
    const ctx = makeCtx();

    skipFailedTrack(ctx, 7, "stream failed");

    expect(ctx.advanceTrack).not.toHaveBeenCalled();
    expect(ctx.suspendForReconnect).toHaveBeenCalledTimes(1);
    expect(ctx.persistPlaybackState).toHaveBeenCalledTimes(1);
    expect(ctx.failedTrackSkips).toBe(0);
    expect(ctx.error).toBeNull();
  });

  it("holds the queue when the browser reports offline", () => {
    connection.managed = true;
    connection.serverOnline = true;
    connection.browserOnline = false;
    const ctx = makeCtx();

    skipFailedTrack(ctx, 7, "stream failed");

    expect(ctx.advanceTrack).not.toHaveBeenCalled();
    expect(ctx.suspendForReconnect).toHaveBeenCalledTimes(1);
  });

  it("treats a run of consecutive failures as outage evidence", () => {
    connection.managed = true;
    connection.serverOnline = true;
    const ctx = makeCtx();

    for (let i = 1; i < OFFLINE_SUSPECT_SKIP_THRESHOLD; i += 1) {
      skipFailedTrack(ctx, 7, `failure ${i}`);
    }
    expect(ctx.advanceTrack).toHaveBeenCalledTimes(
      OFFLINE_SUSPECT_SKIP_THRESHOLD - 1,
    );
    expect(connection.serverOnline).toBe(true);

    skipFailedTrack(ctx, 7, "failure at threshold");

    // The threshold marks the server disconnected and parks the queue on the
    // failed track instead of advancing into more failures.
    expect(connection.serverOnline).toBe(false);
    expect(ctx.advanceTrack).toHaveBeenCalledTimes(
      OFFLINE_SUSPECT_SKIP_THRESHOLD - 1,
    );
    expect(ctx.suspendForReconnect).toHaveBeenCalledTimes(1);
    expect(ctx.failedTrackSkips).toBe(0);
  });

  it("stops advancing after the outage is marked", () => {
    connection.managed = true;
    connection.serverOnline = true;
    const ctx = makeCtx();

    for (let i = 0; i < OFFLINE_SUSPECT_SKIP_THRESHOLD + 10; i += 1) {
      skipFailedTrack(ctx, 7, `failure ${i}`);
    }

    expect(ctx.advanceTrack).toHaveBeenCalledTimes(
      OFFLINE_SUSPECT_SKIP_THRESHOLD - 1,
    );
    expect(ctx.suspendForReconnect).toHaveBeenCalled();
    expect(ctx.queueIndex).toBe(0);
  });

  it("keeps the error-stop limit for corrupt tracks while online", () => {
    connection.managed = true;
    connection.serverOnline = true;
    const ctx = makeCtx({
      advanceTrack: vi.fn(() => false),
    });

    skipFailedTrack(ctx, 7, "decode failed");

    expect(ctx.failedTrackSkips).toBe(0);
    expect(ctx.stopAtQueueEnd).toHaveBeenCalledTimes(1);
    expect(ctx.suspendForReconnect).not.toHaveBeenCalled();
  });
});

function makeQueueCtx(
  overrides: Record<string, unknown> = {},
): MusicTrackBoundaryContext & { appended: SubsonicSong[][] } {
  const appended: SubsonicSong[][] = [];
  const ctx = {
    continuousRefillInFlight: null as Promise<boolean> | null,
    queue: [track("a"), track("b")],
    queueIndex: 0,
    shuffle: false,
    autoplay: true,
    continuousMode: "off",
    shuffleUpcoming: [] as number[],
    queueSettings: { maxQueueSize: 500 },
    library: {
      getRandomSongs: vi.fn(async () => [] as SubsonicSong[]),
    },
    appendTracksToQueue: vi.fn((tracks: SubsonicSong[]) => {
      appended.push(tracks);
      return true;
    }),
    refillLibraryQueue: vi.fn(async () => false),
    refillPersonalQueue: vi.fn(async () => false),
    appendRandomSongsToQueue: vi.fn(async () => false),
    maybeRefillContinuousQueue: vi.fn(async () => false),
    persistPlaybackState: vi.fn(),
    appended,
    ...overrides,
  } as unknown as MusicTrackBoundaryContext & { appended: SubsonicSong[][] };
  return ctx;
}

describe("appendRandomSongsToQueue", () => {
  it("retries when a draw comes back entirely as duplicates", async () => {
    const getRandomSongs = vi
      .fn()
      .mockResolvedValueOnce([track("a"), track("b")])
      .mockResolvedValueOnce([track("a"), track("c")])
      .mockResolvedValue([]);
    const ctx = makeQueueCtx({ library: { getRandomSongs } });

    const ok = await appendRandomSongsToQueue(ctx, 8);

    expect(ok).toBe(true);
    expect(getRandomSongs).toHaveBeenCalledTimes(3);
    expect(ctx.appended).toHaveLength(1);
    expect(ctx.appended[0].map((t) => t.id)).toEqual(["c"]);
  });

  it("gives up after repeated duplicate draws", async () => {
    const getRandomSongs = vi.fn(async () => [track("a"), track("b")]);
    const ctx = makeQueueCtx({ library: { getRandomSongs } });

    const ok = await appendRandomSongsToQueue(ctx, 8);

    expect(ok).toBe(false);
    expect(getRandomSongs).toHaveBeenCalledTimes(3);
    expect(ctx.appended).toHaveLength(0);
  });

  it("stops drawing once the batch is full of uniques", async () => {
    const getRandomSongs = vi
      .fn()
      .mockResolvedValueOnce([track("c"), track("d")]);
    const ctx = makeQueueCtx({ library: { getRandomSongs } });

    const ok = await appendRandomSongsToQueue(ctx, 2);

    expect(ok).toBe(true);
    expect(getRandomSongs).toHaveBeenCalledTimes(1);
    expect(ctx.appended[0].map((t) => t.id)).toEqual(["c", "d"]);
  });
});

describe("maybeRefillContinuousQueue", () => {
  it("refills against upcoming tracks, not total queue length", async () => {
    const getRandomSongs = vi.fn(async (count: number) =>
      Array.from({ length: count }, (_, i) => track(`new-${i}`)),
    );
    const ctx = makeQueueCtx({
      continuousMode: "random",
      queue: Array.from({ length: 25 }, (_, i) => track(`q-${i}`)),
      queueIndex: 20,
      shuffle: true,
      shuffleUpcoming: [21, 22, 23, 24],
      library: { getRandomSongs },
    });
    ctx.appendRandomSongsToQueue = (count: number) =>
      appendRandomSongsToQueue(ctx, count);

    const ok = await maybeRefillContinuousQueue(ctx);

    expect(ok).toBe(true);
    // Soft target is 25 upcoming and only 4 remain, so it draws 21.
    expect(getRandomSongs).toHaveBeenCalledWith(21);
  });
});

describe("onTrackEnded", () => {
  it("advances without blocking when upcoming tracks exist", async () => {
    let resolveRefill: (value: boolean) => void = () => {};
    const refill = new Promise<boolean>((resolve) => {
      resolveRefill = resolve;
    });
    const maybeRefill = vi.fn(() => refill);
    const ctx = makeCtx({
      continuousMode: "random",
      autoplay: true,
      repeat: "off",
      shuffle: false,
      queue: [track("a"), track("b"), track("c")],
      queueIndex: 0,
      currentTrack: track("a"),
      maybeRefillContinuousQueue: maybeRefill,
      recordPlayCompletion: vi.fn(async () => {}),
    });

    await onTrackEnded(ctx);

    expect(maybeRefill).toHaveBeenCalledTimes(1);
    expect(ctx.advanceTrack).toHaveBeenCalledTimes(1);
    resolveRefill(true);
  });

  it("awaits the refill when the queue is dry and stops on failure", async () => {
    const maybeRefill = vi.fn(async () => false);
    const ctx = makeCtx({
      continuousMode: "random",
      autoplay: true,
      repeat: "off",
      shuffle: false,
      queue: [track("a")],
      queueIndex: 0,
      currentTrack: track("a"),
      maybeRefillContinuousQueue: maybeRefill,
      recordPlayCompletion: vi.fn(async () => {}),
    });

    await onTrackEnded(ctx);

    expect(maybeRefill).toHaveBeenCalledTimes(1);
    expect(ctx.stopAtQueueEnd).toHaveBeenCalledTimes(1);
    expect(ctx.advanceTrack).not.toHaveBeenCalled();
  });
});
