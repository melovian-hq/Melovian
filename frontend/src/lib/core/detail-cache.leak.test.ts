// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import { createDetailCache } from "./detail-cache";

describe("createDetailCache memory properties", () => {
  it("never stores more entries than set ids", () => {
    fc.assert(
      fc.property(
        fc.uniqueArray(fc.tuple(fc.string(), fc.string()), {
          selector: ([id]) => id,
          maxLength: 50,
        }),
        (entries) => {
          const cache = createDetailCache<string>();
          for (const [id, value] of entries) {
            cache.set(id, value);
          }
          let stored = 0;
          for (const [id] of entries) {
            if (cache.peek(id)) stored++;
          }
          expect(stored).toBe(entries.length);
        },
      ),
    );
  });
});
