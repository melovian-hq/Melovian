// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import { computeStatsFromHistory } from "./stats-utils";
import type { ListenEntry } from "$lib/subsonic/types";

const entryArb: fc.Arbitrary<ListenEntry> = fc.record({
  trackId: fc
    .string({ minLength: 1, maxLength: 10 })
    .filter((id) => id.trim().length > 0),
  trackTitle: fc.string({ minLength: 1, maxLength: 12 }),
  artistName: fc.string({ minLength: 1, maxLength: 12 }),
  albumId: fc.string({ minLength: 1, maxLength: 8 }),
  albumTitle: fc.string({ minLength: 1, maxLength: 12 }),
  positionMs: fc.integer({ min: 0, max: 200_000 }),
  durationMs: fc.integer({ min: 1_000, max: 300_000 }),
  played: fc.boolean(),
  playCount: fc.integer({ min: 1, max: 20 }),
  listenedMs: fc.integer({ min: 0, max: 600_000 }),
  lastPlayedAt: fc.constant(new Date().toISOString()),
  coverArtId: fc.constant(""),
});

describe("stats-utils listen property", () => {
  it("prefers listenedMs when present for totalListeningMs", () => {
    fc.assert(
      fc.property(
        fc.array(entryArb, { minLength: 1, maxLength: 20 }),
        (history) => {
          const stats = computeStatsFromHistory(history);
          const expected = history.reduce((sum, e) => {
            if ((e.listenedMs ?? 0) > 0) return sum + e.listenedMs;
            return sum + e.durationMs * e.playCount;
          }, 0);
          expect(stats.totalListeningMs).toBe(expected);
          expect(stats.totalPlays).toBe(
            history.reduce((sum, e) => sum + e.playCount, 0),
          );
        },
      ),
      { numRuns: 80 },
    );
  });
});
