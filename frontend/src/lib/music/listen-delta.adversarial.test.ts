// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";

/** Mirrors client delta clamp used before PUT /api/music/items. */
function clientListenDelta(prevMs: number, nextMs: number): number {
  return Math.max(0, Math.min(15_000, nextMs - prevMs));
}

describe("listen delta adversarial property", () => {
  it("never reports negative or oversized deltas", () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 0, max: 600_000 }),
        fc.integer({ min: -100_000, max: 600_000 }),
        (prev, next) => {
          const delta = clientListenDelta(prev, next);
          expect(delta).toBeGreaterThanOrEqual(0);
          expect(delta).toBeLessThanOrEqual(15_000);
          if (next <= prev) expect(delta).toBe(0);
        },
      ),
      { numRuns: 200 },
    );
  });

  it("seek backwards never invents listen time", () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 10_000, max: 300_000 }),
        fc.integer({ min: 1, max: 9_999 }),
        (prev, next) => {
          expect(clientListenDelta(prev, next)).toBe(0);
        },
      ),
      { numRuns: 80 },
    );
  });

  it("forward scrub jumps are capped like seeks", () => {
    expect(clientListenDelta(0, 120_000)).toBe(15_000);
    expect(clientListenDelta(5_000, 12_000)).toBe(7_000);
  });
});
