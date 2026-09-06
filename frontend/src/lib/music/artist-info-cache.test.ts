// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  clearArtistInfoMemoryCache,
  fetchArtistInfoWithCache,
  invalidateArtistInfoCache,
  prefetchArtistInfo,
} from "./artist-info-cache";
import type { MusicLibraryAdapter } from "$lib/music/library-adapter";

function library(
  getArtistInfo: MusicLibraryAdapter["getArtistInfo"],
): MusicLibraryAdapter {
  return { getArtistInfo } as MusicLibraryAdapter;
}

describe("artist-info-cache", () => {
  beforeEach(() => {
    invalidateArtistInfoCache();
    localStorage.clear();
  });

  it("returns cached info without refetching", async () => {
    const getArtistInfo = vi.fn(async () => ({
      biography: "Bio",
      similarArtists: [{ id: "ar2", name: "Other" }],
    })) as MusicLibraryAdapter["getArtistInfo"];
    const lib = library(getArtistInfo);

    const first = await fetchArtistInfoWithCache(lib, "ar1");
    const second = await fetchArtistInfoWithCache(lib, "ar1");

    expect(first.biography).toBe("Bio");
    expect(second.similarArtists).toHaveLength(1);
    expect(getArtistInfo).toHaveBeenCalledTimes(1);
  });

  it("persists info to localStorage for repeat visits", async () => {
    const getArtistInfo = vi.fn(async () => ({
      biography: "Stored bio",
      similarArtists: [{ id: "ar2", name: "Peer" }],
    })) as MusicLibraryAdapter["getArtistInfo"];
    const lib = library(getArtistInfo);

    await fetchArtistInfoWithCache(lib, "ar1");
    clearArtistInfoMemoryCache("ar1");

    const restored = await fetchArtistInfoWithCache(lib, "ar1");
    expect(restored.biography).toBe("Stored bio");
    expect(getArtistInfo).toHaveBeenCalledTimes(1);
  });

  it("prefetches without duplicate requests", async () => {
    const getArtistInfo = vi.fn(async () => ({
      similarArtists: [],
    })) as MusicLibraryAdapter["getArtistInfo"];
    const lib = library(getArtistInfo);

    prefetchArtistInfo(lib, "ar1");
    prefetchArtistInfo(lib, "ar1");
    await fetchArtistInfoWithCache(lib, "ar1");

    expect(getArtistInfo).toHaveBeenCalledTimes(1);
  });

  it("skips remote lookup for local artists", async () => {
    const getArtistInfo = vi.fn(async () => ({
      similarArtists: [{ name: "Local" }],
    })) as MusicLibraryAdapter["getArtistInfo"];
    const lib = library(getArtistInfo);

    const info = await fetchArtistInfoWithCache(lib, "art_abc");

    expect(info.similarArtists).toEqual([]);
    expect(getArtistInfo).not.toHaveBeenCalled();
  });
});
