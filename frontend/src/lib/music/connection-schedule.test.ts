// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  averageOfflineDurationMs,
  computeReconnectDelayMs,
  phaseForAttempt,
  recentOfflineDurationMs,
} from "./connection-schedule";
import { defaultConnectionSettings } from "./connection-settings";

describe("connection-schedule", () => {
  it("uses shorter delay after brief outages", () => {
    const settings = defaultConnectionSettings();
    const history = [
      { disconnectedAt: 0, reconnectedAt: 10_000 },
      { disconnectedAt: 20_000, reconnectedAt: 25_000 },
    ];

    const delay = computeReconnectDelayMs(settings, 3, history, true);
    expect(delay).toBe(settings.minDelayMs);
  });

  it("extends delay after long average outages", () => {
    const settings = defaultConnectionSettings();
    const history = [
      { disconnectedAt: 0, reconnectedAt: 120_000 },
      { disconnectedAt: 200_000, reconnectedAt: 320_000 },
    ];

    const delay = computeReconnectDelayMs(settings, 1, history, true);
    expect(delay).toBeGreaterThan(settings.minDelayMs);
  });

  it("polls while browser reports offline", () => {
    const settings = defaultConnectionSettings();
    const delay = computeReconnectDelayMs(settings, 0, [], false);
    expect(delay).toBe(settings.offlinePollMs);
  });

  it("tracks average and recent offline durations", () => {
    const history = [
      { disconnectedAt: 0, reconnectedAt: 20_000 },
      { disconnectedAt: 100, reconnectedAt: 50_100 },
    ];
    expect(recentOfflineDurationMs(history)).toBe(50_000);
    expect(averageOfflineDurationMs(history)).toBe(35_000);
  });

  it("maps attempts to connection phases", () => {
    expect(phaseForAttempt(0, false)).toBe("reconnecting");
    expect(phaseForAttempt(1, false)).toBe("reconnecting");
    expect(phaseForAttempt(2, false)).toBe("retrying");
    expect(phaseForAttempt(0, true)).toBe("self-healing");
  });
});
