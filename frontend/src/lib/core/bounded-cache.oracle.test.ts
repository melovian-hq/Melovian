// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import { createBoundedMap, createBoundedSet } from "./bounded-cache";

describe("bounded-cache exploratory", () => {
  it("explores map never exceeds capacity under random writes", () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 1, max: 20 }),
        fc.array(fc.tuple(fc.string({ maxLength: 8 }), fc.integer()), {
          maxLength: 80,
        }),
        (capacity, ops) => {
          const map = createBoundedMap<string, number>(capacity);
          for (const [key, value] of ops) {
            map.set(key, value);
            expect(map.size).toBeLessThanOrEqual(capacity);
          }
        },
      ),
      { numRuns: 80 },
    );
  });

  it("explores set never exceeds capacity under random adds", () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 1, max: 20 }),
        fc.array(fc.string({ maxLength: 10 }), { maxLength: 80 }),
        (capacity, values) => {
          const set = createBoundedSet<string>(capacity);
          for (const value of values) {
            set.add(value);
            expect(set.size).toBeLessThanOrEqual(capacity);
          }
        },
      ),
      { numRuns: 80 },
    );
  });
});

describe("bounded-cache oracle", () => {
  it("get refreshes LRU order so recently read keys survive eviction", () => {
    const map = createBoundedMap<string, number>(2);
    map.set("a", 1);
    map.set("b", 2);
    expect(map.get("a")).toBe(1);
    map.set("c", 3);
    expect(map.has("a")).toBe(true);
    expect(map.has("b")).toBe(false);
    expect(map.get("c")).toBe(3);
  });

  it("re-adding an existing set member refreshes eviction order", () => {
    const set = createBoundedSet<string>(2);
    set.add("a");
    set.add("b");
    set.add("a");
    set.add("c");
    expect(set.has("a")).toBe(true);
    expect(set.has("b")).toBe(false);
    expect(set.has("c")).toBe(true);
  });
});
