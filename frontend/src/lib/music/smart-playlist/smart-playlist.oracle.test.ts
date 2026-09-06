// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { compileSmartPlaylistDraft } from "./compile";
import { trackMatchesSmartPlaylist } from "./evaluate";
import type { SmartPlaylistDraft } from "./types";
import type { SmartTrackContext } from "./context";
import { validateSmartPlaylistDraft } from "./validate";

function baseTrack(
  overrides: Partial<SmartTrackContext> = {},
): SmartTrackContext {
  return {
    title: "Midnight City",
    artist: "M83",
    album: "Hurry Up We're Dreaming",
    albumartist: "M83",
    genre: "Electronic",
    language: "en",
    comment: "",
    filepath: "/music/m83.flac",
    codec: "flac",
    filetype: "flac",
    year: 2011,
    date: "2011-10-18",
    dateadded: "2024-01-01",
    releasedate: "2011-10-18",
    lastplayed: "2026-07-01",
    bitrate: 900,
    bitdepth: 16,
    samplerate: 44100,
    duration: 241,
    bpm: 104,
    rating: 5,
    playcount: 12,
    loved: true,
    compilation: false,
    missing: false,
    ...overrides,
  };
}

function draftWithRule(
  field: string,
  operator: string,
  value: string | number | boolean | [number, number] | [string, string],
  logic: "all" | "any" = "all",
): SmartPlaylistDraft {
  return {
    name: "Oracle Mix",
    comment: "oracle",
    public: false,
    sort: "+title",
    limit: 100,
    limitPercent: null,
    root: {
      id: "root",
      logic,
      rules: [{ id: "r1", field, operator, value }],
      groups: [],
    },
  };
}

describe("smart-playlist oracle", () => {
  it("valid drafts compile to payload with trimmed name and rules", () => {
    const draft = draftWithRule("genre", "contains", "Electro");
    expect(validateSmartPlaylistDraft(draft)).toEqual([]);
    const payload = compileSmartPlaylistDraft(draft);
    expect(payload.name).toBe("Oracle Mix");
    expect(payload.rules.all).toBeDefined();
    expect(payload.rules.limit).toBe(100);
  });

  it("compile rejects invalid drafts", () => {
    const draft = draftWithRule("title", "contains", "");
    expect(validateSmartPlaylistDraft(draft).length).toBeGreaterThan(0);
    expect(() => compileSmartPlaylistDraft(draft)).toThrow();
  });

  it("evaluate all-logic requires every rule", () => {
    const draft = draftWithRule("artist", "is", "M83");
    draft.root.rules.push({
      id: "r2",
      field: "year",
      operator: "gt",
      value: 2010,
    });
    expect(trackMatchesSmartPlaylist(baseTrack(), draft.root)).toBe(true);
    expect(
      trackMatchesSmartPlaylist(baseTrack({ year: 2009 }), draft.root),
    ).toBe(false);
  });

  it("evaluate any-logic accepts first matching rule", () => {
    const draft = draftWithRule("artist", "is", "Nobody", "any");
    draft.root.rules.push({
      id: "r2",
      field: "genre",
      operator: "contains",
      value: "Electro",
    });
    expect(trackMatchesSmartPlaylist(baseTrack(), draft.root)).toBe(true);
  });

  it("isMissing and isPresent are complementary for empty fields", () => {
    const missingDraft = draftWithRule("language", "isMissing", true);
    const presentDraft = draftWithRule("language", "isPresent", true);
    const empty = baseTrack({ language: "" });
    const filled = baseTrack({ language: "en" });
    expect(trackMatchesSmartPlaylist(empty, missingDraft.root)).toBe(true);
    expect(trackMatchesSmartPlaylist(filled, missingDraft.root)).toBe(false);
    expect(trackMatchesSmartPlaylist(empty, presentDraft.root)).toBe(false);
    expect(trackMatchesSmartPlaylist(filled, presentDraft.root)).toBe(true);
  });

  it("numeric range oracle is inclusive", () => {
    const draft = draftWithRule("year", "inTheRange", [2010, 2012]);
    expect(
      trackMatchesSmartPlaylist(baseTrack({ year: 2010 }), draft.root),
    ).toBe(true);
    expect(
      trackMatchesSmartPlaylist(baseTrack({ year: 2012 }), draft.root),
    ).toBe(true);
    expect(
      trackMatchesSmartPlaylist(baseTrack({ year: 2009 }), draft.root),
    ).toBe(false);
  });

  it("before and after compare date strings", () => {
    const before = draftWithRule("releasedate", "before", "2012-01-01");
    const after = draftWithRule("releasedate", "after", "2010-01-01");
    expect(trackMatchesSmartPlaylist(baseTrack(), before.root)).toBe(true);
    expect(trackMatchesSmartPlaylist(baseTrack(), after.root)).toBe(true);
    expect(
      trackMatchesSmartPlaylist(
        baseTrack({ releasedate: "2015-01-01" }),
        before.root,
      ),
    ).toBe(false);
  });

  it("lastplayed inTheLast uses date elapsed days", () => {
    const draft = draftWithRule("lastplayed", "inTheLast", 30);
    const recent = baseTrack({
      lastplayed: new Date(Date.now() - 2 * 86_400_000).toISOString(),
    });
    const old = baseTrack({
      lastplayed: new Date(Date.now() - 90 * 86_400_000).toISOString(),
    });
    expect(trackMatchesSmartPlaylist(recent, draft.root)).toBe(true);
    expect(trackMatchesSmartPlaylist(old, draft.root)).toBe(false);
    expect(validateSmartPlaylistDraft(draft)).toEqual([]);
  });
});
