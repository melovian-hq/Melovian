// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  filterByLocalSearch,
  matchesLocalSearch,
  normalizeSearchQuery,
} from "./local-search";

describe("local-search", () => {
  it("normalizes case and diacritics", () => {
    expect(normalizeSearchQuery("  Synthwave  ")).toBe("synthwave");
    expect(normalizeSearchQuery("Björk")).toBe("bjork");
  });

  it("matches across fields", () => {
    expect(matchesLocalSearch("rock", "Pop", "Classic Rock")).toBe(true);
    expect(matchesLocalSearch("jazz", "Pop", "Classic Rock")).toBe(false);
    expect(matchesLocalSearch("", "anything")).toBe(true);
  });

  it("filters items by getter fields", () => {
    const items = [
      { name: "Morning Mix", artist: "DJ One" },
      { name: "Night Drive", artist: "DJ Two" },
    ];
    expect(
      filterByLocalSearch(items, "night", (item) => [item.name, item.artist]),
    ).toEqual([items[1]]);
    expect(filterByLocalSearch(items, "", (item) => [item.name])).toHaveLength(
      2,
    );
  });
});
