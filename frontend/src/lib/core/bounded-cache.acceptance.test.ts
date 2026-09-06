// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { createBoundedMap, createBoundedSet } from "./bounded-cache";

describe("bounded-cache acceptance", () => {
  it("accepts cover-art cache staying bounded during long browse sessions", () => {
    const cache = createBoundedMap<string, string>(3);
    for (let i = 0; i < 20; i++) {
      cache.set(`cover-${i}`, `blob://${i}`);
    }
    expect(cache.size).toBe(3);
    expect(cache.has("cover-19")).toBe(true);
    expect(cache.has("cover-18")).toBe(true);
    expect(cache.has("cover-17")).toBe(true);
    expect(cache.has("cover-0")).toBe(false);
  });

  it("accepts recent-search set eviction for command palette history", () => {
    const recent = createBoundedSet<string>(3);
    recent.add("radiohead");
    recent.add("daft punk");
    recent.add("pink floyd");
    recent.add("bjork");
    expect([...recent]).toEqual(["daft punk", "pink floyd", "bjork"]);
  });
});
