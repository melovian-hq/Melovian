// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { bestMatches, normalizeTerm } from "./search";

describe("search acceptance", () => {
  it("accepts did-you-mean ranking for a typo against a library", () => {
    const artists = [
      "Radiohead",
      "Radionow",
      "Pink Floyd",
      "The Beatles",
      "Radio Free",
    ];
    const matches = bestMatches("radiohed", artists, (item) => item, {
      limit: 3,
      threshold: 0.2,
    });
    expect(matches.length).toBeGreaterThan(0);
    expect(matches[0]?.item).toBe("Radiohead");
    for (let i = 1; i < matches.length; i++) {
      expect(matches[0]!.score).toBeGreaterThanOrEqual(matches[i]!.score);
    }
    if (matches.length > 1) {
      expect(matches[0]!.score).toBeGreaterThan(matches[1]!.score);
    }
  });

  it("accepts case and accent insensitive lookup", () => {
    expect(normalizeTerm("Café")).toBe("cafe");
    const albums = ["Café del Mar", "Cafe Society", "Night Drive"];
    const matches = bestMatches("cafe", albums, (item) => item, {
      limit: 5,
      threshold: 0.2,
    });
    expect(matches.map((m) => m.item)).toContain("Café del Mar");
  });
});
