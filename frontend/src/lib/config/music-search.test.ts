// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from "vitest";
import type { SubsonicSearchResult } from "$lib/subsonic";
import type { MusicLibraryAdapter } from "$lib/music/library-adapter";

const emptyResult: SubsonicSearchResult = {
  artists: [],
  albums: [],
  songs: [],
};

const mockLibrary = {
  search3: vi.fn(),
  getArtists: vi.fn(),
  getSimilarSongs: vi.fn(),
} as unknown as MusicLibraryAdapter;

vi.mock("$lib/features/sources/store.svelte", () => ({
  sources: {
    hasLocalActive: false,
    hasSubsonicActive: true,
    needsSetup: false,
    ready: true,
    activeLabel: "Server",
  },
}));

vi.mock("$lib/music/library-adapter", () => ({
  createLocalLibraryAdapter: vi.fn(),
  createSubsonicLibraryAdapter: vi.fn(() => mockLibrary),
}));

import { music } from "$lib/config/music.svelte";

const mockedSearch3 = vi.mocked(mockLibrary.search3);
const mockedGetArtists = vi.mocked(mockLibrary.getArtists);
const mockedGetSimilar = vi.mocked(mockLibrary.getSimilarSongs);

describe("music.searchAll", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("returns an empty result without querying for blank input", async () => {
    const res = await music.searchAll("   ");
    expect(res.query).toBe("");
    expect(res.suggestion).toBeNull();
    expect(mockedSearch3).not.toHaveBeenCalled();
  });

  it("offers a did-you-mean suggestion from sparse search hits", async () => {
    mockedSearch3.mockResolvedValue({
      artists: [{ id: "1", name: "Daft Punk" }],
      albums: [],
      songs: [],
    });
    mockedGetSimilar.mockResolvedValue([]);

    const res = await music.searchAll("daft pank");

    expect(mockedGetArtists).not.toHaveBeenCalled();
    expect(res.suggestion).toBe("Daft Punk");
  });

  it("does not suggest when sparse search has no label candidates", async () => {
    mockedSearch3.mockResolvedValue(emptyResult);
    mockedGetSimilar.mockResolvedValue([]);

    const res = await music.searchAll("zzzzzz");

    expect(mockedGetArtists).not.toHaveBeenCalled();
    expect(res.suggestion).toBeNull();
  });

  it("does not block on similar tracks", async () => {
    mockedSearch3.mockResolvedValue({
      artists: [],
      albums: [],
      songs: [
        { id: "song-1", title: "One More Time" },
        { id: "song-2", title: "Aerodynamic" },
      ] as SubsonicSearchResult["songs"],
    });

    const res = await music.searchAll("daft punk");

    expect(mockedGetSimilar).not.toHaveBeenCalled();
    expect(res.similar).toEqual([]);
  });
});

describe("music.searchSimilarTracks", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("fetches similar tracks for a song id", async () => {
    mockedGetSimilar.mockResolvedValue([
      {
        id: "sim-1",
        title: "Digital Love",
      } as SubsonicSearchResult["songs"][number],
    ]);

    const similar = await music.searchSimilarTracks("song-1", 12);

    expect(mockedGetSimilar).toHaveBeenCalledWith("song-1", 12);
    expect(similar).toHaveLength(1);
  });
});
