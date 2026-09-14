// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";
import { StorageKeys } from "$lib/brand";
import {
  settingsNavAdvancedCollapsed,
  settingsNavAdvancedOpen,
  toggleSettingsNavAdvanced,
} from "./nav-state.svelte";

describe("settings nav advanced state", () => {
  it("defaults to collapsed", () => {
    expect(settingsNavAdvancedCollapsed()).toBe(true);
  });

  it("persists the toggle across recreation", async () => {
    toggleSettingsNavAdvanced();
    expect(settingsNavAdvancedCollapsed()).toBe(false);
    expect(localStorage.getItem(StorageKeys.settingsNavAdvancedCollapsed)).toBe(
      "false",
    );

    vi.resetModules();
    const fresh = await import("./nav-state.svelte");
    expect(fresh.settingsNavAdvancedCollapsed()).toBe(false);

    fresh.toggleSettingsNavAdvanced();
    expect(fresh.settingsNavAdvancedCollapsed()).toBe(true);
  });

  it("force-expands while the active tab is advanced", () => {
    expect(settingsNavAdvancedOpen(true, true)).toBe(true);
    expect(settingsNavAdvancedOpen(true, false)).toBe(false);
    expect(settingsNavAdvancedOpen(false, false)).toBe(true);
    expect(settingsNavAdvancedOpen(false, true)).toBe(true);
  });

  it("does not write the stored preference when force-expanded", () => {
    const open = settingsNavAdvancedOpen(settingsNavAdvancedCollapsed(), true);
    expect(open).toBe(true);
    expect(settingsNavAdvancedCollapsed()).toBe(true);
  });
});
