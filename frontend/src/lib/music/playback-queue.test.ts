// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  canAdvanceQueue,
  nextSequentialIndex,
  popShuffleHistory,
  shouldStopAtQueueEnd,
  shuffleIndices,
  shuffleNewIndices,
} from "./playback-queue";

describe("nextSequentialIndex", () => {
  it("advances within the queue", () => {
    expect(nextSequentialIndex(0, 3, "off")).toBe(1);
    expect(nextSequentialIndex(1, 3, "off")).toBe(2);
  });

  it("stops at the end when repeat is off", () => {
    expect(nextSequentialIndex(2, 3, "off")).toBeNull();
  });

  it("wraps at the end when repeat is all", () => {
    expect(nextSequentialIndex(2, 3, "all")).toBe(0);
  });
});

describe("shouldStopAtQueueEnd", () => {
  it("stops at the end with repeat off", () => {
    expect(shouldStopAtQueueEnd(true, "off", false)).toBe(true);
  });

  it("continues when autoplay extended the queue", () => {
    expect(shouldStopAtQueueEnd(true, "off", true)).toBe(false);
  });

  it("continues when repeat all wraps", () => {
    expect(shouldStopAtQueueEnd(true, "all", false)).toBe(false);
  });
});

describe("canAdvanceQueue", () => {
  it("blocks next at the end with repeat off", () => {
    expect(canAdvanceQueue(2, 3, "off", false)).toBe(false);
  });

  it("allows next at the end with repeat all", () => {
    expect(canAdvanceQueue(2, 3, "all", false)).toBe(true);
  });
});

describe("shuffleIndices", () => {
  it("includes every index except the current track", () => {
    const order = shuffleIndices(5, 2);
    expect(order).toHaveLength(4);
    expect(order).not.toContain(2);
    expect(new Set(order).size).toBe(4);
    for (const index of order) {
      expect(index).toBeGreaterThanOrEqual(0);
      expect(index).toBeLessThan(5);
    }
  });

  it("appends only new queue indices", () => {
    const added = shuffleNewIndices(5, 8, 2);
    expect(added).toHaveLength(3);
    expect(added.every((index) => index >= 5 && index < 8)).toBe(true);
    expect(added).not.toContain(2);
  });
});

describe("popShuffleHistory", () => {
  it("returns the last played index and shortens history", () => {
    const step = popShuffleHistory([1, 4, 7], 9);
    expect(step).toEqual({
      previousIndex: 7,
      history: [1, 4],
      prependUpcoming: 9,
    });
  });

  it("returns null when there is no shuffle history", () => {
    expect(popShuffleHistory([], 3)).toBeNull();
    expect(popShuffleHistory([2], -1)).toBeNull();
  });
});
