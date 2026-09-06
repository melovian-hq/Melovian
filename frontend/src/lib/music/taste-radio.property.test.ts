// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import {
  buildTasteProfile,
  scoreTrackTaste,
  weightedSampleTracks,
} from "./taste-score";
import {
  createPersonalRadioState,
  notePersonalComplete,
  notePersonalSkip,
  seedPersonalRadioTracks,
  wasTrackSkipped,
} from "./personal-radio";
import type { ListenEntry, SubsonicSong } from "$lib/subsonic/types";

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

const entryArb: fc.Arbitrary<ListenEntry> = fc.record({
  trackId: fc
    .string({ minLength: 1, maxLength: 10 })
    .filter((id) => id.trim().length > 0),
  trackTitle: fc.string({ minLength: 1, maxLength: 16 }),
  artistName: fc.string({ minLength: 1, maxLength: 12 }),
  albumId: fc.string({ minLength: 1, maxLength: 8 }),
  albumTitle: fc.string({ minLength: 1, maxLength: 12 }),
  positionMs: fc.integer({ min: 0, max: 300_000 }),
  durationMs: fc.integer({ min: 1_000, max: 400_000 }),
  played: fc.boolean(),
  playCount: fc.integer({ min: 1, max: 40 }),
  listenedMs: fc.integer({ min: 0, max: 800_000 }),
  lastPlayedAt: fc.constant(new Date().toISOString()),
  coverArtId: fc.constant(""),
});

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

describe("personal-radio property", () => {
  it("skip then complete clears the skip for that track", () => {
    fc.assert(
      fc.property(songArb, (track) => {
        const state = createPersonalRadioState();
        notePersonalSkip(state, track);
        expect(state.skipTrackIds.has(track.id)).toBe(true);
        notePersonalComplete(state, track);
        expect(state.skipTrackIds.has(track.id)).toBe(false);
        expect(state.lastCompletedId).toBe(track.id);
      }),
      { numRuns: 60 },
    );
  });

  it("wasTrackSkipped is consistent with the half-duration rule", () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 0, max: 600_000 }),
        fc.integer({ min: 0, max: 600_000 }),
        (positionMs, durationMs) => {
          const skipped = wasTrackSkipped(positionMs, durationMs);
          if (durationMs <= 0) {
            expect(skipped).toBe(positionMs < 30_000);
            return;
          }
          expect(skipped).toBe(positionMs / durationMs < 0.5);
        },
      ),
      { numRuns: 100 },
    );
  });

  it("seedPersonalRadioTracks never returns excluded or skipped ids", async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.array(songArb, { minLength: 5, maxLength: 20 }),
        fc.array(entryArb, { minLength: 3, maxLength: 12 }),
        fc.integer({ min: 1, max: 12 }),
        async (pool, history, count) => {
          const state = createPersonalRadioState();
          const exclude = new Set(pool.slice(0, 2).map((t) => t.id));
          for (const track of pool.slice(2, 4)) {
            notePersonalSkip(state, track);
          }
          const profile = buildTasteProfile(
            null,
            history,
            new Map(history.map((e) => [e.trackId, e.playCount])),
            new Set(),
          );
          const seeded = await seedPersonalRadioTracks(
            state,
            {
              getSimilarSongs: async () => pool,
              getRandomSongs: async () => pool,
              searchArtistSongs: async () => pool,
            },
            profile,
            history,
            null,
            count,
            exclude,
            (entry) => ({
              id: entry.trackId,
              title: entry.trackTitle,
              artist: entry.artistName,
              album: entry.albumTitle,
              albumId: entry.albumId,
            }),
          );
          for (const track of seeded) {
            expect(exclude.has(track.id)).toBe(false);
            expect(state.skipTrackIds.has(track.id)).toBe(false);
          }
          expect(seeded.length).toBeLessThanOrEqual(count);
          expect(new Set(seeded.map((t) => t.id)).size).toBe(seeded.length);
        },
      ),
      { numRuns: 40 },
    );
  });
});
