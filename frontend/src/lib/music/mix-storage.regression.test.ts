// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  ALL_MIX_IDS,
  defaultMixSettings,
  mergeMixSettings,
} from "./mix-settings";
import { mixTrackCount, slimPersonalMix } from "./mix-storage";
import type { GeneratedMix } from "./mix-generator";

describe("mix settings and storage regression", () => {
  it("exposes the full mix id catalog", () => {
    expect(ALL_MIX_IDS.length).toBeGreaterThanOrEqual(10);
    const settings = defaultMixSettings();
    expect(settings.minTracksPerMix).toBeGreaterThan(0);
    expect(settings.maxTracksPerMix).toBeGreaterThanOrEqual(
      settings.minTracksPerMix,
    );
  });

  it("clamps mix track floors to safe ranges", () => {
    const merged = mergeMixSettings({
      minTracksPerMix: 1,
      maxTracksPerMix: 5,
    });
    expect(merged.minTracksPerMix).toBeGreaterThanOrEqual(5);
    expect(merged.maxTracksPerMix).toBeGreaterThanOrEqual(10);
  });

  it("slim mix preserves track count without keeping full track objects", () => {
    const mix: GeneratedMix = {
      id: "on-repeat",
      title: "On Repeat",
      subtitle: "test",
      gradient: "linear-gradient(#000,#111)",
      tracks: Array.from({ length: 40 }, (_, i) => ({
        id: `t${i}`,
        title: `Song ${i}`,
        artist: "A",
        album: "B",
        duration: 100,
      })),
    };
    const slim = slimPersonalMix(mix);
    expect(mixTrackCount(slim)).toBe(40);
    expect(slim.tracks.length).toBe(0);
  });
});

describe("mix storage crash safety", () => {
  it("handles empty mixes", () => {
    const slim = slimPersonalMix({
      id: "discover",
      title: "Discover",
      subtitle: "",
      gradient: "",
      tracks: [],
    });
    expect(mixTrackCount(slim)).toBe(0);
  });
});
