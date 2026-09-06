// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  formatHumanDuration,
  playlistDurationSeconds,
} from "./playlist-duration";

describe("playlist-duration acceptance", () => {
  it("accepts summing track durations for playlist header", () => {
    const seconds = playlistDurationSeconds({
      tracks: [{ durationMs: 180_000 }, { durationMs: 240_000 }],
    });
    expect(seconds).toBe(420);
    expect(formatHumanDuration(seconds)).toBe("7 min");
  });
});
