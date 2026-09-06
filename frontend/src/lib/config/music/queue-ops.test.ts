// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";
import { moveInQueue, type MusicQueueContext } from "./queue-ops";
import type { QueueTrack } from "$lib/subsonic";

function track(id: string): QueueTrack {
  return { id, title: id, artist: "a", album: "b", duration: 1 } as QueueTrack;
}

function makeCtx(
  overrides: Partial<MusicQueueContext> = {},
): MusicQueueContext {
  const seedShuffleUpcoming = vi.fn();
  return {
    queue: [track("a"), track("b"), track("c")],
    queueIndex: 0,
    shuffle: true,
    shuffleUpcoming: [2, 1],
    shuffleHistory: [0],
    queueSettings: { maxQueueSize: 500 },
    engine: null,
    playing: false,
    continuousMode: "off",
    queueOpen: false,
    currentTrack: track("a"),
    playerLayout: "mini",
    pendingStartAt: null,
    pendingStartPaused: false,
    playTracks: vi.fn(),
    prefetchAround: vi.fn(),
    persistPlaybackState: vi.fn(),
    requestPlayCurrent: vi.fn(),
    seedShuffleUpcoming,
    stopSmoothProgress: vi.fn(),
    ...overrides,
  };
}

describe("moveInQueue", () => {
  it("reseeds shuffle after reorder", () => {
    const ctx = makeCtx();
    moveInQueue(ctx, 1, 2);
    expect(ctx.queue.map((t) => t.id)).toEqual(["a", "c", "b"]);
    expect(ctx.shuffleHistory).toEqual([]);
    expect(ctx.seedShuffleUpcoming).toHaveBeenCalledTimes(1);
  });

  it("does not touch shuffle bags when shuffle is off", () => {
    const ctx = makeCtx({
      shuffle: false,
      shuffleUpcoming: [2],
      shuffleHistory: [0],
    });
    moveInQueue(ctx, 1, 2);
    expect(ctx.shuffleHistory).toEqual([0]);
    expect(ctx.shuffleUpcoming).toEqual([2]);
    expect(ctx.seedShuffleUpcoming).not.toHaveBeenCalled();
  });
});
