// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import {
  compileSmartPlaylistDraft,
  createEmptySmartPlaylistDraft,
} from "./compile";
import { trackMatchesSmartPlaylist } from "./evaluate";
import { SMART_PLAYLIST_FIELDS } from "./fields";
import type { SmartPlaylistDraft, SmartPlaylistRule } from "./types";
import type { SmartTrackContext } from "./context";
import { validateSmartPlaylistDraft } from "./validate";

function baseTrack(
  overrides: Partial<SmartTrackContext> = {},
): SmartTrackContext {
  return {
    title: "Song",
    artist: "Artist",
    album: "Album",
    albumartist: "Artist",
    genre: "Rock",
    language: "en",
    comment: "",
    filepath: "/music/song.flac",
    codec: "flac",
    filetype: "flac",
    year: 2020,
    date: "2020-01-01",
    dateadded: "2020-01-02",
    releasedate: "2020-01-01",
    lastplayed: "2026-01-01",
    bitrate: 1000,
    bitdepth: 16,
    samplerate: 44100,
    duration: 200,
    bpm: 120,
    rating: 5,
    playcount: 3,
    loved: true,
    compilation: false,
    missing: false,
    ...overrides,
  };
}

describe("smart-playlist exploratory", () => {
  it("explores empty drafts always fail validation", () => {
    fc.assert(
      fc.property(fc.string({ maxLength: 40 }), (name) => {
        const draft = createEmptySmartPlaylistDraft();
        draft.name = name;
        const errors = validateSmartPlaylistDraft(draft);
        if (!name.trim()) {
          expect(errors.some((e) => e.path === "name")).toBe(true);
        }
        expect(errors.some((e) => e.path.startsWith("root"))).toBe(true);
      }),
      { numRuns: 40 },
    );
  });

  it("explores contains matching across random titles", () => {
    fc.assert(
      fc.property(
        fc.string({ minLength: 1, maxLength: 20 }),
        fc.string({ minLength: 0, maxLength: 12 }),
        (title, needle) => {
          const rule: SmartPlaylistRule = {
            id: "r1",
            field: "title",
            operator: "contains",
            value: needle,
          };
          const root = {
            id: "g1",
            logic: "all" as const,
            rules: [rule],
            groups: [],
          };
          const matched = trackMatchesSmartPlaylist(baseTrack({ title }), root);
          const wrapped = String(needle).match(/^\/(.+)\/([gimsuy]*)$/);
          let expected: boolean;
          if (wrapped) {
            try {
              expected = new RegExp(wrapped[1] ?? needle, "i").test(title);
            } catch {
              expected = title
                .toLowerCase()
                .includes((wrapped[1] ?? needle).toLowerCase());
            }
          } else {
            expected = title
              .toLowerCase()
              .includes(String(needle).toLowerCase());
          }
          expect(matched).toBe(expected);
        },
      ),
      { numRuns: 80 },
    );
  });

  it("explores unknown fields never validate", () => {
    const fields = new Set(SMART_PLAYLIST_FIELDS.map((f) => f.id));
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 16 }), (field) => {
        fc.pre(!fields.has(field));
        const draft: SmartPlaylistDraft = {
          name: "Test",
          comment: "",
          public: false,
          sort: "+title",
          limit: 50,
          limitPercent: null,
          root: {
            id: "g1",
            logic: "all",
            rules: [{ id: "r1", field, operator: "is", value: "x" }],
            groups: [],
          },
        };
        const errors = validateSmartPlaylistDraft(draft);
        expect(errors.length).toBeGreaterThan(0);
        expect(() => compileSmartPlaylistDraft(draft)).toThrow();
      }),
      { numRuns: 40 },
    );
  });
});
