// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  albumPlaybackTracks,
  groupTrackVersions,
  trackQualityScore,
} from "./track-versions";
import type { SubsonicSong } from "$lib/subsonic/types";

function song(
  id: string,
  title: string,
  extra: Partial<SubsonicSong> = {},
): SubsonicSong {
  return { id, title, ...extra };
}

describe("trackQualityScore", () => {
  it("prefers lossless sources over mp3", () => {
    const flac = song("1", "Track", { suffix: "flac" });
    const mp3 = song("2", "Track", { suffix: "mp3", bitRate: 320 });
    expect(trackQualityScore(flac)).toBeGreaterThan(trackQualityScore(mp3));
  });
});

describe("groupTrackVersions", () => {
  it("groups duplicate track numbers and titles", () => {
    const songs = [
      song("a", "Airbag", { track: 1, suffix: "mp3", bitRate: 128 }),
      song("b", "Airbag", { track: 1, suffix: "flac" }),
      song("c", "Paranoid Android", { track: 2, suffix: "flac" }),
    ];
    const groups = groupTrackVersions(songs);
    expect(groups).toHaveLength(2);
    expect(groups[0]?.versions).toHaveLength(2);
    expect(groups[0]?.primary.id).toBe("b");
    expect(groups[1]?.versions).toHaveLength(1);
  });
});

describe("albumPlaybackTracks", () => {
  it("returns one preferred version per logical track", () => {
    const songs = [
      song("a", "Airbag", { track: 1, suffix: "mp3", bitRate: 128 }),
      song("b", "Airbag", { track: 1, suffix: "flac" }),
      song("c", "Paranoid Android", { track: 2, suffix: "flac" }),
    ];
    expect(albumPlaybackTracks(songs).map((track) => track.id)).toEqual([
      "b",
      "c",
    ]);
  });
});
