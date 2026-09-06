// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import {
  buildTasteProfile,
  scoreTrackTaste,
  weightedSampleTracks,
} from "./taste-score";
import type { SubsonicSong } from "$lib/subsonic/types";

const songArb = fc.record({
  id: fc
    .string({ minLength: 1, maxLength: 10 })
    .filter((id) => id.trim().length > 0),
  title: fc.string({ minLength: 1, maxLength: 16 }),
  artist: fc.string({ minLength: 1, maxLength: 12 }),
  album: fc.option(fc.string({ minLength: 1, maxLength: 12 }), {
    nil: undefined,
  }),
  genre: fc.option(fc.string({ minLength: 1, maxLength: 10 }), {
    nil: undefined,
  }),
}) as fc.Arbitrary<SubsonicSong>;

describe("taste-score property", () => {
  it("samples never exceed requested count or invent duplicates", () => {
    fc.assert(
      fc.property(
        fc.array(songArb, { minLength: 1, maxLength: 40 }),
        fc.integer({ min: 0, max: 30 }),
        fc.double({ min: 0, max: 0.999, noNaN: true }),
        (tracks, count, roll) => {
          const profile = buildTasteProfile(null, [], new Map(), new Set());
          const picks = weightedSampleTracks(
            tracks,
            profile,
            count,
            () => roll,
          );
          expect(picks.length).toBeLessThanOrEqual(count);
          expect(picks.length).toBeLessThanOrEqual(
            new Set(tracks.map((t) => t.id).filter(Boolean)).size,
          );
          expect(new Set(picks.map((t) => t.id)).size).toBe(picks.length);
        },
      ),
      { numRuns: 120 },
    );
  });

  it("skipped tracks always score below zero", () => {
    fc.assert(
      fc.property(songArb, (track) => {
        const profile = buildTasteProfile(
          null,
          [],
          new Map(),
          new Set([track.id]),
        );
        expect(scoreTrackTaste(track, profile)).toBeLessThan(0);
      }),
      { numRuns: 80 },
    );
  });
});
