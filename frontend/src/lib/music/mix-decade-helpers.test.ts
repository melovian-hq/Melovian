// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  albumYearIndex,
  rankedDecades,
  trackMatchesDecade,
} from "./mix-generator/helpers";
import { buildMixContextInput } from "./mix-generator";
import { mergeMixSettings } from "./mix-settings";

describe("decade mix helpers", () => {
  it("ranks multiple decades from taste and frequent albums", () => {
    const ctx = buildMixContextInput(
      {
        totalPlays: 10,
        uniqueTracks: 3,
        totalListeningMs: 600_000,
        topArtists: [],
        topTracks: [],
        topAlbums: [],
      },
      [
        {
          trackId: "t1",
          trackTitle: "A",
          artistName: "A",
          albumId: "al-1",
          albumTitle: "Album",
          coverArtId: "c1",
          lastPlayedAt: new Date().toISOString(),
          playCount: 5,
          durationMs: 180_000,
          listenedMs: 180_000,
          played: true,
          positionMs: 180_000,
        },
      ],
      [
        { id: "al-90", name: "Nineties", year: 1994 },
        { id: "al-00", name: "Aughts", year: 2003 },
        { id: "al-00b", name: "Aughts Two", year: 2008 },
        { id: "al-10", name: "Teens", year: 2011 },
      ],
      "seed",
      mergeMixSettings({}),
      (entry) => ({
        id: entry.trackId,
        title: entry.trackTitle,
        artist: entry.artistName,
        year: 2005,
      }),
    );

    expect(rankedDecades(ctx, 3)).toEqual([2000, 1990, 2010]);
  });

  it("matches decade from album year when track year is missing", () => {
    const years = albumYearIndex([
      { id: "al-1", name: "Album", year: 1997 },
      { id: "al-2", name: "Other", year: 2014 },
    ]);

    expect(
      trackMatchesDecade(
        { id: "t1", title: "A", albumId: "al-1" },
        1990,
        years,
      ),
    ).toBe(true);
    expect(
      trackMatchesDecade(
        { id: "t2", title: "B", albumId: "al-1" },
        2000,
        years,
      ),
    ).toBe(false);
    expect(
      trackMatchesDecade({ id: "t3", title: "C", year: 2016 }, 2010, years),
    ).toBe(true);
  });
});
