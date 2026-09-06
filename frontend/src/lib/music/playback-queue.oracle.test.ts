// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import {
  canAdvanceQueue,
  nextSequentialIndex,
  playNextShuffleUpcoming,
  queueIndexAfterRemove,
  shouldStopAtQueueEnd,
  shuffleIndices,
  type RepeatMode,
} from "./playback-queue";

const repeatArb = fc.constantFrom<RepeatMode>("off", "all", "one");

describe("playback-queue oracle", () => {
  it("repeat-all never returns null for a non-empty in-bounds queue", () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 0, max: 20 }),
        fc.integer({ min: 1, max: 21 }),
        (index, length) => {
          const queueIndex = Math.min(index, length - 1);
          expect(nextSequentialIndex(queueIndex, length, "all")).not.toBeNull();
        },
      ),
    );
  });

  it("stop-at-end implies advance is blocked without shuffle", () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 0, max: 15 }),
        repeatArb,
        fc.boolean(),
        (length, repeat, autoplay) => {
          if (length === 0) return;
          const atEnd = true;
          const stop = shouldStopAtQueueEnd(atEnd, repeat, autoplay);
          const queueIndex = length - 1;
          const canAdvance = canAdvanceQueue(queueIndex, length, repeat, false);
          if (stop) {
            expect(canAdvance).toBe(false);
          }
        },
      ),
    );
  });

  it("shuffleIndices permutes all eligible indices", () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 1, max: 25 }),
        fc.integer({ min: -1, max: 24 }),
        (length, excludeRaw) => {
          const exclude = excludeRaw >= length ? -1 : excludeRaw;
          const order = shuffleIndices(length, exclude);
          const expected = Array.from({ length }, (_, i) => i).filter(
            (i) => i !== exclude,
          );
          expect([...order].sort((a, b) => a - b)).toEqual(expected);
        },
      ),
    );
  });

  it("play next keeps inserted indices at the front of shuffle upcoming", () => {
    fc.assert(
      fc.property(
        fc.array(fc.integer({ min: 0, max: 20 }), {
          minLength: 0,
          maxLength: 12,
        }),
        fc.integer({ min: 0, max: 8 }),
        fc.integer({ min: 1, max: 4 }),
        (upcoming, insertAt, count) => {
          const next = playNextShuffleUpcoming(upcoming, insertAt, count);
          const inserted = Array.from(
            { length: count },
            (_, i) => insertAt + i,
          );
          expect(next.slice(0, count)).toEqual(inserted);
          const shifted = upcoming.map((i) => (i >= insertAt ? i + count : i));
          expect(next.slice(count)).toEqual(shifted);
        },
      ),
    );
  });

  it("removing the current track restarts at the slid-in index", () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 1, max: 12 }),
        fc.integer({ min: 0, max: 11 }),
        (length, removeRaw) => {
          const removedIndex = Math.min(removeRaw, length - 1);
          const queueIndex = removedIndex;
          const newLength = length - 1;
          const got = queueIndexAfterRemove(
            queueIndex,
            removedIndex,
            newLength,
          );
          if (newLength === 0) {
            expect(got).toEqual({ queueIndex: -1, restart: false });
            return;
          }
          expect(got.restart).toBe(true);
          expect(got.queueIndex).toBeGreaterThanOrEqual(0);
          expect(got.queueIndex).toBeLessThan(newLength);
        },
      ),
    );
  });
});
