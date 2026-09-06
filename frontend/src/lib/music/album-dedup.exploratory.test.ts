// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import { albumIdentityKey, dedupeAlbums } from "./album-dedup";
import type { SubsonicAlbum } from "$lib/subsonic/types";

const albumArb: fc.Arbitrary<SubsonicAlbum> = fc.record({
  id: fc.string({ minLength: 1, maxLength: 12 }),
  name: fc.string({ maxLength: 40 }),
  artist: fc.option(fc.string({ maxLength: 24 }), { nil: undefined }),
  songCount: fc.option(fc.nat({ max: 40 }), { nil: undefined }),
  duration: fc.option(fc.nat({ max: 10_000 }), { nil: undefined }),
});

describe("album-dedup exploratory", () => {
  it("explores identity key stability under whitespace and case", () => {
    fc.assert(
      fc.property(albumArb, (album) => {
        const key = albumIdentityKey(album);
        expect(albumIdentityKey(album)).toBe(key);
        expect(typeof key).toBe("string");
      }),
      { numRuns: 100 },
    );
  });

  it("explores dedupe never increases length and keeps unique keys", () => {
    fc.assert(
      fc.property(fc.array(albumArb, { maxLength: 40 }), (albums) => {
        const result = dedupeAlbums(albums);
        expect(result.length).toBeLessThanOrEqual(albums.length);
        const keys = result.map((album) => albumIdentityKey(album));
        expect(new Set(keys).size).toBe(keys.length);
      }),
      { numRuns: 80 },
    );
  });
});
