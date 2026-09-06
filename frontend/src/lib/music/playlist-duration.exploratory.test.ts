// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import {
  formatHumanDuration,
  playlistDurationSeconds,
} from "./playlist-duration";

describe("playlist-duration exploratory", () => {
  it("explores duration never negative and prefers durationMs", () => {
    fc.assert(
      fc.property(
        fc.nat({ max: 3_600_000 }),
        fc.nat({ max: 3600 }),
        (durationMs, duration) => {
          const seconds = playlistDurationSeconds({ durationMs, duration });
          expect(seconds).toBeGreaterThanOrEqual(0);
          if (durationMs > 0) {
            expect(seconds).toBe(Math.floor(durationMs / 1000));
          }
        },
      ),
      { numRuns: 80 },
    );
  });

  it("formatHumanDuration returns null for non-positive input", () => {
    expect(formatHumanDuration(0)).toBeNull();
    expect(formatHumanDuration(-5)).toBeNull();
  });
});
