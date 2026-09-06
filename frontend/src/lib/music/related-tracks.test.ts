// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";
import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import type { SubsonicSong } from "$lib/subsonic";
import { loadRelatedTracks } from "./related-tracks";

function track(
  id: string,
  overrides: Partial<SubsonicSong> = {},
): SubsonicSong {
  return {
    id,
    title: `Track ${id}`,
    artist: "Artist",
    ...overrides,
  };
}

function mockLibrary(impl: Partial<MusicLibraryAdapter>): MusicLibraryAdapter {
  return {
    getArtists: vi.fn(),
    getArtist: vi.fn(),
    getAlbum: vi.fn(),
    getAlbumList2: vi.fn(),
    search3: vi.fn(),
    getSong: vi.fn(),
    getRandomSongs: vi.fn(),
    getGenres: vi.fn(),
    getSongsByGenre: vi.fn(),
    getServerPlaylists: vi.fn(),
    getServerPlaylist: vi.fn(),
    getInternetRadioStations: vi.fn(),
    getStarred2: vi.fn(),
    star: vi.fn(),
    unstar: vi.fn(),
    scrobble: vi.fn(),
    getSimilarSongs: vi.fn(),
    searchArtistAlbums: vi.fn(),
    getArtistInfo: vi.fn(),
    getLyricsForSong: vi.fn(),
    searchLyricsByText: vi.fn(),
    ...impl,
  } as MusicLibraryAdapter;
}

describe("loadRelatedTracks", () => {
  it("returns similar songs when available", async () => {
    const library = mockLibrary({
      getSimilarSongs: vi
        .fn()
        .mockResolvedValue([track("sim-1"), track("current")]),
    });

    const result = await loadRelatedTracks(library, track("current"));
    expect(result.map((item) => item.id)).toEqual(["sim-1"]);
  });

  it("falls back to album and artist tracks when similar is empty", async () => {
    const library = mockLibrary({
      getSimilarSongs: vi.fn().mockResolvedValue([]),
      getAlbum: vi.fn().mockResolvedValue({
        album: { id: "alb-1", name: "Album" },
        songs: [track("current"), track("alb-track")],
      }),
      getArtist: vi.fn().mockResolvedValue({
        artist: { id: "art-1", name: "Artist" },
        albums: [{ id: "alb-2", name: "Other album" }],
      }),
      search3: vi.fn().mockResolvedValue({
        artists: [],
        albums: [],
        songs: [track("search-track", { artist: "Artist" })],
      }),
    });

    const result = await loadRelatedTracks(
      library,
      track("current", { albumId: "alb-1", artistId: "art-1" }),
    );

    expect(result.map((item) => item.id)).toEqual([
      "alb-track",
      "search-track",
    ]);
  });

  it("falls back when similar only returns the current track", async () => {
    const library = mockLibrary({
      getSimilarSongs: vi.fn().mockResolvedValue([track("current")]),
      getAlbum: vi.fn().mockResolvedValue({
        album: { id: "alb-1", name: "Album" },
        songs: [track("current"), track("sibling")],
      }),
    });

    const result = await loadRelatedTracks(
      library,
      track("current", { albumId: "alb-1" }),
    );

    expect(result.map((item) => item.id)).toEqual(["sibling"]);
  });
});
