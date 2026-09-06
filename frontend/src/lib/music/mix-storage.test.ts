// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  fromCachedMix,
  mixTrackCount,
  normalizeLoadedMix,
  slimPersonalMix,
  toCachedMix,
} from "./mix-storage";
import type { GeneratedMix } from "./mix-generator";

const fullMix = (): GeneratedMix => ({
  id: "discover",
  title: "Discover",
  subtitle: "Fresh picks",
  tracks: [
    { id: "t1", title: "One", artist: "Artist" },
    { id: "t2", title: "Two", artist: "Artist" },
  ],
  coverArtId: "cover-1",
  gradient: "linear-gradient(red, blue)",
});

describe("mix-storage", () => {
  it("counts tracks from ids when the track list is empty", () => {
    const slim = slimPersonalMix(fullMix());
    expect(mixTrackCount(slim)).toBe(2);
    expect(slim.tracks).toEqual([]);
    expect(slim.trackIds).toEqual(["t1", "t2"]);
  });

  it("round-trips cached mixes without full song objects", () => {
    const cached = toCachedMix(fullMix());
    const restored = fromCachedMix(cached);
    expect(restored.tracks).toEqual([]);
    expect(restored.trackIds).toEqual(["t1", "t2"]);
    expect(mixTrackCount(restored)).toBe(2);
  });

  it("normalizes legacy cached mixes that still include tracks", () => {
    const legacy = fullMix();
    const normalized = normalizeLoadedMix(legacy);
    expect(normalized.tracks).toEqual([]);
    expect(normalized.trackIds).toEqual(["t1", "t2"]);
  });
});
