// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fc from "fast-check";
import { createDetailCache } from "./detail-cache";

describe("createDetailCache properties", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("peek returns null for unknown ids", () => {
    fc.assert(
      fc.property(fc.string(), (id) => {
        const cache = createDetailCache<string>();
        expect(cache.peek(id)).toBeNull();
      }),
    );
  });

  it("set then peek returns fresh entry before ttl expires", () => {
    fc.assert(
      fc.property(fc.string(), fc.string(), (id, value) => {
        const cache = createDetailCache<string>(1000);
        cache.set(id, value);
        expect(cache.peek(id)).toEqual({ value, stale: false });
      }),
    );
  });

  it("marks entries stale after ttl", () => {
    fc.assert(
      fc.property(
        fc.string(),
        fc.string(),
        fc.integer({ min: 1, max: 60_000 }),
        (id, value, ttl) => {
          const cache = createDetailCache<string>(ttl);
          cache.set(id, value);
          vi.advanceTimersByTime(ttl + 1);
          expect(cache.peek(id)).toEqual({ value, stale: true });
        },
      ),
    );
  });

  it("delete and clear remove entries", () => {
    fc.assert(
      fc.property(
        fc.array(fc.tuple(fc.string(), fc.string()), { maxLength: 20 }),
        (entries) => {
          const cache = createDetailCache<string>();
          for (const [id, value] of entries) {
            cache.set(id, value);
          }
          for (const [id] of entries) {
            cache.delete(id);
            expect(cache.peek(id)).toBeNull();
          }
          for (const [id, value] of entries) {
            cache.set(id, value);
          }
          cache.clear();
          for (const [id] of entries) {
            expect(cache.peek(id)).toBeNull();
          }
        },
      ),
    );
  });
});
