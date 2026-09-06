// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  formatHumanDuration,
  formatPlaylistDuration,
  formatPlaylistDurationClock,
  playlistDurationSeconds,
} from "./playlist-duration";

describe("playlist duration helpers", () => {
  it("prefers durationMs when present", () => {
    expect(playlistDurationSeconds({ durationMs: 125_000 })).toBe(125);
    expect(formatPlaylistDuration({ durationMs: 125_000 })).toBe("3 min");
    expect(formatPlaylistDurationClock({ durationMs: 125_000 })).toBe("2:05");
  });

  it("falls back to server duration seconds", () => {
    expect(playlistDurationSeconds({ duration: 3600 })).toBe(3600);
    expect(formatPlaylistDuration({ duration: 90 })).toBe("2 min");
    expect(formatPlaylistDuration({ duration: 3600 })).toBe("1 hr");
    expect(formatPlaylistDuration({ duration: 3660 })).toBe("1 hr 1 min");
  });

  it("formats short and long totals in human-readable units", () => {
    expect(formatHumanDuration(45)).toBe("1 min");
    expect(formatHumanDuration(3600)).toBe("1 hr");
    expect(formatHumanDuration(5400)).toBe("1 hr 30 min");
    expect(formatHumanDuration(10_800)).toBe("3 hr");
  });

  it("sums track durations when needed", () => {
    expect(
      playlistDurationSeconds({
        tracks: [{ durationMs: 60_000 }, { durationMs: 30_000 }],
      }),
    ).toBe(90);
    expect(
      formatPlaylistDuration({
        tracks: [{ durationMs: 60_000 }, { durationMs: 30_000 }],
      }),
    ).toBe("2 min");
  });
});
