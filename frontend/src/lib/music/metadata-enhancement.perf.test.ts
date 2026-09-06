// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { coverArtFallbackUrl } from "./cover-art-fallback";
import {
  normalizeMetadataName,
  metadataNamesMatch,
} from "./metadata-enhancement";

describe("cover and metadata performance", () => {
  it("generates many fallback covers quickly", () => {
    const started = performance.now();
    const urls = new Set<string>();
    for (let i = 0; i < 2_000; i++) {
      urls.add(coverArtFallbackUrl(`seed-${i % 400}`, `artist-${i % 50}`));
    }
    const elapsed = performance.now() - started;
    expect(urls.size).toBeGreaterThan(50);
    expect(elapsed).toBeLessThan(250);
  });

  it("normalizes and matches names quickly", () => {
    const started = performance.now();
    let hits = 0;
    for (let i = 0; i < 5_000; i++) {
      const left = normalizeMetadataName(`Artist ${i % 100}`);
      const right = normalizeMetadataName(`artist ${i % 100}`);
      if (metadataNamesMatch(left, right)) hits++;
    }
    const elapsed = performance.now() - started;
    expect(hits).toBe(5_000);
    expect(elapsed).toBeLessThan(200);
  });
});
