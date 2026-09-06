// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from "vitest";
import fc from "fast-check";
import { SubsonicClient } from "./client";
import { getArtistInfo, search3 } from "./api";

function mockSubsonicResponse(body: Record<string, unknown>) {
  return {
    ok: true,
    status: 200,
    text: async () =>
      JSON.stringify({
        "subsonic-response": {
          status: "ok",
          ...body,
        },
      }),
  };
}

describe("subsonic api contract properties", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("getArtistInfo dedupes similar artists and never throws on arbitrary payloads", async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.array(
          fc.record({
            id: fc.option(fc.string(), { nil: undefined }),
            name: fc.string(),
          }),
          { maxLength: 30 },
        ),
        async (similarArtist) => {
          vi.stubGlobal(
            "fetch",
            vi.fn().mockResolvedValue(
              mockSubsonicResponse({
                artistInfo: {
                  biography: "Bio",
                  similarArtist,
                },
              }),
            ),
          );

          const client = new SubsonicClient();
          const info = await getArtistInfo(client, "artist-1");

          expect(Array.isArray(info.similarArtists)).toBe(true);
          const keys = new Set<string>();
          for (const artist of info.similarArtists) {
            expect(typeof artist.name).toBe("string");
            const key = artist.id ?? artist.name.toLowerCase();
            expect(keys.has(key)).toBe(false);
            keys.add(key);
          }
        },
      ),
      { numRuns: 25 },
    );
  });

  it("search3 accepts single-object and array result shapes", async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.boolean(),
        fc.array(
          fc.record({
            id: fc.string({ minLength: 1 }),
            name: fc.string({ minLength: 1 }),
          }),
          { maxLength: 5 },
        ),
        fc.array(
          fc.record({
            id: fc.string({ minLength: 1 }),
            name: fc.string({ minLength: 1 }),
          }),
          { maxLength: 5 },
        ),
        async (singleArtist, artists, albums) => {
          const artistPayload = singleArtist ? artists[0] : artists;
          vi.stubGlobal(
            "fetch",
            vi.fn().mockResolvedValue(
              mockSubsonicResponse({
                searchResult3: {
                  artist: artistPayload,
                  album: albums,
                  song: [],
                },
              }),
            ),
          );

          const client = new SubsonicClient();
          const result = await search3(client, "query");

          expect(result.artists.length).toBeLessThanOrEqual(
            artists.length || 1,
          );
          expect(result.albums.length).toBeLessThanOrEqual(albums.length);
          for (const artist of result.artists) {
            expect(artist.id.length).toBeGreaterThan(0);
            expect(artist.name.length).toBeGreaterThan(0);
          }
        },
      ),
      { numRuns: 25 },
    );
  });
});
