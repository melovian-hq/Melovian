// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { dedupeBy, listItemKey, stableItemKey } from "./collection";

describe("collection helpers", () => {
  it("dedupes items by key", () => {
    const items = [
      { id: "a", name: "One" },
      { id: "a", name: "One duplicate" },
      { id: "", name: "Two" },
      { id: "", name: "Two" },
    ];
    expect(
      dedupeBy(items, (item) => item.id || item.name.toLowerCase()),
    ).toEqual([
      { id: "a", name: "One" },
      { id: "", name: "Two" },
    ]);
  });

  it("builds stable keys with index fallback", () => {
    expect(stableItemKey("abc", 0)).toBe("abc");
    expect(stableItemKey("", 2, "artist")).toBe("artist-2");
    expect(stableItemKey(undefined, 4)).toBe("item-4");
  });

  it("builds unique list keys even when ids repeat", () => {
    expect(listItemKey("track-1", 0)).toBe("track-1#0");
    expect(listItemKey("track-1", 1)).toBe("track-1#1");
    expect(listItemKey("", 2, "track")).toBe("track-2#2");
  });
});
