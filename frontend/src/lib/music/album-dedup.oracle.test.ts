// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import {
  albumIdentityKey,
  albumQualityScore,
  dedupeAlbums,
} from "./album-dedup";
import type { SubsonicAlbum } from "$lib/subsonic/types";

const albumArb: fc.Arbitrary<SubsonicAlbum> = fc.record({
  id: fc.string({ minLength: 1, maxLength: 12 }),
  name: fc.string({ maxLength: 40 }),
  artist: fc.option(fc.string({ maxLength: 24 }), { nil: undefined }),
  songCount: fc.option(fc.nat({ max: 40 }), { nil: undefined }),
  duration: fc.option(fc.nat({ max: 10_000 }), { nil: undefined }),
});

describe("album-dedup oracle", () => {
  it("prefers higher quality score for identical identity keys", () => {
    const low: SubsonicAlbum = {
      id: "mp3",
      name: "Abbey Road (MP3)",
      artist: "The Beatles",
      songCount: 10,
      duration: 1000,
    };
    const high: SubsonicAlbum = {
      id: "flac",
      name: "Abbey Road (FLAC)",
      artist: "The Beatles",
      songCount: 17,
      duration: 2800,
    };
    expect(albumIdentityKey(low)).toBe(albumIdentityKey(high));
    expect(albumQualityScore(high)).toBeGreaterThan(albumQualityScore(low));
    expect(dedupeAlbums([low, high])[0]?.id).toBe("flac");
  });

  it("dedupe is idempotent", () => {
    fc.assert(
      fc.property(fc.array(albumArb, { maxLength: 20 }), (albums) => {
        const once = dedupeAlbums(albums);
        expect(dedupeAlbums(once)).toEqual(once);
      }),
    );
  });
});
