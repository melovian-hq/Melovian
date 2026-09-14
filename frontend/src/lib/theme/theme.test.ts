// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { flushSync } from "svelte";
import { beforeEach, describe, expect, it } from "vitest";
import { StorageKeys } from "$lib/brand";
import {
  ACCENT_PRESETS,
  CUSTOM_ACCENT_ID,
  DEFAULT_ACCENT_HUE,
  DEFAULT_ACCENT_PRESET_ID,
  hexToHue,
} from "./accent";
import { resolveThemeMode, ThemeStore } from "./theme.svelte";

function freshThemeStore() {
  const store = new ThemeStore();
  flushSync();
  return store;
}

describe("resolveThemeMode", () => {
  it("passes explicit modes through", () => {
    expect(resolveThemeMode("dark", false)).toBe("dark");
    expect(resolveThemeMode("dark", true)).toBe("dark");
    expect(resolveThemeMode("light", true)).toBe("light");
    expect(resolveThemeMode("light", false)).toBe("light");
  });

  it("resolves system from the OS preference", () => {
    expect(resolveThemeMode("system", true)).toBe("dark");
    expect(resolveThemeMode("system", false)).toBe("light");
  });
});

describe("theme store", () => {
  beforeEach(() => {
    document.documentElement.removeAttribute("data-theme");
    document.documentElement.style.removeProperty("--jb-accent");
    document.documentElement.style.removeProperty("--jb-accent-text");
  });

  it("persists the mode and applies data-theme", () => {
    const store = freshThemeStore();
    store.setMode("light");
    flushSync();
    expect(localStorage.getItem(StorageKeys.theme)).toBe("light");
    expect(document.documentElement.dataset.theme).toBe("light");

    store.setMode("system");
    flushSync();
    expect(localStorage.getItem(StorageKeys.theme)).toBe("system");
    // The matchMedia stub reports light.
    expect(document.documentElement.dataset.theme).toBe("light");
  });

  it("applies preset accents inline and persists the preset id", () => {
    const store = freshThemeStore();
    const ocean = ACCENT_PRESETS.find((preset) => preset.id === "ocean");
    expect(ocean).toBeDefined();

    store.setAccentPreset("ocean");
    flushSync();
    expect(localStorage.getItem(StorageKeys.themeAccentPreset)).toBe("ocean");
    expect(document.documentElement.style.getPropertyValue("--jb-accent")).toBe(
      ocean!.dark,
    );
    expect(
      document.documentElement.style.getPropertyValue("--jb-accent-text"),
    ).not.toBe("");
  });

  it("clears accent overrides when the default preset is restored", () => {
    const store = freshThemeStore();
    store.setAccentPreset("ocean");
    flushSync();
    store.setAccentPreset(DEFAULT_ACCENT_PRESET_ID);
    flushSync();
    expect(localStorage.getItem(StorageKeys.themeAccentPreset)).toBeNull();
    expect(document.documentElement.style.getPropertyValue("--jb-accent")).toBe(
      "",
    );
    expect(
      document.documentElement.style.getPropertyValue("--jb-accent-text"),
    ).toBe("");
  });

  it("switches to the custom preset when a hue is picked", () => {
    const store = freshThemeStore();
    store.setCustomHue(200);
    flushSync();
    expect(store.accentPreset).toBe(CUSTOM_ACCENT_ID);
    expect(store.customHue).toBe(200);
    expect(localStorage.getItem(StorageKeys.themeAccentHue)).toBe("200");
    expect(localStorage.getItem(StorageKeys.themeAccentPreset)).toBe(
      CUSTOM_ACCENT_ID,
    );
  });

  it("migrates a legacy hex accent into the custom hue flow", () => {
    localStorage.setItem(StorageKeys.themeAccent, "#22cc55");
    const store = freshThemeStore();
    expect(store.accentPreset).toBe(CUSTOM_ACCENT_ID);
    expect(store.customHue).toBe(hexToHue("#22cc55"));
    // The legacy key is consumed on load.
    expect(localStorage.getItem(StorageKeys.themeAccent)).toBeNull();
  });

  it("resets accent and custom css together", () => {
    const store = freshThemeStore();
    store.setCustomHue(120);
    store.setCustomCss(":root { --jb-bg: #000000; }");
    flushSync();
    store.resetCustomization();
    flushSync();
    expect(store.accentPreset).toBe(DEFAULT_ACCENT_PRESET_ID);
    expect(store.customHue).toBe(DEFAULT_ACCENT_HUE);
    expect(store.customCss).toBe("");
    expect(localStorage.getItem(StorageKeys.themeAccentPreset)).toBeNull();
    expect(localStorage.getItem(StorageKeys.customCss)).toBeNull();
  });
});
