// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { createBoundedMap, createBoundedSet } from "./bounded-cache";

describe("createBoundedMap", () => {
  it("evicts least recently used entries", () => {
    const map = createBoundedMap<string, number>(2);
    map.set("a", 1);
    map.set("b", 2);
    map.set("c", 3);
    expect(map.has("a")).toBe(false);
    expect(map.get("b")).toBe(2);
    expect(map.get("c")).toBe(3);
  });

  it("refresh on get delays eviction", () => {
    const map = createBoundedMap<string, number>(2);
    map.set("a", 1);
    map.set("b", 2);
    expect(map.get("a")).toBe(1);
    map.set("c", 3);
    expect(map.has("a")).toBe(true);
    expect(map.has("b")).toBe(false);
  });
});

describe("createBoundedSet", () => {
  it("caps size and keeps newest", () => {
    const set = createBoundedSet<string>(2);
    set.add("a");
    set.add("b");
    set.add("c");
    expect(set.has("a")).toBe(false);
    expect(set.has("b")).toBe(true);
    expect(set.has("c")).toBe(true);
  });
});
