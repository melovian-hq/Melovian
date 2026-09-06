// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { createBoundedMap, createBoundedSet } from "./bounded-cache";

describe("bounded-cache performance", () => {
  it("handles many map writes under 100ms", () => {
    const map = createBoundedMap<string, number>(256);
    const started = performance.now();
    for (let i = 0; i < 20_000; i++) {
      map.set(`k-${i}`, i);
    }
    const elapsed = performance.now() - started;
    expect(map.size).toBeLessThanOrEqual(256);
    expect(elapsed).toBeLessThan(100);
  });

  it("handles many set writes under 100ms", () => {
    const set = createBoundedSet<string>(128);
    const started = performance.now();
    for (let i = 0; i < 20_000; i++) {
      set.add(`v-${i}`);
    }
    const elapsed = performance.now() - started;
    expect(set.size).toBeLessThanOrEqual(128);
    expect(elapsed).toBeLessThan(100);
  });
});
