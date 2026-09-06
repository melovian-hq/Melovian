// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  compileSmartPlaylistDraft,
  createEmptySmartPlaylistDraft,
} from "./compile";
import type { SmartPlaylistDraft } from "./types";

describe("compileSmartPlaylistDraft", () => {
  it("compiles AND/OR groups into Navidrome rules", () => {
    const draft: SmartPlaylistDraft = {
      name: "80s favorites",
      comment: "",
      public: true,
      sort: "-year",
      limit: 50,
      limitPercent: null,
      root: {
        id: "g1",
        logic: "all",
        rules: [
          {
            id: "r1",
            field: "genre",
            operator: "contains",
            value: "Rock",
          },
        ],
        groups: [
          {
            id: "g2",
            logic: "any",
            rules: [
              {
                id: "r2",
                field: "artist",
                operator: "contains",
                value: "Queen",
              },
              {
                id: "r3",
                field: "artist",
                operator: "contains",
                value: "Duran Duran",
              },
            ],
            groups: [],
          },
        ],
      },
    };

    const payload = compileSmartPlaylistDraft(draft);
    expect(payload.name).toBe("80s favorites");
    expect(payload.public).toBe(true);
    expect(payload.rules.sort).toBe("-year");
    expect(payload.rules.limit).toBe(50);
    expect(payload.rules.all).toEqual([
      { contains: { genre: "Rock" } },
      {
        any: [
          { contains: { artist: "Queen" } },
          { contains: { artist: "Duran Duran" } },
        ],
      },
    ]);
  });

  it("rejects invalid drafts", () => {
    const draft = createEmptySmartPlaylistDraft();
    draft.name = "";
    expect(() => compileSmartPlaylistDraft(draft)).toThrow(
      "Playlist name is required",
    );
  });

  it("strips regex delimiters for contains rules", () => {
    const draft = createEmptySmartPlaylistDraft();
    draft.name = "Regex mix";
    draft.root.rules = [
      {
        id: "r1",
        field: "title",
        operator: "contains",
        value: "/live$/i",
      },
    ];
    const payload = compileSmartPlaylistDraft(draft);
    expect(payload.rules.all).toEqual([{ contains: { title: "live$" } }]);
  });
});
