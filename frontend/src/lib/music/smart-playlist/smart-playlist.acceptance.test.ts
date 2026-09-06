// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  compileSmartPlaylistDraft,
  createEmptySmartPlaylistDraft,
} from "./compile";
import { trackMatchesSmartPlaylist } from "./evaluate";
import type { SmartTrackContext } from "./context";
import { validateSmartPlaylistDraft } from "./validate";

function track(overrides: Partial<SmartTrackContext> = {}): SmartTrackContext {
  return {
    title: "Nightcall",
    artist: "Kavinsky",
    album: "OutRun",
    albumartist: "Kavinsky",
    genre: "Synthwave",
    language: "en",
    comment: "",
    filepath: "/library/nightcall.flac",
    codec: "flac",
    filetype: "flac",
    year: 2010,
    date: "2010-04-01",
    dateadded: "2025-01-01",
    releasedate: "2010-04-01",
    lastplayed: "2026-06-01",
    bitrate: 850,
    bitdepth: 16,
    samplerate: 44100,
    duration: 257,
    bpm: 108,
    rating: 4,
    playcount: 8,
    loved: true,
    compilation: false,
    missing: false,
    ...overrides,
  };
}

describe("smart-playlist acceptance", () => {
  it("accepts create-validate-compile journey for a genre filter", () => {
    const draft = createEmptySmartPlaylistDraft();
    draft.name = "Synthwave Nights";
    draft.comment = "Weekend driving";
    draft.root.rules[0] = {
      id: "rule-genre",
      field: "genre",
      operator: "contains",
      value: "Synth",
    };
    draft.limit = 50;

    const errors = validateSmartPlaylistDraft(draft);
    expect(errors).toEqual([]);

    const payload = compileSmartPlaylistDraft(draft);
    expect(payload.name).toBe("Synthwave Nights");
    expect(payload.comment).toBe("Weekend driving");
    expect(payload.rules.limit).toBe(50);
    expect(payload.rules.all).toEqual([{ contains: { genre: "Synth" } }]);
  });

  it("accepts local evaluation of compiled intent against library tracks", () => {
    const draft = createEmptySmartPlaylistDraft();
    draft.name = "Loved Flac";
    draft.root.logic = "all";
    draft.root.rules = [
      { id: "r1", field: "loved", operator: "is", value: true },
      { id: "r2", field: "filetype", operator: "is", value: "flac" },
    ];

    expect(validateSmartPlaylistDraft(draft)).toEqual([]);
    expect(trackMatchesSmartPlaylist(track(), draft.root)).toBe(true);
    expect(trackMatchesSmartPlaylist(track({ loved: false }), draft.root)).toBe(
      false,
    );
    expect(
      trackMatchesSmartPlaylist(
        track({ filetype: "mp3", codec: "mp3" }),
        draft.root,
      ),
    ).toBe(false);
  });

  it("accepts nested any-group inside all root", () => {
    const draft = createEmptySmartPlaylistDraft();
    draft.name = "Nested";
    draft.root = {
      id: "root",
      logic: "all",
      rules: [{ id: "r1", field: "year", operator: "gt", value: 2000 }],
      groups: [
        {
          id: "g1",
          logic: "any",
          rules: [
            { id: "r2", field: "artist", operator: "is", value: "Kavinsky" },
            { id: "r3", field: "artist", operator: "is", value: "M83" },
          ],
          groups: [],
        },
      ],
    };

    expect(validateSmartPlaylistDraft(draft)).toEqual([]);
    expect(trackMatchesSmartPlaylist(track(), draft.root)).toBe(true);
    expect(
      trackMatchesSmartPlaylist(
        track({ artist: "Other", year: 2015 }),
        draft.root,
      ),
    ).toBe(false);
  });
});
