// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import { createBoundedMap, createBoundedSet } from "./bounded-cache";

describe("bounded-cache memory properties", () => {
  it("map never exceeds capacity", () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 1, max: 20 }),
        fc.array(fc.tuple(fc.string(), fc.integer()), { maxLength: 80 }),
        (capacity, writes) => {
          const map = createBoundedMap<string, number>(capacity);
          for (const [key, value] of writes) {
            map.set(key, value);
          }
          expect(map.size).toBeLessThanOrEqual(capacity);
        },
      ),
    );
  });

  it("set never exceeds capacity", () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 1, max: 20 }),
        fc.array(fc.string(), { maxLength: 80 }),
        (capacity, values) => {
          const set = createBoundedSet<string>(capacity);
          for (const value of values) {
            set.add(value);
          }
          expect(set.size).toBeLessThanOrEqual(capacity);
        },
      ),
    );
  });
});
