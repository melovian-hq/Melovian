// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  albumHasUnknownMetadata,
  artistHasUnknownMetadata,
  isUnknownAlbumName,
  isUnknownArtistName,
  rejectUnknownAlbums,
  rejectUnknownArtists,
  rejectUnknownListenEvents,
  rejectUnknownTracks,
  trackHasUnknownMetadata,
} from "./unknown-metadata";

describe("unknown metadata labels", () => {
  it("treats bracketed and plain unknown artist names as unknown", () => {
    expect(isUnknownArtistName("[Unknown Artist]")).toBe(true);
    expect(isUnknownArtistName("Unknown Artist")).toBe(true);
    expect(isUnknownArtistName(" unknown ")).toBe(true);
    expect(isUnknownArtistName("")).toBe(true);
    expect(isUnknownArtistName(undefined)).toBe(true);
  });

  it("keeps named artists including Various Artists", () => {
    expect(isUnknownArtistName("Slowdive")).toBe(false);
    expect(isUnknownArtistName("Various Artists")).toBe(false);
    expect(isUnknownArtistName("8BP050")).toBe(false);
  });

  it("treats unknown album titles as unknown", () => {
    expect(isUnknownAlbumName("[Unknown Album]")).toBe(true);
    expect(isUnknownAlbumName("Unknown Album")).toBe(true);
    expect(isUnknownAlbumName("Afscheid")).toBe(false);
  });

  it("flags albums with an unknown artist even when the title is real", () => {
    expect(
      albumHasUnknownMetadata({
        name: "8BP050",
        artist: "[Unknown Artist]",
      }),
    ).toBe(true);
    expect(
      albumHasUnknownMetadata({ name: "Souvlaki", artist: "Slowdive" }),
    ).toBe(false);
  });

  it("flags artists and tracks the same way", () => {
    expect(artistHasUnknownMetadata({ name: "[Unknown Artist]" })).toBe(true);
    expect(
      trackHasUnknownMetadata({
        artist: "[Unknown Artist]",
        album: "Afscheid",
      }),
    ).toBe(true);
    expect(
      trackHasUnknownMetadata({
        artist: "Slowdive",
        album: "[Unknown Album]",
      }),
    ).toBe(true);
  });
});

describe("unknown metadata filters", () => {
  const albums = [
    { id: "a", name: "Souvlaki", artist: "Slowdive" },
    { id: "b", name: "8BP050", artist: "[Unknown Artist]" },
    { id: "c", name: "[Unknown Album]", artist: "Flavor Foley" },
  ];
  const artists = [
    { id: "1", name: "Slowdive" },
    { id: "2", name: "[Unknown Artist]" },
  ];
  const tracks = [
    { id: "t1", artist: "Slowdive", album: "Souvlaki" },
    { id: "t2", artist: "[Unknown Artist]", album: "Afscheid" },
  ];

  it("returns the original lists when hiding is off", () => {
    expect(rejectUnknownAlbums(albums, false)).toBe(albums);
    expect(rejectUnknownArtists(artists, false)).toBe(artists);
    expect(rejectUnknownTracks(tracks, false)).toBe(tracks);
  });

  it("drops unknown albums, artists, tracks, and listen events when hiding is on", () => {
    expect(rejectUnknownAlbums(albums, true).map((item) => item.id)).toEqual([
      "a",
    ]);
    expect(rejectUnknownArtists(artists, true).map((item) => item.id)).toEqual([
      "1",
    ]);
    expect(rejectUnknownTracks(tracks, true).map((item) => item.id)).toEqual([
      "t1",
    ]);
    expect(
      rejectUnknownListenEvents(
        [
          { id: 1, artistName: "Slowdive", albumTitle: "Souvlaki" },
          { id: 2, artistName: "[Unknown Artist]", albumTitle: "Aikaintaite" },
        ],
        true,
      ).map((item) => item.id),
    ).toEqual([1]);
  });
});
