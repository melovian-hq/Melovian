// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { filterByLocalSearch } from "$lib/utils/local-search";
import { matchRoute } from "$lib/router/match";
import { formatRelativePlayedAt } from "$lib/music/relative-time";

describe("browse-library acceptance", () => {
  it("accepts filtering artists then opening a detail route", () => {
    const artists = [
      { id: "1", name: "Radiohead" },
      { id: "2", name: "Daft Punk" },
      { id: "3", name: "Bjork" },
    ];
    const filtered = filterByLocalSearch(artists, "daft", (a) => [a.name]);
    expect(filtered).toEqual([{ id: "2", name: "Daft Punk" }]);

    const route = matchRoute(
      [{ path: "/artist/:id" }, { path: "/album/:id" }],
      "/artist/2",
      "?from=search",
    );
    expect(route).toEqual({
      path: "/artist/:id",
      match: { params: { id: "2" }, query: { from: "search" } },
    });
  });

  it("accepts recently played labels in history UI", () => {
    const now = Date.UTC(2026, 6, 18, 15, 0, 0);
    expect(
      formatRelativePlayedAt(new Date(now - 5 * 60_000).toISOString(), now),
    ).toBe("5 min ago");
    expect(
      formatRelativePlayedAt(new Date(now - 2 * 3_600_000).toISOString(), now),
    ).toBe("2 hours ago");
  });
});
