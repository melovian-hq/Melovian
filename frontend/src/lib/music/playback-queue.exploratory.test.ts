// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import {
  canAdvanceQueue,
  nextSequentialIndex,
  popShuffleHistory,
  shouldStopAtQueueEnd,
  shuffleIndices,
  shuffleNewIndices,
  type RepeatMode,
} from "./playback-queue";

const repeatArb = fc.constantFrom<RepeatMode>("off", "all", "one");

describe("playback-queue exploratory", () => {
  it("explores sequential index bounds across random queues", () => {
    fc.assert(
      fc.property(
        fc.integer({ min: -2, max: 40 }),
        fc.integer({ min: 0, max: 40 }),
        repeatArb,
        (queueIndex, queueLength, repeat) => {
          const next = nextSequentialIndex(queueIndex, queueLength, repeat);
          if (next === null) return;
          expect(next).toBeGreaterThanOrEqual(0);
          expect(next).toBeLessThan(queueLength);
        },
      ),
      { numRuns: 200 },
    );
  });

  it("explores shuffle index uniqueness under random lengths", () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 0, max: 30 }),
        fc.integer({ min: -1, max: 30 }),
        (length, exclude) => {
          const order = shuffleIndices(length, exclude);
          expect(new Set(order).size).toBe(order.length);
          for (const index of order) {
            expect(index).toBeGreaterThanOrEqual(0);
            expect(index).toBeLessThan(length);
            expect(index).not.toBe(exclude);
          }
        },
      ),
      { numRuns: 150 },
    );
  });

  it("explores stop-at-end and advance combinations", () => {
    fc.assert(
      fc.property(
        fc.boolean(),
        repeatArb,
        fc.boolean(),
        fc.integer({ min: -1, max: 20 }),
        fc.integer({ min: 0, max: 20 }),
        fc.boolean(),
        (atEnd, repeat, autoplay, queueIndex, queueLength, shuffle) => {
          const stop = shouldStopAtQueueEnd(atEnd, repeat, autoplay);
          if (!atEnd) expect(stop).toBe(false);
          if (repeat === "all" && atEnd) expect(stop).toBe(false);
          if (autoplay && atEnd && repeat !== "all") expect(stop).toBe(false);

          const canAdvance = canAdvanceQueue(
            queueIndex,
            queueLength,
            repeat,
            shuffle,
          );
          if (queueLength === 0) expect(canAdvance).toBe(false);
        },
      ),
      { numRuns: 200 },
    );
  });

  it("explores shuffle history pops never invent indices", () => {
    fc.assert(
      fc.property(
        fc.array(fc.nat({ max: 50 }), { maxLength: 20 }),
        fc.integer({ min: -2, max: 50 }),
        (history, current) => {
          const step = popShuffleHistory(history, current);
          if (!step) {
            expect(history.length === 0 || current < 0).toBe(true);
            return;
          }
          expect(step.previousIndex).toBe(history[history.length - 1]);
          expect(step.history).toEqual(history.slice(0, -1));
          expect(step.prependUpcoming).toBe(current);
        },
      ),
      { numRuns: 150 },
    );
  });

  it("explores shuffleNewIndices range membership", () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 0, max: 20 }),
        fc.integer({ min: 0, max: 30 }),
        fc.integer({ min: -1, max: 30 }),
        (from, toRaw, exclude) => {
          const to = Math.max(from, toRaw);
          const added = shuffleNewIndices(from, to, exclude);
          expect(new Set(added).size).toBe(added.length);
          for (const index of added) {
            expect(index).toBeGreaterThanOrEqual(from);
            expect(index).toBeLessThan(to);
            expect(index).not.toBe(exclude);
          }
        },
      ),
      { numRuns: 150 },
    );
  });
});
