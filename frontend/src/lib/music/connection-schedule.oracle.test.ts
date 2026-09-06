// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import {
  averageOfflineDurationMs,
  computeReconnectDelayMs,
  recentOfflineDurationMs,
} from "./connection-schedule";
import { defaultConnectionSettings } from "./connection-settings";

describe("connection-schedule oracle", () => {
  it("average offline duration is null without completed events", () => {
    expect(averageOfflineDurationMs([])).toBeNull();
    expect(averageOfflineDurationMs([{ disconnectedAt: 100 }])).toBeNull();
  });

  it("reconnect delay stays within configured bounds when online", () => {
    const settings = defaultConnectionSettings();
    fc.assert(
      fc.property(fc.integer({ min: 0, max: 12 }), (attempt) => {
        const delay = computeReconnectDelayMs(settings, attempt, [], true);
        expect(delay).toBeGreaterThanOrEqual(settings.minDelayMs);
        expect(delay).toBeLessThanOrEqual(settings.maxDelayMs);
      }),
    );
  });

  it("offline browser uses offline poll interval", () => {
    const settings = defaultConnectionSettings();
    expect(computeReconnectDelayMs(settings, 5, [], false)).toBe(
      settings.offlinePollMs,
    );
  });

  it("recentOfflineDurationMs returns the latest completed outage", () => {
    expect(
      recentOfflineDurationMs([
        { disconnectedAt: 0, reconnectedAt: 1000 },
        { disconnectedAt: 2000, reconnectedAt: 2500 },
      ]),
    ).toBe(500);
  });
});
