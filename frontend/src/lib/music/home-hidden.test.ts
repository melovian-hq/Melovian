// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, beforeEach } from "vitest";
import {
  emptyHiddenHome,
  filterHiddenById,
  hideHomeId,
  isHomeIdHidden,
  parseHiddenHome,
} from "./home-hidden.svelte";

describe("home hidden items", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("parses stored ids and ignores junk", () => {
    expect(parseHiddenHome(null)).toEqual(emptyHiddenHome());
    expect(parseHiddenHome("not-json")).toEqual(emptyHiddenHome());
    expect(
      parseHiddenHome(
        JSON.stringify({ albums: ["a1", "a1", ""], mixes: [1, "m1"] }),
      ),
    ).toEqual({
      albums: ["a1"],
      mixes: ["m1"],
      playlists: [],
      artists: [],
    });
  });

  it("hides an album id once", () => {
    const hidden = hideHomeId(emptyHiddenHome(), "album", "alb-1");
    expect(isHomeIdHidden(hidden, "album", "alb-1")).toBe(true);
    expect(hideHomeId(hidden, "album", "alb-1").albums).toEqual(["alb-1"]);
  });

  it("filters hidden albums out of a shelf", () => {
    const albums = [{ id: "keep" }, { id: "gone" }, { id: "also" }];
    expect(filterHiddenById(albums, new Set(["gone"]))).toEqual([
      { id: "keep" },
      { id: "also" },
    ]);
  });
});
