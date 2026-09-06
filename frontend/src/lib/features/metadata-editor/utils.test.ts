// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  buildLookupQuery,
  filenameSuggestionToMatch,
  formStateToUpdate,
  hasFilenameSuggestion,
  isMetadataFormDirty,
  nextTrackId,
  parseOptionalInt,
  trackToFormState,
} from "./utils";
import type { MetadataTrack } from "./types";

const sampleTrack: MetadataTrack = {
  id: "trk_1",
  title: "Alison",
  artist: "Slowdive",
  album: "Souvlaki",
  albumArtist: "Slowdive",
  trackNum: 3,
  discNum: 1,
  year: 1993,
  genre: "Shoegaze",
  durationMs: 0,
  format: "mp3",
  relPath: "Slowdive/Souvlaki/03 Alison.mp3",
  issues: [],
  coverArt: "trk_1",
};

describe("metadata editor utils", () => {
  it("detects dirty form state", () => {
    const form = trackToFormState(sampleTrack);
    expect(isMetadataFormDirty(sampleTrack, form)).toBe(false);
    expect(
      isMetadataFormDirty(sampleTrack, { ...form, title: "Changed" }),
    ).toBe(true);
  });

  it("builds lookup query from track fields", () => {
    expect(buildLookupQuery(sampleTrack)).toBe("Slowdive Alison Souvlaki");
    expect(buildLookupQuery(sampleTrack, "custom query")).toBe("custom query");
  });

  it("parses optional integers", () => {
    expect(parseOptionalInt("")).toBeUndefined();
    expect(parseOptionalInt("3")).toBe(3);
    expect(parseOptionalInt("nope")).toBeUndefined();
  });

  it("converts form state to update payload", () => {
    expect(
      formStateToUpdate({
        title: " Alison ",
        artist: "Slowdive",
        album: "Souvlaki",
        albumArtist: "",
        trackNum: "3",
        discNum: "",
        year: "1993",
        genre: "Shoegaze",
      }),
    ).toEqual({
      title: "Alison",
      artist: "Slowdive",
      album: "Souvlaki",
      albumArtist: "",
      trackNum: 3,
      discNum: undefined,
      year: 1993,
      genre: "Shoegaze",
    });
  });

  it("maps filename suggestions to lookup matches", () => {
    const match = filenameSuggestionToMatch({
      source: "filename",
      title: "Alison",
      artist: "Slowdive",
      album: "Souvlaki",
      albumArtist: "Slowdive",
      trackNum: 3,
      discNum: 1,
    });
    expect(match.id).toBe("filename");
    expect(match.title).toBe("Alison");
    expect(
      hasFilenameSuggestion({
        source: "filename",
        title: "Alison",
        artist: "Slowdive",
        album: "Souvlaki",
        albumArtist: "Slowdive",
        trackNum: 3,
        discNum: 1,
      }),
    ).toBe(true);
  });

  it("navigates tracks", () => {
    const tracks = [
      { ...sampleTrack, id: "a" },
      { ...sampleTrack, id: "b" },
      { ...sampleTrack, id: "c" },
    ];
    expect(nextTrackId(tracks, "b", 1)).toBe("c");
    expect(nextTrackId(tracks, "b", -1)).toBe("a");
  });
});
