// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it } from "vitest";
import {
  getOrCreateDeviceId,
  guessDeviceName,
  loadDeviceName,
  saveDeviceName,
} from "./device-id";

describe("device-id", () => {
  afterEach(() => {
    localStorage.removeItem("mel-device-id");
    localStorage.removeItem("mel-device-name");
  });

  it("creates a stable device id", () => {
    const first = getOrCreateDeviceId();
    const second = getOrCreateDeviceId();
    expect(first).toBeTruthy();
    expect(second).toBe(first);
  });

  it("persists renamed device name", () => {
    saveDeviceName("  Kitchen Speaker  ");
    expect(loadDeviceName()).toBe("Kitchen Speaker");
  });

  it("guesses names from user agents", () => {
    expect(guessDeviceName("Mozilla/5.0 (iPhone; CPU iPhone OS 17)")).toBe(
      "iPhone",
    );
    expect(guessDeviceName("Mozilla/5.0 (Windows NT 10.0; Win64)")).toBe(
      "Windows",
    );
    expect(guessDeviceName("totally-unknown")).toBe("Browser");
  });
});
