// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  createPersonalRadioState,
  isPersonalColdStart,
  notePersonalComplete,
  notePersonalSkip,
  wasTrackSkipped,
} from "./personal-radio";

describe("personal-radio", () => {
  it("detects cold start from sparse history", () => {
    expect(isPersonalColdStart([], null)).toBe(true);
    expect(
      isPersonalColdStart([], {
        totalPlays: 20,
        uniqueTracks: 10,
        totalListeningMs: 1000,
        topArtists: [],
        topTracks: [],
        topAlbums: [],
      }),
    ).toBe(false);
  });

  it("tracks skips and completions", () => {
    const state = createPersonalRadioState();
    expect(state.sessionSeed).toMatch(/^personal-radio:/);
    expect(state.similarCache.size).toBe(0);
    expect(state.recentCompletions).toEqual([]);
    notePersonalSkip(state, { id: "t1", title: "A", artist: "Alpha" });
    expect(state.skipTrackIds.has("t1")).toBe(true);
    expect(state.skipArtistKeys.get("alpha")).toBe(1);

    notePersonalComplete(state, { id: "t1", title: "A", artist: "Alpha" });
    expect(state.skipTrackIds.has("t1")).toBe(false);
    expect(state.lastCompletedId).toBe("t1");
    expect(state.recentCompletions.map((t) => t.id)).toEqual(["t1"]);

    notePersonalComplete(state, { id: "t2", title: "B", artist: "Beta" });
    expect(state.recentCompletions.map((t) => t.id)).toEqual(["t2", "t1"]);
  });

  it("infers skips from incomplete progress", () => {
    expect(wasTrackSkipped(10_000, 200_000)).toBe(true);
    expect(wasTrackSkipped(180_000, 200_000)).toBe(false);
  });
});
