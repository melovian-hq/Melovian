// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import {
  fetchAlbumWithCache,
  fetchArtistWithCache,
  resetSubsonicDetailCaches,
} from "./detail-cache";

describe("subsonic detail cache", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    resetSubsonicDetailCaches();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns cached album data without refetching", async () => {
    const getAlbum = vi.fn().mockResolvedValue({
      album: { id: "a1", name: "Album" },
      songs: [],
    });
    const library = { getAlbum } as unknown as MusicLibraryAdapter;

    await fetchAlbumWithCache(library, "a1");
    await fetchAlbumWithCache(library, "a1");

    expect(getAlbum).toHaveBeenCalledTimes(1);
  });

  it("serves stale album data immediately and refreshes in background path", async () => {
    const getAlbum = vi
      .fn()
      .mockResolvedValueOnce({
        album: { id: "a1", name: "Old" },
        songs: [],
      })
      .mockResolvedValueOnce({
        album: { id: "a1", name: "New" },
        songs: [],
      });
    const library = { getAlbum } as unknown as MusicLibraryAdapter;

    const first = await fetchAlbumWithCache(library, "a1");
    vi.advanceTimersByTime(5 * 60 * 1000 + 1);

    let staleShown = false;
    const second = await fetchAlbumWithCache(library, "a1", () => {
      staleShown = true;
    });

    expect(first.album.name).toBe("Old");
    expect(staleShown).toBe(true);
    expect(second.album.name).toBe("New");
    expect(getAlbum).toHaveBeenCalledTimes(2);
  });

  it("caches artist detail separately", async () => {
    const getArtist = vi.fn().mockResolvedValue({
      artist: { id: "ar1", name: "Artist" },
      albums: [],
    });
    const library = { getArtist } as unknown as MusicLibraryAdapter;

    await fetchArtistWithCache(library, "ar1");
    await fetchArtistWithCache(library, "ar1");

    expect(getArtist).toHaveBeenCalledTimes(1);
  });
});
