// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  buildTasteProfile,
  completionRatio,
  listenAffinityWeight,
  scoreTrackTaste,
} from "./taste-score";
import type { ListenEntry, SubsonicSong } from "$lib/subsonic/types";

const song = (id: string, artist: string): SubsonicSong => ({
  id,
  title: id,
  artist,
});

const entry = (
  overrides: Partial<ListenEntry> & Pick<ListenEntry, "trackId">,
): ListenEntry => ({
  trackId: overrides.trackId,
  trackTitle: overrides.trackTitle ?? overrides.trackId,
  artistName: overrides.artistName ?? "Artist",
  albumId: overrides.albumId ?? "al",
  albumTitle: overrides.albumTitle ?? "Album",
  positionMs: overrides.positionMs ?? 0,
  durationMs: overrides.durationMs ?? 180_000,
  played: overrides.played ?? true,
  playCount: overrides.playCount ?? 1,
  listenedMs: overrides.listenedMs ?? 0,
  lastPlayedAt: overrides.lastPlayedAt ?? new Date().toISOString(),
  coverArtId: "",
});

describe("taste-score listen time unit", () => {
  it("weights affinity by listened ms over play count alone", () => {
    expect(
      listenAffinityWeight(
        entry({ trackId: "a", playCount: 1, listenedMs: 600_000 }),
      ),
    ).toBeGreaterThan(
      listenAffinityWeight(
        entry({ trackId: "b", playCount: 20, listenedMs: 0 }),
      ),
    );
  });

  it("computes completion ratio from listened ms", () => {
    expect(
      completionRatio(
        entry({ trackId: "t", durationMs: 100_000, listenedMs: 80_000 }),
      ),
    ).toBeCloseTo(0.8);
    expect(
      completionRatio(entry({ trackId: "t", played: true, listenedMs: 0 })),
    ).toBe(1);
  });

  it("scores high listen-time tracks above play-count-only peers", () => {
    const deepListen = entry({
      trackId: "deep",
      artistName: "Solo",
      playCount: 1,
      listenedMs: 540_000,
      durationMs: 180_000,
    });
    const spamPlays = entry({
      trackId: "spam",
      artistName: "Solo",
      playCount: 8,
      listenedMs: 8_000,
      durationMs: 180_000,
    });
    const profile = buildTasteProfile(
      null,
      [deepListen, spamPlays],
      new Map([
        ["deep", 1],
        ["spam", 8],
      ]),
      new Set(),
      [],
      new Map([
        ["deep", 540_000],
        ["spam", 8_000],
      ]),
    );

    const deepScore = scoreTrackTaste(song("deep", "Solo"), profile);
    const spamScore = scoreTrackTaste(song("spam", "Solo"), profile);
    expect(deepScore).toBeGreaterThan(spamScore);
  });

  it("soft-penalizes low listen-ratio repeats", () => {
    const skippedish = entry({
      trackId: "skippy",
      artistName: "Band",
      playCount: 4,
      listenedMs: 20_000,
      durationMs: 200_000,
    });
    const solid = entry({
      trackId: "solid",
      artistName: "Band",
      playCount: 4,
      listenedMs: 700_000,
      durationMs: 200_000,
    });
    const profile = buildTasteProfile(
      null,
      [skippedish, solid],
      new Map([
        ["skippy", 4],
        ["solid", 4],
      ]),
      new Set(),
      [],
      new Map([
        ["skippy", 20_000],
        ["solid", 700_000],
      ]),
    );
    expect(scoreTrackTaste(song("solid", "Band"), profile)).toBeGreaterThan(
      scoreTrackTaste(song("skippy", "Band"), profile),
    );
  });

  it("still hard-excludes inferred skips", () => {
    const profile = buildTasteProfile(
      null,
      [],
      new Map([["x", 10]]),
      new Set(["x"]),
      [],
      new Map([["x", 900_000]]),
    );
    expect(scoreTrackTaste(song("x", "Anyone"), profile)).toBeLessThan(0);
  });
});
