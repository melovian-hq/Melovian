// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";
import { StorageKeys } from "$lib/brand";
import { settingsNavMode, setSettingsNavMode } from "./nav-state.svelte";

describe("settings nav mode", () => {
  it("defaults to simple", () => {
    expect(settingsNavMode()).toBe("simple");
  });

  it("persists the mode across recreation", async () => {
    setSettingsNavMode("advanced");
    expect(settingsNavMode()).toBe("advanced");
    expect(localStorage.getItem(StorageKeys.settingsNavMode)).toBe(
      '"advanced"',
    );

    vi.resetModules();
    const fresh = await import("./nav-state.svelte");
    expect(fresh.settingsNavMode()).toBe("advanced");

    fresh.setSettingsNavMode("simple");
    expect(fresh.settingsNavMode()).toBe("simple");
  });
});
