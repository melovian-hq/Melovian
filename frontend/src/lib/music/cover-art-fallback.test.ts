// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  albumCoverPaletteKey,
  albumCoverSeed,
  coverArtFallbackUrl,
  trackCoverPaletteKey,
  trackCoverSeed,
} from "./cover-art-fallback";

describe("coverArtFallbackUrl", () => {
  it("returns a stable data url for the same seed", () => {
    const first = coverArtFallbackUrl("track-1", "Artist A");
    const second = coverArtFallbackUrl("track-1", "Artist A");
    expect(first).toBe(second);
    expect(first.startsWith("data:image/svg+xml,")).toBe(true);
  });

  it("varies by seed and palette key", () => {
    const bySeed = coverArtFallbackUrl("track-1", "Artist A");
    const byPalette = coverArtFallbackUrl("track-2", "Artist A");
    const byArtist = coverArtFallbackUrl("track-1", "Artist B");
    expect(bySeed).not.toBe(byPalette);
    expect(bySeed).not.toBe(byArtist);
  });
});

describe("cover seed helpers", () => {
  it("prefers album id for track art", () => {
    expect(trackCoverSeed({ id: "trk_1", albumId: "alb_1" })).toBe("alb_1");
  });

  it("uses artist for palette key", () => {
    expect(
      trackCoverPaletteKey({
        artist: "Band",
        album: "Album",
        title: "Song",
      }),
    ).toBe("Band");
  });

  it("uses album id and artist name", () => {
    expect(albumCoverSeed({ id: "alb_1" })).toBe("alb_1");
    expect(albumCoverPaletteKey({ artist: "Band", name: "Album" })).toBe(
      "Band",
    );
  });
});
