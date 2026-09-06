// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import {
  buildTasteProfile,
  completionRatio,
  listenAffinityWeight,
  scoreTrackTaste,
} from "./taste-score";
import type { ListenEntry, SubsonicSong } from "$lib/subsonic/types";

const entryArb: fc.Arbitrary<ListenEntry> = fc.record({
  trackId: fc
    .string({ minLength: 1, maxLength: 12 })
    .filter((id) => id.trim().length > 0),
  trackTitle: fc.string({ minLength: 1, maxLength: 16 }),
  artistName: fc.string({ minLength: 1, maxLength: 12 }),
  albumId: fc.string({ minLength: 1, maxLength: 8 }),
  albumTitle: fc.string({ minLength: 1, maxLength: 12 }),
  positionMs: fc.integer({ min: 0, max: 400_000 }),
  durationMs: fc.integer({ min: 1_000, max: 400_000 }),
  played: fc.boolean(),
  playCount: fc.integer({ min: 0, max: 40 }),
  listenedMs: fc.integer({ min: 0, max: 2_000_000 }),
  lastPlayedAt: fc.constant(new Date().toISOString()),
  coverArtId: fc.constant(""),
});

describe("taste-score listen property", () => {
  it("affinity weight is always positive finite", () => {
    fc.assert(
      fc.property(entryArb, (entry) => {
        const weight = listenAffinityWeight(entry);
        expect(Number.isFinite(weight)).toBe(true);
        expect(weight).toBeGreaterThan(0);
      }),
      { numRuns: 150 },
    );
  });

  it("completion ratio stays within a sane bound", () => {
    fc.assert(
      fc.property(entryArb, (entry) => {
        const ratio = completionRatio(entry);
        expect(Number.isFinite(ratio)).toBe(true);
        expect(ratio).toBeGreaterThanOrEqual(0);
        expect(ratio).toBeLessThanOrEqual(2);
      }),
      { numRuns: 150 },
    );
  });

  it("listenedMs map always influences score when track matches", () => {
    fc.assert(
      fc.property(
        entryArb,
        fc.integer({ min: 1_000, max: 900_000 }),
        (base, listenedMs) => {
          const entry = {
            ...base,
            listenedMs,
            playCount: Math.max(1, base.playCount),
          };
          const profile = buildTasteProfile(
            null,
            [entry],
            new Map([[entry.trackId, entry.playCount]]),
            new Set(),
            [],
            new Map([[entry.trackId, listenedMs]]),
          );
          const track: SubsonicSong = {
            id: entry.trackId,
            title: entry.trackTitle,
            artist: entry.artistName,
            duration: Math.floor(entry.durationMs / 1000),
          };
          const score = scoreTrackTaste(track, profile);
          expect(Number.isFinite(score)).toBe(true);
        },
      ),
      { numRuns: 100 },
    );
  });

  it("more listen time never decreases affinity weight for same playCount", () => {
    fc.assert(
      fc.property(
        entryArb,
        fc.integer({ min: 0, max: 100_000 }),
        fc.integer({ min: 1, max: 500_000 }),
        (base, low, bump) => {
          const a = { ...base, listenedMs: low, playCount: 3 };
          const b = { ...base, listenedMs: low + bump, playCount: 3 };
          expect(listenAffinityWeight(b)).toBeGreaterThanOrEqual(
            listenAffinityWeight(a),
          );
        },
      ),
      { numRuns: 120 },
    );
  });
});
