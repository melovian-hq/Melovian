// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import { buildMixContextInput, buildSingleMix } from "./mix-generator";
import { defaultMixSettings, mergeMixSettings } from "./mix-settings";
import type { SubsonicSong } from "$lib/subsonic/types";

const MIX_IDS = [
  "on-repeat",
  "replay",
  "for-you-50",
  "for-you-100",
  "daily-mix-1",
  "daily-mix-2",
  "daily-mix-3",
  "genre-mix-1",
  "genre-mix-2",
  "genre-mix-3",
  "discover",
  "deep-cuts",
  "release-radar",
  "throwback",
  "year-rewind",
  "popular",
  "your-artists",
  "overlooked",
  "favorites-mix",
  "decade-mix-1",
  "decade-mix-2",
  "decade-mix-3",
  "fresh-finds",
] as const;

function song(id: string, artist = "A"): SubsonicSong {
  return { id, title: id, artist, albumId: `al-${id}`, duration: 180 };
}

describe("buildSingleMix exploratory", () => {
  it("unknown mix ids never throw and return null", async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.string({ minLength: 1, maxLength: 24 }),
        async (id) => {
          if ((MIX_IDS as readonly string[]).includes(id)) return;
          const ctx = buildMixContextInput(
            null,
            [],
            [],
            "seed",
            mergeMixSettings(defaultMixSettings()),
            (entry) => song(entry.trackId, entry.artistName),
          );
          const mix = await buildSingleMix(id, ctx, {
            searchArtistSongs: async () => [],
            searchArtistAlbums: async () => [],
            getAlbumSongs: async () => [],
            getSimilarSongs: async () => [],
            getRandomSongs: async () => [],
            getRandomAlbums: async () => [],
            getNewestAlbums: async () => [],
            getGenreSongs: async () => [],
            getGenres: async () => [],
            getStarredSongs: async () => [],
          });
          expect(mix).toBeNull();
        },
      ),
      { numRuns: 40 },
    );
  });

  it("known mix ids return matching id or null without throwing", async () => {
    const pool = Array.from({ length: 40 }, (_, i) =>
      song(`t${i}`, `artist-${i % 7}`),
    );
    const fetchers = {
      searchArtistSongs: async () => pool,
      searchArtistAlbums: async () =>
        pool.slice(0, 5).map((track) => ({
          id: track.albumId!,
          name: track.album ?? track.id,
          coverArt: track.coverArt,
          genre: "Rock",
        })),
      getAlbumSongs: async () => pool,
      getSimilarSongs: async () => pool,
      getRandomSongs: async () => pool,
      getRandomAlbums: async () =>
        pool.slice(0, 4).map((track) => ({
          id: track.albumId!,
          name: track.title,
          coverArt: track.coverArt,
        })),
      getNewestAlbums: async () =>
        pool.slice(0, 6).map((track) => ({
          id: track.albumId!,
          name: track.title,
          coverArt: track.coverArt,
          year: 2024,
        })),
      getGenreSongs: async () => pool,
      getGenres: async () => [{ name: "Rock", songCount: 40 }],
      getStarredSongs: async () => pool.slice(0, 10),
    };

    for (const id of MIX_IDS) {
      const history = pool.slice(0, 20).map((track, index) => ({
        trackId: track.id,
        trackTitle: track.title,
        artistName: track.artist ?? "A",
        albumId: track.albumId ?? "",
        albumTitle: track.album ?? "",
        positionMs: 120_000,
        durationMs: 180_000,
        played: true,
        playCount: 3 + (index % 5),
        listenedMs: 0,
        lastPlayedAt: new Date(Date.now() - index * 86_400_000).toISOString(),
        coverArtId: "",
      }));
      const ctx = buildMixContextInput(
        {
          totalPlays: 80,
          uniqueTracks: 20,
          totalListeningMs: 1_000_000,
          topArtists: [
            { key: "artist-0", label: "artist-0", count: 12 },
            { key: "artist-1", label: "artist-1", count: 8 },
          ],
          topTracks: pool.slice(0, 8).map((track, i) => ({
            key: track.id,
            label: track.title,
            count: 10 - i,
          })),
          topAlbums: [],
        },
        history,
        [],
        `day:${id}`,
        mergeMixSettings(defaultMixSettings()),
        (entry) => song(entry.trackId, entry.artistName),
      );
      const mix = await buildSingleMix(id, ctx, fetchers);
      if (mix) {
        expect(mix.id).toBe(id);
        expect(mix.tracks.length).toBeGreaterThan(0);
      }
    }
  });
});
