// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import { dedupeBy, stableItemKey } from "./collection";

describe("dedupeBy properties", () => {
  it("never increases length", () => {
    fc.assert(
      fc.property(
        fc.array(
          fc.record({
            id: fc.string(),
            name: fc.string(),
          }),
        ),
        (items) => {
          const result = dedupeBy(
            items,
            (item) => item.id || item.name.toLowerCase(),
          );
          expect(result.length).toBeLessThanOrEqual(items.length);
        },
      ),
    );
  });

  it("preserves first occurrence order and removes duplicate keys", () => {
    fc.assert(
      fc.property(
        fc.array(
          fc.record({
            id: fc.string(),
            name: fc.string(),
          }),
        ),
        (items) => {
          const result = dedupeBy(
            items,
            (item) => item.id || item.name.toLowerCase(),
          );
          const seen = new Set<string>();
          for (const item of result) {
            const key = item.id || item.name.toLowerCase();
            expect(seen.has(key)).toBe(false);
            seen.add(key);
          }
          for (const item of result) {
            const key = item.id || item.name.toLowerCase();
            expect(
              items.find(
                (entry) => (entry.id || entry.name.toLowerCase()) === key,
              ),
            ).toEqual(item);
          }
        },
      ),
    );
  });
});

describe("stableItemKey properties", () => {
  it("returns non-empty keys", () => {
    fc.assert(
      fc.property(fc.string(), fc.nat(), fc.string(), (id, index, fallback) => {
        const key = stableItemKey(id, index, fallback || "item");
        expect(key.length).toBeGreaterThan(0);
      }),
    );
  });

  it("prefers non-empty ids", () => {
    fc.assert(
      fc.property(fc.string({ minLength: 1 }), fc.nat(), (id, index) => {
        expect(stableItemKey(id, index)).toBe(id);
      }),
    );
  });

  it("uses index fallback when id is empty", () => {
    fc.assert(
      fc.property(fc.nat(), fc.string({ minLength: 1 }), (index, fallback) => {
        expect(stableItemKey("", index, fallback)).toBe(`${fallback}-${index}`);
      }),
    );
  });
});
