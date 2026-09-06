// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { validateSmartPlaylistDraft } from "./validate";
import { createEmptySmartPlaylistDraft } from "./compile";

describe("validateSmartPlaylistDraft", () => {
  it("requires a playlist name", () => {
    const draft = createEmptySmartPlaylistDraft();
    draft.name = "  ";
    draft.root.rules[0]!.value = "Jazz";
    const errors = validateSmartPlaylistDraft(draft);
    expect(errors.some((error) => error.path === "name")).toBe(true);
  });

  it("validates numeric ranges", () => {
    const draft = createEmptySmartPlaylistDraft();
    draft.name = "Quality";
    draft.root.rules = [
      {
        id: "r1",
        field: "bitrate",
        operator: "inTheRange",
        value: [900, 320],
      },
    ];
    const errors = validateSmartPlaylistDraft(draft);
    expect(errors.some((error) => error.path.includes("rules[0]"))).toBe(true);
  });

  it("accepts a complete draft", () => {
    const draft = createEmptySmartPlaylistDraft();
    draft.name = "Recent jazz";
    draft.root.rules = [
      {
        id: "r1",
        field: "genre",
        operator: "contains",
        value: "Jazz",
      },
      {
        id: "r2",
        field: "lastplayed",
        operator: "inTheLast",
        value: 30,
      },
    ];
    expect(validateSmartPlaylistDraft(draft)).toEqual([]);
  });
});
