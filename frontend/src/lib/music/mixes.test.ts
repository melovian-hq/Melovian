// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { findMix } from "./mixes";

const mixes = [
  {
    id: "discover",
    title: "Discover Weekly",
    subtitle: "Fresh picks",
    tracks: [],
    gradient: "linear-gradient(red, blue)",
  },
  {
    id: "replay",
    title: "Replay Mix",
    subtitle: "On repeat",
    tracks: [],
    gradient: "linear-gradient(black, red)",
  },
];

describe("findMix", () => {
  it("finds mix by id", () => {
    expect(findMix(mixes, "discover")?.title).toBe("Discover Weekly");
  });

  it("returns undefined for empty or unknown ids", () => {
    expect(findMix(mixes, "")).toBeUndefined();
    expect(findMix(mixes, "missing")).toBeUndefined();
  });

  it("resolves legacy genre-mix and decade-mix ids to slotted mixes", () => {
    const slotted = [
      ...mixes,
      {
        id: "genre-mix-1",
        title: "Rock Mix",
        subtitle: "",
        tracks: [],
        gradient: "",
      },
      {
        id: "decade-mix-2",
        title: "2000s Mix",
        subtitle: "",
        tracks: [],
        gradient: "",
      },
    ];
    expect(findMix(slotted, "genre-mix")?.id).toBe("genre-mix-1");
    expect(findMix(slotted, "decade-mix")?.id).toBe("decade-mix-2");
  });
});
