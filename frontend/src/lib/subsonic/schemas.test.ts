// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import * as v from "valibot";
import {
  listenEntrySchema,
  musicPlaylistSchema,
  musicStatusSchema,
  subsonicSearchResponseSchema,
} from "./schemas";

const listenEntry = {
  trackId: "t1",
  trackTitle: "Song",
  artistName: "Artist",
  albumId: "a1",
  albumTitle: "Album",
  positionMs: 0,
  durationMs: 120000,
  played: true,
  playCount: 2,
  listenedMs: 90000,
  lastPlayedAt: "2026-01-01T00:00:00Z",
  coverArtId: "a1",
};

describe("listenEntrySchema", () => {
  it("accepts a full listen entry", () => {
    expect(v.safeParse(listenEntrySchema, listenEntry).success).toBe(true);
  });

  it("rejects entries missing required fields", () => {
    const { trackTitle: _dropped, ...partial } = listenEntry;
    expect(v.safeParse(listenEntrySchema, partial).success).toBe(false);
  });
});

describe("musicPlaylistSchema", () => {
  const playlist = {
    id: "pl1",
    name: "Mix",
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-02T00:00:00Z",
    trackCount: 0,
  };

  it("normalizes null list fields to undefined", () => {
    const result = v.parse(musicPlaylistSchema, {
      ...playlist,
      tracks: null,
      coverArtIds: null,
    });
    expect(result.tracks).toBeUndefined();
    expect(result.coverArtIds).toBeUndefined();
  });

  it("keeps parsed tracks", () => {
    const result = v.parse(musicPlaylistSchema, {
      ...playlist,
      trackCount: 1,
      tracks: [
        {
          trackId: "t1",
          trackTitle: "Song",
          artistName: "Artist",
          albumId: "a1",
          albumTitle: "Album",
          durationMs: 1000,
          coverArtId: "a1",
        },
      ],
    });
    expect(result.tracks).toHaveLength(1);
  });

  it("rejects a missing trackCount", () => {
    const { trackCount: _dropped, ...partial } = playlist;
    expect(v.safeParse(musicPlaylistSchema, partial).success).toBe(false);
  });
});

describe("musicStatusSchema", () => {
  it("accepts source values beyond the declared union", () => {
    const result = v.parse(musicStatusSchema, {
      enabled: true,
      connected: true,
      source: "unified",
    });
    expect(result.source).toBe("unified");
  });

  it("rejects non-boolean connected", () => {
    expect(
      v.safeParse(musicStatusSchema, { enabled: true, connected: "yes" })
        .success,
    ).toBe(false);
  });
});

describe("subsonicSearchResponseSchema", () => {
  it("accepts empty and null containers", () => {
    expect(
      v.parse(subsonicSearchResponseSchema, {}).searchResult3,
    ).toBeUndefined();
    expect(
      v.parse(subsonicSearchResponseSchema, { searchResult3: null })
        .searchResult3,
    ).toBeNull();
  });
});
