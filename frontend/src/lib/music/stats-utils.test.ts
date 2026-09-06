// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  computeStatsFromHistory,
  filterHistoryByPeriod,
  uniqueRecentHistory,
} from "./stats-utils";
import type { ListenEntry } from "$lib/subsonic/types";

const entry = (
  overrides: Partial<ListenEntry> & Pick<ListenEntry, "trackId">,
): ListenEntry => ({
  trackId: overrides.trackId,
  trackTitle: overrides.trackTitle ?? "Track",
  artistName: overrides.artistName ?? "Artist",
  albumId: overrides.albumId ?? "al-1",
  albumTitle: overrides.albumTitle ?? "Album",
  positionMs: 0,
  durationMs: 180_000,
  played: true,
  playCount: overrides.playCount ?? 1,
  listenedMs: overrides.listenedMs ?? 0,
  lastPlayedAt: overrides.lastPlayedAt ?? new Date().toISOString(),
  coverArtId: "",
});

describe("stats-utils", () => {
  it("filters history by period", () => {
    const old = entry({
      trackId: "old",
      lastPlayedAt: new Date(Date.now() - 40 * 86_400_000).toISOString(),
    });
    const recent = entry({ trackId: "recent" });
    expect(filterHistoryByPeriod([old, recent], "30d")).toHaveLength(1);
    expect(filterHistoryByPeriod([old, recent], "all")).toHaveLength(2);
  });

  it("filters history by calendar year", () => {
    const in2024 = entry({
      trackId: "y2024",
      lastPlayedAt: "2024-08-01T12:00:00.000Z",
    });
    const in2025 = entry({
      trackId: "y2025",
      lastPlayedAt: "2025-02-01T12:00:00.000Z",
    });
    expect(filterHistoryByPeriod([in2024, in2025], "year:2024")).toHaveLength(
      1,
    );
    expect(filterHistoryByPeriod([in2024, in2025], "year:2025")).toHaveLength(
      1,
    );
  });

  it("computes top artists and tracks from history", () => {
    const stats = computeStatsFromHistory([
      entry({ trackId: "t1", artistName: "Alpha", playCount: 3 }),
      entry({ trackId: "t2", artistName: "Beta", playCount: 1 }),
    ]);
    expect(stats.totalPlays).toBe(4);
    expect(stats.topArtists[0]?.label).toBe("Alpha");
    expect(stats.topArtists[0]?.count).toBe(3);
  });

  it("keeps only the latest play per track in recent history", () => {
    const unique = uniqueRecentHistory([
      entry({
        trackId: "t1",
        trackTitle: "Older",
        lastPlayedAt: "2024-01-01T12:00:00.000Z",
      }),
      entry({
        trackId: "t1",
        trackTitle: "Newer",
        lastPlayedAt: "2025-01-01T12:00:00.000Z",
      }),
      entry({
        trackId: "t2",
        trackTitle: "Other",
        lastPlayedAt: "2024-06-01T12:00:00.000Z",
      }),
    ]);
    expect(unique).toHaveLength(2);
    expect(unique[0]?.trackTitle).toBe("Newer");
  });
});
