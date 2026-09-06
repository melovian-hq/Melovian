// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  resolveTrackAlbum,
  resolveTrackArtists,
  splitArtistNames,
  trackHasFullMetadata,
  trackNeedsLinkMetadata,
} from "./track-metadata";

describe("splitArtistNames", () => {
  it("splits common collaboration separators", () => {
    expect(splitArtistNames("OG Buda & Artist2")).toEqual([
      "OG Buda",
      "Artist2",
    ]);
    expect(splitArtistNames("Drake feat. Travis Scott")).toEqual([
      "Drake",
      "Travis Scott",
    ]);
    expect(splitArtistNames("A, B; C")).toEqual(["A", "B", "C"]);
    expect(splitArtistNames("Alpha x Beta")).toEqual(["Alpha", "Beta"]);
  });

  it("keeps single artist names intact", () => {
    expect(splitArtistNames("OG Buda")).toEqual(["OG Buda"]);
  });
});

describe("trackHasFullMetadata", () => {
  it("returns true when title, artist, and duration are present", () => {
    expect(
      trackHasFullMetadata({
        id: "1",
        title: "Track",
        artist: "Artist",
        duration: 240,
      }),
    ).toBe(true);
  });

  it("returns false when metadata is incomplete", () => {
    expect(
      trackHasFullMetadata({ id: "1", title: "Track", artist: "Artist" }),
    ).toBe(false);
    expect(
      trackHasFullMetadata({ id: "1", title: "Track", duration: 240 }),
    ).toBe(false);
  });
});

describe("trackNeedsLinkMetadata", () => {
  it("requires link ids when display names exist", () => {
    expect(
      trackNeedsLinkMetadata({
        id: "1",
        title: "Track",
        artist: "OG Buda",
        album: "Album",
        duration: 240,
      }),
    ).toBe(true);
    expect(
      trackNeedsLinkMetadata({
        id: "1",
        title: "Track",
        artist: "OG Buda",
        artistId: "art_1",
        album: "Album",
        albumId: "alb_1",
        duration: 240,
      }),
    ).toBe(false);
  });

  it("requests enrichment for collaborations without structured artists", () => {
    expect(
      trackNeedsLinkMetadata({
        id: "1",
        title: "Track",
        artist: "OG Buda & Artist2",
        artistId: "art_collab",
        albumId: "alb_1",
        duration: 240,
      }),
    ).toBe(true);
    expect(
      trackNeedsLinkMetadata({
        id: "1",
        title: "Track",
        artist: "OG Buda & Artist2",
        artistId: "art_collab",
        albumId: "alb_1",
        duration: 240,
        artists: [
          { id: "art_1", name: "OG Buda" },
          { id: "art_2", name: "Artist2" },
        ],
      }),
    ).toBe(false);
  });
});

describe("resolveTrackArtists", () => {
  it("uses structured artists when available", () => {
    expect(
      resolveTrackArtists({
        id: "1",
        title: "Track",
        artist: "OG Buda & Artist2",
        artists: [
          { id: "art_1", name: "OG Buda" },
          { id: "art_2", name: "Artist2" },
        ],
      }),
    ).toEqual([
      { id: "art_1", name: "OG Buda" },
      { id: "art_2", name: "Artist2" },
    ]);
  });

  it("splits collaborations into separate local artist links", () => {
    expect(
      resolveTrackArtists(
        {
          id: "1",
          title: "Track",
          artist: "OG Buda & Artist2",
        },
        { localIds: true },
      ),
    ).toEqual([
      { id: "art_6e8c05a9f72985e1", name: "OG Buda" },
      { id: "art_d4d4b6292e0d2ae8", name: "Artist2" },
    ]);
  });

  it("keeps a single subsonic artist id on solo tracks", () => {
    expect(
      resolveTrackArtists({
        id: "1",
        title: "Track",
        artist: "OG Buda",
        artistId: "server-artist-1",
      }),
    ).toEqual([{ id: "server-artist-1", name: "OG Buda" }]);
  });
});

describe("resolveTrackAlbum", () => {
  it("returns album credit when album title exists", () => {
    expect(
      resolveTrackAlbum({
        id: "1",
        title: "Track",
        album: "My Album",
        albumId: "alb_1",
      }),
    ).toEqual({ id: "alb_1", name: "My Album" });
  });

  it("returns null when album title is missing", () => {
    expect(resolveTrackAlbum({ id: "1", title: "Track" })).toBeNull();
  });
});
