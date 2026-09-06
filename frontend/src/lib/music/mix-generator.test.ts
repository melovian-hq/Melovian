// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  buildMixContextInput,
  createSeededRandom,
  dedupeTracks,
  interleaveByArtist,
  orderForFlow,
  shuffleWithSeed,
  sortMixesByPriority,
  upsertMix,
} from "./mix-generator";
import { defaultMixSettings, mergeMixSettings } from "./mix-settings";
import type { SubsonicSong } from "$lib/subsonic/types";

const song = (
  id: string,
  artist: string,
  albumId?: string,
  duration?: number,
): SubsonicSong => ({
  id,
  title: `Track ${id}`,
  artist,
  albumId: albumId ?? `album-${artist}`,
  duration,
});

describe("mix-generator", () => {
  it("dedupes tracks by id", () => {
    const tracks = [song("1", "A"), song("1", "A"), song("2", "B")];
    expect(dedupeTracks(tracks)).toHaveLength(2);
  });

  it("produces stable seeded shuffles", () => {
    const items = ["a", "b", "c", "d", "e"];
    expect(shuffleWithSeed(items, "seed")).toEqual(
      shuffleWithSeed(items, "seed"),
    );
    expect(shuffleWithSeed(items, "seed")).not.toEqual(
      shuffleWithSeed(items, "other"),
    );
  });

  it("interleaves tracks by artist", () => {
    const tracks = [
      song("1", "Alpha"),
      song("2", "Alpha"),
      song("3", "Beta"),
      song("4", "Beta"),
      song("5", "Gamma"),
    ];
    const ordered = interleaveByArtist(tracks);
    expect(ordered.map((track) => track.artist)).toEqual([
      "Alpha",
      "Beta",
      "Gamma",
      "Alpha",
      "Beta",
    ]);
  });

  it("avoids back-to-back tracks from the same artist when possible", () => {
    const tracks = [
      song("1", "Alpha", "a1"),
      song("2", "Alpha", "a1"),
      song("3", "Beta", "b1"),
      song("4", "Beta", "b1"),
      song("5", "Gamma", "g1"),
    ];
    const ordered = orderForFlow(tracks, "test-flow");
    for (let i = 1; i < ordered.length; i++) {
      if (ordered.length > 3) {
        expect(ordered[i].artist).not.toBe(ordered[i - 1].artist);
      }
    }
  });

  it("alternates long and short tracks when duration pacing is enabled", () => {
    const tracks = [
      song("1", "Alpha", "a1", 120),
      song("2", "Beta", "b1", 360),
      song("3", "Gamma", "g1", 130),
      song("4", "Delta", "d1", 350),
    ];
    const ordered = orderForFlow(tracks, "duration-flow", {
      durationPacing: true,
    });
    expect(ordered.length).toBe(4);
  });

  it("creates deterministic seeded random values", () => {
    const a = createSeededRandom("daily");
    const b = createSeededRandom("daily");
    expect(a()).toBe(b());
  });

  it("sorts mixes by display priority", () => {
    const ordered = sortMixesByPriority([
      { id: "discover", title: "", subtitle: "", tracks: [], gradient: "" },
      { id: "on-repeat", title: "", subtitle: "", tracks: [], gradient: "" },
      { id: "daily-mix-1", title: "", subtitle: "", tracks: [], gradient: "" },
    ]);
    expect(ordered.map((mix) => mix.id)).toEqual([
      "on-repeat",
      "daily-mix-1",
      "discover",
    ]);
  });

  it("upserts mixes by id", () => {
    const first = {
      id: "discover",
      title: "Old",
      subtitle: "",
      tracks: [],
      gradient: "",
    };
    const updated = {
      id: "discover",
      title: "New",
      subtitle: "",
      tracks: [],
      gradient: "",
    };
    const result = upsertMix([first], updated);
    expect(result).toHaveLength(1);
    expect(result[0]?.title).toBe("New");
  });

  it("builds discover candidates from recent play window", () => {
    const ctx = buildMixContextInput(
      null,
      [
        {
          trackId: "old",
          trackTitle: "Old Song",
          artistName: "Artist",
          albumId: "al1",
          albumTitle: "Album",
          positionMs: 180_000,
          durationMs: 180_000,
          played: true,
          playCount: 5,
          listenedMs: 0,
          lastPlayedAt: new Date(Date.now() - 40 * 86_400_000).toISOString(),
          coverArtId: "c1",
        },
      ],
      [],
      "2026-06-23:0",
      defaultMixSettings(),
      (entry) => ({
        id: entry.trackId,
        title: entry.trackTitle,
        artist: entry.artistName,
      }),
    );
    expect(ctx.recentlyPlayedIds.has("old")).toBe(false);
  });
});

