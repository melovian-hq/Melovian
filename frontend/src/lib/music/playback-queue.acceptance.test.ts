// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  canAdvanceQueue,
  nextSequentialIndex,
  popShuffleHistory,
  shouldStopAtQueueEnd,
  shuffleIndices,
} from "./playback-queue";

describe("playback-queue acceptance", () => {
  it("accepts a full listen-through with repeat off", () => {
    const length = 3;
    let index = 0;
    const visited: number[] = [index];

    while (true) {
      const atEnd = index >= length - 1;
      if (shouldStopAtQueueEnd(atEnd, "off", false)) break;
      if (!canAdvanceQueue(index, length, "off", false)) break;
      const next = nextSequentialIndex(index, length, "off");
      expect(next).not.toBeNull();
      index = next!;
      visited.push(index);
    }

    expect(visited).toEqual([0, 1, 2]);
    expect(canAdvanceQueue(2, length, "off", false)).toBe(false);
  });

  it("accepts repeat-all continuous cycling", () => {
    const index = 2;
    const next = nextSequentialIndex(index, 3, "all");
    expect(next).toBe(0);
    expect(canAdvanceQueue(2, 3, "all", false)).toBe(true);
    expect(shouldStopAtQueueEnd(true, "all", false)).toBe(false);
  });

  it("accepts shuffle previous restores history order", () => {
    const history = [0, 2, 1];
    const step = popShuffleHistory(history, 4);
    expect(step).toEqual({
      previousIndex: 1,
      history: [0, 2],
      prependUpcoming: 4,
    });
    const again = popShuffleHistory(step!.history, step!.previousIndex);
    expect(again?.previousIndex).toBe(2);
  });

  it("accepts shuffle mode can advance even at sequential end", () => {
    expect(canAdvanceQueue(2, 3, "off", true)).toBe(true);
    const order = shuffleIndices(3, 2);
    expect(order).toHaveLength(2);
    expect(order).not.toContain(2);
  });
});
