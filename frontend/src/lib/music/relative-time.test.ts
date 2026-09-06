// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  formatExactPlayedAt,
  formatPlayedClock,
  formatRelativePlayedAt,
} from "./relative-time";

describe("relative-time", () => {
  const now = Date.UTC(2026, 6, 3, 12, 0, 0);

  it("formats recent plays", () => {
    expect(formatRelativePlayedAt("2026-07-03T11:59:30.000Z", now)).toBe(
      "Just now",
    );
    expect(formatRelativePlayedAt("2026-07-03T11:30:00.000Z", now)).toBe(
      "30 min ago",
    );
    expect(formatRelativePlayedAt("2026-07-03T09:00:00.000Z", now)).toBe(
      "3 hours ago",
    );
  });

  it("formats day month and year boundaries", () => {
    expect(formatRelativePlayedAt("2026-07-01T12:00:00.000Z", now)).toBe(
      "2 days ago",
    );
    expect(formatRelativePlayedAt("2026-06-03T12:00:00.000Z", now)).toBe(
      "1 month ago",
    );
    expect(formatRelativePlayedAt("2025-07-03T12:00:00.000Z", now)).toBe(
      "1 year ago",
    );
  });

  it("formats clock and exact timestamps", () => {
    const iso = "2026-07-03T15:45:00.000Z";
    expect(formatPlayedClock(iso)).toMatch(/\d/);
    expect(formatExactPlayedAt(iso)).toContain("2026");
  });
});
