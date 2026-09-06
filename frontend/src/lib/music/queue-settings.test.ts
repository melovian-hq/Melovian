// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  enforceQueueLimit,
  mergeQueueSettings,
  remainingQueueSlots,
  truncateTrackList,
} from "./queue-settings";

describe("queue-settings", () => {
  it("merges and clamps max queue size", () => {
    expect(mergeQueueSettings({ maxQueueSize: 9999 }).maxQueueSize).toBe(5000);
    expect(mergeQueueSettings({ maxQueueSize: -5 }).maxQueueSize).toBe(0);
    expect(mergeQueueSettings({ maxQueueSize: 250 }).maxQueueSize).toBe(250);
  });

  it("truncates incoming track lists", () => {
    const tracks = Array.from({ length: 10 }, (_, i) => i);
    expect(truncateTrackList(tracks, 0)).toHaveLength(10);
    expect(truncateTrackList(tracks, 4)).toEqual([0, 1, 2, 3]);
  });

  it("computes remaining slots", () => {
    expect(remainingQueueSlots(8, 0)).toBe(Number.POSITIVE_INFINITY);
    expect(remainingQueueSlots(8, 10)).toBe(2);
    expect(remainingQueueSlots(10, 10)).toBe(0);
  });

  it("trims queue around the current index", () => {
    const queue = ["a", "b", "c", "d", "e"];
    const { queue: trimmed, queueIndex } = enforceQueueLimit(queue, 2, 3);
    expect(trimmed).toEqual(["a", "b", "c"]);
    expect(queueIndex).toBe(2);
  });
});
