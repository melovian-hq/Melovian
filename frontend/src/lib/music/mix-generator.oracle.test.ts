// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import {
  buildMixContextInput,
  createSeededRandom,
  dedupeTracks,
  emptyMixSeed,
  orderForFlow,
  selectMixTracks,
  shuffleWithSeed,
} from "./mix-generator";
import { defaultMixPolicy } from "./mix-select";
import { defaultMixSettings } from "./mix-settings";
import type { SubsonicSong } from "$lib/subsonic/types";

const song = (id: string, artist: string, duration?: number): SubsonicSong => ({
  id,
  title: `Track ${id}`,
  artist,
  albumId: `album-${artist}`,
  duration,
});

describe("mix-generator oracle", () => {
  it("same seed yields identical random streams", () => {
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 32 }), (seed) => {
        const a = createSeededRandom(seed);
        const b = createSeededRandom(seed);
        for (let i = 0; i < 8; i++) {
          expect(a()).toBe(b());
        }
      }),
    );
  });

  it("seeded random values stay in [0, 1)", () => {
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 24 }), (seed) => {
        const rand = createSeededRandom(seed);
        for (let i = 0; i < 20; i++) {
          const value = rand();
          expect(value).toBeGreaterThanOrEqual(0);
          expect(value).toBeLessThan(1);
        }
      }),
    );
  });

  it("orderForFlow never drops or invents tracks", () => {
    const tracks = [
      song("1", "A", 120),
      song("2", "B", 300),
      song("3", "C", 140),
      song("4", "A", 280),
      song("5", "B", 100),
    ];
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 20 }), (seed) => {
        const ordered = orderForFlow(tracks, seed, { durationPacing: true });
        expect(ordered.map((t) => t.id).sort()).toEqual(
          tracks.map((t) => t.id).sort(),
        );
      }),
    );
  });

  it("dedupeTracks is idempotent", () => {
    const tracks = [
      song("1", "A"),
      song("1", "A"),
      song("2", "B"),
      song("2", "C"),
    ];
    const once = dedupeTracks(tracks);
    expect(dedupeTracks(once)).toEqual(once);
  });

  it("shuffleWithSeed is a permutation of the input", () => {
    fc.assert(
      fc.property(
        fc.array(fc.integer(), { maxLength: 15 }),
        fc.string({ maxLength: 16 }),
        (items, seed) => {
          const shuffled = shuffleWithSeed(items, seed);
          expect(shuffled).toHaveLength(items.length);
          expect([...shuffled].sort((a, b) => a - b)).toEqual(
            [...items].sort((a, b) => a - b),
          );
        },
      ),
    );
  });

  it("selectMixTracks never duplicates ids and stays within the pool", () => {
    const tracks = [
      song("1", "A", 120),
      song("2", "B", 300),
      song("3", "C", 140),
      song("1", "A", 120),
      song("4", "A", 280),
    ];
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 20 }), (seed) => {
        const ctx = buildMixContextInput(
          null,
          [],
          [],
          seed,
          defaultMixSettings(),
          () => song("x", "X"),
        );
        const picked = selectMixTracks(
          tracks,
          {
            playCountByTrack: ctx.playCountByTrack,
            listenedMsByTrack: ctx.listenedMsByTrack,
            skippedTrackIds: ctx.skippedTrackIds,
            recentlyPlayedIds: ctx.recentlyPlayedIds,
            recentMixTrackIds: ctx.recentMixTrackIds,
            settings: ctx.settings,
            tasteProfile: ctx.tasteProfile,
            starredTrackIds: new Set(),
          },
          emptyMixSeed(),
          { ...defaultMixPolicy(4), targetCount: 4 },
          seed,
        );
        const ids = picked.map((track) => track.id);
        expect(new Set(ids).size).toBe(ids.length);
        expect(ids.every((id) => ["1", "2", "3", "4"].includes(id))).toBe(true);
      }),
    );
  });
});
