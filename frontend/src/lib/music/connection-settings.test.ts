// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it } from "vitest";
import {
  appendConnectionEvent,
  closeLatestConnectionEvent,
  defaultConnectionSettings,
  loadConnectionSettings,
  mergeConnectionSettings,
  saveConnectionSettings,
} from "./connection-settings";

describe("connection-settings", () => {
  afterEach(() => {
    localStorage.removeItem("mel-connection-settings");
    localStorage.removeItem("mel-connection-history");
  });

  it("persists connection settings", () => {
    const settings = { ...defaultConnectionSettings(), minDelayMs: 5000 };
    saveConnectionSettings(settings);
    expect(loadConnectionSettings().minDelayMs).toBe(5000);
  });

  it("merges partial connection settings", () => {
    const merged = mergeConnectionSettings({ minDelayMs: 8000 });
    expect(merged.minDelayMs).toBe(8000);
    expect(merged.autoReconnect).toBe(true);
  });

  it("records and closes disconnect events", () => {
    let history = appendConnectionEvent([], { disconnectedAt: 1000 }, 8);
    history = closeLatestConnectionEvent(history, 9000);
    expect(history[0]).toEqual({ disconnectedAt: 1000, reconnectedAt: 9000 });
  });
});
