// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createDetailCache } from "./detail-cache";

describe("createDetailCache", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns fresh entries and marks them stale after ttl", () => {
    const cache = createDetailCache<string>(1000);
    cache.set("a", "album");
    expect(cache.peek("a")).toEqual({ value: "album", stale: false });

    vi.advanceTimersByTime(1001);
    expect(cache.peek("a")).toEqual({ value: "album", stale: true });
  });

  it("deletes and clears entries", () => {
    const cache = createDetailCache<number>();
    cache.set("x", 1);
    cache.delete("x");
    expect(cache.peek("x")).toBeNull();
    cache.set("y", 2);
    cache.clear();
    expect(cache.peek("y")).toBeNull();
  });
});
