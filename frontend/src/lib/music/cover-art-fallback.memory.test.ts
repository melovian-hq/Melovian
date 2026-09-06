// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { coverArtFallbackUrl } from "./cover-art-fallback";

describe("cover-art-fallback memory", () => {
  it("reuses cached urls instead of unbounded growth", () => {
    const samples: string[] = [];
    for (let i = 0; i < 5_000; i++) {
      samples.push(coverArtFallbackUrl("same-seed", "same-palette"));
    }
    const unique = new Set(samples);
    expect(unique.size).toBe(1);
  });
});
