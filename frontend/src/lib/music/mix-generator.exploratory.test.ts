// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import {
  dedupeTracks,
  interleaveByArtist,
  orderForFlow,
  shuffleWithSeed,
} from "./mix-generator";
import type { SubsonicSong } from "$lib/subsonic/types";

const songArb = fc.record({
  id: fc.string({ minLength: 1, maxLength: 12 }),
  title: fc.string({ maxLength: 24 }),
  artist: fc.string({ minLength: 1, maxLength: 16 }),
  albumId: fc.option(fc.string({ minLength: 1, maxLength: 12 }), {
    nil: undefined,
  }),
  duration: fc.option(fc.integer({ min: 30, max: 600 }), { nil: undefined }),
}) as fc.Arbitrary<SubsonicSong>;

describe("mix-generator exploratory", () => {
  it("explores dedupe never grows and preserves first id wins", () => {
    fc.assert(
      fc.property(fc.array(songArb, { maxLength: 40 }), (tracks) => {
        const deduped = dedupeTracks(tracks);
        expect(deduped.length).toBeLessThanOrEqual(tracks.length);
        const ids = deduped.map((t) => t.id);
        expect(new Set(ids).size).toBe(ids.length);
      }),
      { numRuns: 100 },
    );
  });

  it("explores seeded shuffle reproducibility and permutation", () => {
    fc.assert(
      fc.property(
        fc.array(fc.string({ maxLength: 8 }), { maxLength: 20 }),
        fc.string({ maxLength: 12 }),
        (items, seed) => {
          const a = shuffleWithSeed(items, seed);
          const b = shuffleWithSeed(items, seed);
          expect(a).toEqual(b);
          expect([...a].sort()).toEqual([...items].sort());
        },
      ),
      { numRuns: 80 },
    );
  });

  it("explores interleave and flow ordering length invariants", () => {
    fc.assert(
      fc.property(fc.array(songArb, { maxLength: 25 }), (tracks) => {
        const unique = dedupeTracks(tracks);
        expect(interleaveByArtist(unique)).toHaveLength(unique.length);
        expect(orderForFlow(unique, "explore-seed")).toHaveLength(
          unique.length,
        );
      }),
      { numRuns: 80 },
    );
  });
});
