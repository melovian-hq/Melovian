// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { QueueTrack } from "$lib/subsonic";
import { connection } from "$lib/music/connection.svelte";
import { OFFLINE_SUSPECT_SKIP_THRESHOLD } from "$lib/music/offline-gate";
import {
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