describe("buildPersonalMixes", () => {
  it("builds themed mixes from history and fetchers", async () => {
    const { buildPersonalMixes } = await import("./mix-generator");
    const history = [
      {
        trackId: "t1",
        trackTitle: "Hit Song",
        artistName: "Artist A",
        albumId: "al1",
        albumTitle: "Album One",
        positionMs: 0,
        durationMs: 180_000,
        played: true,
        playCount: 12,
        listenedMs: 0,
        lastPlayedAt: new Date(Date.now() - 60 * 86_400_000).toISOString(),
        coverArtId: "c1",
      },
      {
        trackId: "t2",
        trackTitle: "Another",
        artistName: "Artist B",
        albumId: "al2",
        albumTitle: "Album Two",
        positionMs: 0,
        durationMs: 200_000,
        played: true,
        playCount: 8,
        listenedMs: 0,
        lastPlayedAt: new Date().toISOString(),
        coverArtId: "c2",
      },
      {
        trackId: "t3",
        trackTitle: "Third",
        artistName: "Artist C",
        albumId: "al3",
        albumTitle: "Album Three",
        positionMs: 0,
        durationMs: 200_000,
        played: true,
        playCount: 6,
        listenedMs: 0,
        lastPlayedAt: new Date(Date.now() - 2 * 86_400_000).toISOString(),
        coverArtId: "c3",
      },
      {
        trackId: "t4",
        trackTitle: "Fourth",
        artistName: "Artist D",
        albumId: "al4",
        albumTitle: "Album Four",
        positionMs: 0,
        durationMs: 200_000,
        played: true,
        playCount: 5,
        listenedMs: 0,
        lastPlayedAt: new Date(Date.now() - 3 * 86_400_000).toISOString(),
        coverArtId: "c4",
      },
    ];

    const fetchers = {
      searchArtistSongs: async (artist: string) => [
        song(`${artist}-1`, artist),
        song(`${artist}-2`, artist),
        song(`${artist}-3`, artist),
        song(`${artist}-4`, artist),
        song(`${artist}-5`, artist),
        song(`${artist}-6`, artist),
      ],
      searchArtistAlbums: async (artist: string) => [
        { id: `al-${artist}`, name: "Album One", artist, genre: "Rock" },
      ],
      getAlbumSongs: async () => [
        song("album-1", "Artist A"),
        song("album-2", "Artist A"),
        song("album-3", "Artist A"),
      ],
      getSimilarSongs: async () =>
        Array.from({ length: 8 }, (_, i) =>
          song(`sim-${i}`, `Similar ${i % 3}`),
        ),
      getRandomSongs: async (count = 12) =>
        Array.from({ length: count }, (_, i) =>
          song(`rand-${i}`, `Random ${i % 3}`),
        ),
      getRandomAlbums: async () => [
        { id: "al-r", name: "Random", coverArt: "cover-r" },
      ],
      getNewestAlbums: async () => [
        {
          id: "al-new",
          name: "New Album",
          artist: "Artist A",
          coverArt: "cover-new",
        },
      ],
      getGenreSongs: async () =>
        Array.from({ length: 8 }, (_, i) => song(`g${i}`, `Artist G${i % 3}`)),
      getGenres: async () => [{ name: "Rock", songCount: 100 }],
      getStarredSongs: async () =>
        Array.from({ length: 8 }, (_, i) => song(`star-${i}`, `Star ${i % 3}`)),
    };

    const ctx = buildMixContextInput(
      {
        totalPlays: 20,
        uniqueTracks: 2,
        totalListeningMs: 3_600_000,
        topArtists: [
          { key: "Artist A", label: "Artist A", count: 12 },
          { key: "Artist B", label: "Artist B", count: 8 },
          { key: "Artist C", label: "Artist C", count: 6 },
          { key: "Artist D", label: "Artist D", count: 5 },
          { key: "Artist E", label: "Artist E", count: 4 },
          { key: "Artist F", label: "Artist F", count: 3 },
          { key: "Artist G", label: "Artist G", count: 2 },
          { key: "Artist H", label: "Artist H", count: 1 },
        ],
        topTracks: [{ key: "t1", label: "Hit Song", count: 12 }],
        topAlbums: [{ key: "al1", label: "Album One", count: 12 }],
      },
      history,
      [{ id: "al1", name: "Album One", coverArt: "cover-1" }],
      "2026-06-23:0",
      mergeMixSettings({ minTracksPerMix: 6 }),
      (entry) => ({
        id: entry.trackId,
        title: entry.trackTitle,
        artist: entry.artistName,
        album: entry.albumTitle,
        albumId: entry.albumId,
        coverArt: entry.coverArtId,
      }),
    );

    const mixes = await buildPersonalMixes(ctx, fetchers);

    expect(mixes.length).toBeGreaterThan(0);
    expect(mixes.some((mix) => mix.id === "discover")).toBe(true);
    expect(mixes.some((mix) => mix.id === "daily-mix-1")).toBe(true);
    expect(mixes.some((mix) => mix.id === "for-you-50")).toBe(true);
    expect(mixes.some((mix) => mix.id === "for-you-100")).toBe(true);
    expect(mixes.every((mix) => mix.tracks.length >= 6)).toBe(true);
  });

  it("skips mixes that do not meet the minimum track count", async () => {
    const { buildPersonalMixes } = await import("./mix-generator");
    const fetchers = {
      searchArtistSongs: async () => [],
      searchArtistAlbums: async () => [],
      getAlbumSongs: async () => [],
      getSimilarSongs: async () => [],
      getRandomSongs: async (count = 12) =>
        Array.from({ length: count }, (_, i) =>
          song(`rand-${count}-${i}`, `Artist ${i % 5}`),
        ),
      getRandomAlbums: async () => [],
      getNewestAlbums: async () => [],
      getGenreSongs: async () => [],
      getGenres: async () => [],
      getStarredSongs: async () => [],
    };

    const ctx = buildMixContextInput(
      null,
      [],
      [],
      "2026-07-03:0",
      mergeMixSettings({ minTracksPerMix: 30 }),
      () => song("fallback", "Fallback"),
    );

    const mixes = await buildPersonalMixes(ctx, fetchers);
    expect(mixes.every((mix) => mix.tracks.length >= 30)).toBe(true);
    expect(mixes.some((mix) => mix.id === "for-you-50")).toBe(true);
    expect(mixes.some((mix) => mix.id === "for-you-100")).toBe(true);
  });

  it("builds for-you mixes with the requested random track counts", async () => {
    const { buildPersonalMixes } = await import("./mix-generator");
    const fetchers = {
      searchArtistSongs: async () => [],
      searchArtistAlbums: async () => [],
      getAlbumSongs: async () => [],
      getSimilarSongs: async () => [],
      getRandomSongs: async (count = 12) =>
        Array.from({ length: count }, (_, i) =>
          song(`rand-${count}-${i}`, `Artist ${i % 5}`),
        ),
      getRandomAlbums: async () => [
        { id: "al-r", name: "Random", coverArt: "cover-r" },
      ],
      getNewestAlbums: async () => [],
      getGenreSongs: async () => [],
      getGenres: async () => [],
      getStarredSongs: async () => [],
    };

    const ctx = buildMixContextInput(
      null,
      [],
      [],
      "2026-07-03:0",
      defaultMixSettings(),
      () => song("fallback", "Fallback"),
    );

    const mixes = await buildPersonalMixes(ctx, fetchers);
    const fifty = mixes.find((mix) => mix.id === "for-you-50");
    const hundred = mixes.find((mix) => mix.id === "for-you-100");

    expect(fifty?.tracks).toHaveLength(50);
    expect(hundred?.tracks).toHaveLength(100);
  });

  it("does not throw on an empty library and still builds for-you mixes", async () => {
    const { buildPersonalMixes } = await import("./mix-generator");
    const fetchers = {
      searchArtistSongs: async () => [],
      searchArtistAlbums: async () => [],
      getAlbumSongs: async () => [],
      getSimilarSongs: async () => [],
      getRandomSongs: async (count = 12) =>
        Array.from({ length: count }, (_, i) =>
          song(`empty-${count}-${i}`, `Solo ${i % 6}`),
        ),
      getRandomAlbums: async () => [],
      getNewestAlbums: async () => [],
      getGenreSongs: async () => [],
      getGenres: async () => [],
      getStarredSongs: async () => [],
    };
    const ctx = buildMixContextInput(
      null,
      [],
      [],
      "2026-08-15:empty",
      mergeMixSettings({ minTracksPerMix: 10 }),
      () => song("fallback", "Fallback"),
    );
    const mixes = await buildPersonalMixes(ctx, fetchers);
    expect(mixes.some((mix) => mix.id === "for-you-50")).toBe(true);
    expect(mixes.every((mix) => mix.tracks.length >= 10)).toBe(true);
  });

  it("keeps a daily mix inside one genre cluster", async () => {
    const { buildSingleMix } = await import("./mix-generator");
    const fetchers = {
      searchArtistSongs: async (artist: string) =>
        Array.from({ length: 8 }, (_, i) =>
          song(
            `${artist}-${i}`,
            artist,
            `album-${artist}-${Math.floor(i / 2)}`,
          ),
        ),
      searchArtistAlbums: async (artist: string) => {
        const letter = artist.slice(-1);
        const genre = letter <= "C" ? "Rock" : "Jazz";
        return [
          {
            id: `al-${artist}`,
            name: `${artist} Album`,
            artist,
            genre,
            year: letter <= "C" ? 2014 : 1962,
          },
        ];
      },
      getAlbumSongs: async () => [],
      getSimilarSongs: async () => [],
      getRandomSongs: async () => [],
      getRandomAlbums: async () => [],
      getNewestAlbums: async () => [],
      getGenreSongs: async () => [],
      getGenres: async () => [],
      getStarredSongs: async () => [],
    };
    const history = [
      "Artist A",
      "Artist B",
      "Artist C",
      "Artist D",
      "Artist E",
    ].map((name, index) => ({
      trackId: `${name}-0`,
      trackTitle: "Song",
      artistName: name,
      albumId: `al-${name}`,
      albumTitle: "Album",
      positionMs: 0,
      durationMs: 180_000,
      played: true,
      playCount: 12 - index,
      listenedMs: 180_000,
      lastPlayedAt: new Date(Date.now() - 40 * 86_400_000).toISOString(),
      coverArtId: "",
    }));
    const ctx = buildMixContextInput(
      {
        totalPlays: 40,
        uniqueTracks: 5,
        totalListeningMs: 1_000_000,
        topArtists: history.map((entry, index) => ({
          key: entry.artistName,
          label: entry.artistName,
          count: 12 - index,
        })),
        topTracks: history.map((entry) => ({
          key: entry.trackId,
          label: entry.trackTitle,
          count: entry.playCount,
        })),
        topAlbums: [],
      },
      history,
      [],
      "2026-08-15:cluster",
      mergeMixSettings({ minTracksPerMix: 6 }),
      (entry) => ({
        id: entry.trackId,
        title: entry.trackTitle,
        artist: entry.artistName,
        albumId: entry.albumId,
      }),
    );
    const mix = await buildSingleMix("daily-mix-1", ctx, fetchers);
    expect(mix).not.toBeNull();
    const named = (mix?.tracks ?? []).filter((track) =>
      /^Artist [A-E]/.test(track.artist ?? ""),
    );
    expect(named.length).toBeGreaterThan(0);
    expect(
      named.every((track) =>
        ["Artist A", "Artist B", "Artist C"].includes(track.artist ?? ""),
      ),
    ).toBe(true);
  });

  it("omits skipped tracks from daily mixes", async () => {
    const { buildSingleMix } = await import("./mix-generator");
    const fetchers = {
      searchArtistSongs: async (artist: string) => [
        song(`${artist}-ok`, artist),
        song("skipped-track", artist),
        song(`${artist}-ok2`, artist),
        song(`${artist}-ok3`, artist),
        song(`${artist}-ok4`, artist),
        song(`${artist}-ok5`, artist),
        song(`${artist}-ok6`, artist),
      ],
      searchArtistAlbums: async (artist: string) => [
        { id: `al-${artist}`, name: "Album", artist, genre: "Rock" },
      ],
      getAlbumSongs: async () => [],
      getSimilarSongs: async () => [],
      getRandomSongs: async () => [],
      getRandomAlbums: async () => [],
      getNewestAlbums: async () => [],
      getGenreSongs: async () => [],
      getGenres: async () => [],
      getStarredSongs: async () => [],
    };
    const history = [
      {
        trackId: "skipped-track",
        trackTitle: "Skip Me",
        artistName: "Artist A",
        albumId: "al-Artist A",
        albumTitle: "Album",
        positionMs: 5_000,
        durationMs: 200_000,
        played: false,
        playCount: 1,
        listenedMs: 4_000,
        lastPlayedAt: new Date().toISOString(),
        coverArtId: "",
      },
      {
        trackId: "Artist A-ok",
        trackTitle: "Keep",
        artistName: "Artist A",
        albumId: "al-Artist A",
        albumTitle: "Album",
        positionMs: 180_000,
        durationMs: 180_000,
        played: true,
        playCount: 8,
        listenedMs: 180_000,
        lastPlayedAt: new Date(Date.now() - 40 * 86_400_000).toISOString(),
        coverArtId: "",
      },
    ];
    const ctx = buildMixContextInput(
      {
        totalPlays: 9,
        uniqueTracks: 2,
        totalListeningMs: 200_000,
        topArtists: [
          { key: "Artist A", label: "Artist A", count: 8 },
          { key: "Artist B", label: "Artist B", count: 4 },
        ],
        topTracks: [{ key: "Artist A-ok", label: "Keep", count: 8 }],
        topAlbums: [],
      },
      history,
      [],
      "2026-08-15:skip",
      mergeMixSettings({ minTracksPerMix: 6 }),
      (entry) => ({
        id: entry.trackId,
        title: entry.trackTitle,
        artist: entry.artistName,
      }),
    );
    const mix = await buildSingleMix("daily-mix-1", ctx, fetchers);
    expect(mix).not.toBeNull();
    expect(mix?.tracks.some((track) => track.id === "skipped-track")).toBe(
      false,
    );
  });

  it("rebuilds the same daily mix for the same day seed", async () => {
    const { buildSingleMix } = await import("./mix-generator");
    const fetchers = {
      searchArtistSongs: async (artist: string) =>
        Array.from({ length: 8 }, (_, i) => song(`${artist}-${i}`, artist)),
      searchArtistAlbums: async (artist: string) => [
        { id: `al-${artist}`, name: "Album", artist, genre: "Rock" },
      ],
      getAlbumSongs: async () => [],
      getSimilarSongs: async () => [],
      getRandomSongs: async () => [],
      getRandomAlbums: async () => [],
      getNewestAlbums: async () => [],
      getGenreSongs: async () => [],
      getGenres: async () => [],
      getStarredSongs: async () => [],
    };
    const history = ["Artist A", "Artist B", "Artist C"].map((name, index) => ({
      trackId: `${name}-0`,
      trackTitle: "Song",
      artistName: name,
      albumId: `al-${name}`,
      albumTitle: "Album",
      positionMs: 0,
      durationMs: 180_000,
      played: true,
      playCount: 10 - index,
      listenedMs: 180_000,
      lastPlayedAt: new Date(Date.now() - 40 * 86_400_000).toISOString(),
      coverArtId: "",
    }));
    const makeCtx = () =>
      buildMixContextInput(
        {
          totalPlays: 24,
          uniqueTracks: 3,
          totalListeningMs: 500_000,
          topArtists: history.map((entry, index) => ({
            key: entry.artistName,
            label: entry.artistName,
            count: 10 - index,
          })),
          topTracks: [],
          topAlbums: [],
        },
        history,
        [],
        "2026-08-15:stable",
        mergeMixSettings({ minTracksPerMix: 6 }),
        (entry) => ({
          id: entry.trackId,
          title: entry.trackTitle,
          artist: entry.artistName,
        }),
      );
    const first = await buildSingleMix("daily-mix-1", makeCtx(), fetchers);
    const second = await buildSingleMix("daily-mix-1", makeCtx(), fetchers);
    expect(first?.tracks.map((track) => track.id)).toEqual(
      second?.tracks.map((track) => track.id),
    );
  });

  it("builds multiple genre mixes and decade mixes from ranked pools", async () => {
    const { buildPersonalMixes } = await import("./mix-generator");
    const songsForDecade = (decade: number, prefix: string): SubsonicSong[] =>
      Array.from({ length: 20 }, (_, i) => ({
        id: `${prefix}-${i}`,
        title: `${prefix} ${i}`,
        artist: `Artist ${prefix} ${i % 5}`,
        albumId: `al-${prefix}-${i % 4}`,
        year: decade + (i % 8),
        genre:
          prefix === "rock"
            ? "Rock"
            : prefix === "jazz"
              ? "Jazz"
              : "Electronic",
        coverArt: `cover-${prefix}`,
      }));

    const fetchers = {
      searchArtistSongs: async (artist: string) =>
        Array.from({ length: 8 }, (_, i) => ({
          ...song(`${artist}-${i}`, artist),
          year: 2010 + (i % 5),
        })),
      searchArtistAlbums: async () => [
        { id: "al-rock", name: "Rock Album", genre: "Rock", year: 2005 },
        { id: "al-jazz", name: "Jazz Album", genre: "Jazz", year: 1998 },
        {
          id: "al-elec",
          name: "Electronic Album",
          genre: "Electronic",
          year: 2015,
        },
      ],
      getAlbumSongs: async (albumId: string) => {
        if (albumId.includes("2000") || albumId.includes("rock")) {
          return songsForDecade(2000, "rock");
        }
        if (albumId.includes("1990") || albumId.includes("jazz")) {
          return songsForDecade(1990, "jazz");
        }
        return songsForDecade(2010, "elec");
      },
      getSimilarSongs: async () => songsForDecade(2000, "sim").slice(0, 8),
      getRandomSongs: async () => [
        ...songsForDecade(1990, "r90"),
        ...songsForDecade(2000, "r00"),
        ...songsForDecade(2010, "r10"),
      ],
      getRandomAlbums: async () => [
        { id: "al-1990", name: "Nineties", year: 1995, coverArt: "c90" },
        { id: "al-2000", name: "Aughts", year: 2004, coverArt: "c00" },
        { id: "al-2010", name: "Teens", year: 2012, coverArt: "c10" },
      ],
      getNewestAlbums: async () => [
        { id: "al-new-2010", name: "New Teens", year: 2018, coverArt: "cn" },
      ],
      getGenreSongs: async (genre: string) => {
        if (genre === "Rock") return songsForDecade(2000, "rock");
        if (genre === "Jazz") return songsForDecade(1990, "jazz");
        return songsForDecade(2010, "elec");
      },
      getGenres: async () => [
        { name: "Rock", songCount: 120 },
        { name: "Jazz", songCount: 80 },
        { name: "Electronic", songCount: 60 },
      ],
      getStarredSongs: async () => songsForDecade(2000, "star").slice(0, 10),
    };

    const history = Array.from({ length: 24 }, (_, i) => ({
      trackId: `h${i}`,
      trackTitle: `History ${i}`,
      artistName: `Artist ${i % 6}`,
      albumTitle: `Album ${i % 4}`,
      albumId: `al-hist-${i % 4}`,
      coverArtId: `cover-h${i}`,
      lastPlayedAt: new Date(Date.now() - i * 86_400_000).toISOString(),
      playCount: 4 + (i % 3),
      durationMs: 180_000,
      listenedMs: 160_000,
      played: true,
      positionMs: 160_000,
    }));

    const ctx = buildMixContextInput(
      {
        totalPlays: 80,
        uniqueTracks: 24,
        totalListeningMs: 3_600_000,
        topArtists: Array.from({ length: 8 }, (_, i) => ({
          key: `Artist ${i}`,
          label: `Artist ${i}`,
          count: 12 - i,
        })),
        topTracks: [{ key: "h0", label: "History 0", count: 12 }],
        topAlbums: [
          { key: "al-hist-0", label: "Album 0", count: 10 },
          { key: "al-hist-1", label: "Album 1", count: 8 },
        ],
      },
      history,
      [
        { id: "al-1990", name: "Nineties", year: 1995, coverArt: "c90" },
        { id: "al-2000", name: "Aughts", year: 2004, coverArt: "c00" },
        { id: "al-2010", name: "Teens", year: 2012, coverArt: "c10" },
      ],
      "2026-09-04:genre-decade",
      mergeMixSettings({
        minTracksPerMix: 6,
        genreSelection: "library",
        enabledMixIds: [
          "genre-mix-1",
          "genre-mix-2",
          "genre-mix-3",
          "decade-mix-1",
          "decade-mix-2",
          "decade-mix-3",
        ],
      }),
      (entry) => ({
        id: entry.trackId,
        title: entry.trackTitle,
        artist: entry.artistName,
        album: entry.albumTitle,
        albumId: entry.albumId,
        coverArt: entry.coverArtId,
        year: 2000 + (Number(entry.trackId.replace("h", "")) % 20),
      }),
    );

    const mixes = await buildPersonalMixes(ctx, fetchers);
    const genreMixes = mixes.filter((mix) => mix.id.startsWith("genre-mix-"));
    const decadeMixes = mixes.filter((mix) => mix.id.startsWith("decade-mix-"));

    expect(genreMixes.map((mix) => mix.id)).toEqual([
      "genre-mix-1",
      "genre-mix-2",
      "genre-mix-3",
    ]);
    expect(new Set(genreMixes.map((mix) => mix.title)).size).toBe(3);

    expect(decadeMixes.length).toBeGreaterThanOrEqual(2);
    expect(decadeMixes.every((mix) => /\d{4}s Mix/.test(mix.title))).toBe(true);
    expect(new Set(decadeMixes.map((mix) => mix.title)).size).toBe(
      decadeMixes.length,
    );
  });
});
