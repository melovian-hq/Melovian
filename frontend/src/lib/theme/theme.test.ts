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

  it("applies a palette through data-palette and persists it for both modes", () => {
    const store = freshThemeStore();
    store.setMode("dark");
    store.setPalette("oled");
    flushSync();
    expect(store.palette).toBe("oled");
    expect(document.documentElement.dataset.palette).toBe("oled");
    expect(localStorage.getItem(StorageKeys.themePaletteDark)).toBe("oled");
    expect(localStorage.getItem(StorageKeys.themePaletteLight)).toBe("oled");

    store.setPalette("default");
    flushSync();
    expect(document.documentElement.dataset.palette).toBeUndefined();
    expect(localStorage.getItem(StorageKeys.themePaletteDark)).toBeNull();
    expect(localStorage.getItem(StorageKeys.themePaletteLight)).toBeNull();
  });

  it("rejects unknown palette ids", () => {
    const store = freshThemeStore();
    store.setMode("dark");
    flushSync();
    store.setPalette("does-not-exist");
    flushSync();
    expect(store.palette).toBe("default");
    expect(document.documentElement.dataset.palette).toBeUndefined();
  });

  it("keeps the same palette id when the resolved theme flips", () => {
    const store = freshThemeStore();
    store.setMode("dark");
    store.setPalette("midnight");
    flushSync();
    store.setMode("light");
    flushSync();
    expect(store.palette).toBe("midnight");
    expect(document.documentElement.dataset.palette).toBe("midnight");
  });

  it("applies a custom palette through the jb-custom variables", () => {
    const store = freshThemeStore();
    store.setMode("dark");
    store.setPalette("custom");
    flushSync();
    store.setCustomPaletteColor("dark", "bg", "#112233");
    store.setCustomPaletteColor("dark", "surface", "#223344");
    store.setCustomPaletteColor("dark", "accent", "#33cc66");
    flushSync();
    const style = document.documentElement.style;
    expect(document.documentElement.dataset.palette).toBe("custom");
    expect(style.getPropertyValue("--jb-custom-bg")).toBe("#112233");
    expect(style.getPropertyValue("--jb-custom-surface")).toBe("#223344");
    expect(style.getPropertyValue("--jb-custom-accent")).toBe("#33cc66");
    const stored = JSON.parse(
      localStorage.getItem(StorageKeys.themeCustomPalette) ?? "{}",
    );
    expect(stored.dark).toEqual({
      bg: "#112233",
      surface: "#223344",
      accent: "#33cc66",
    });
  });

  it("rejects malformed custom palette colors", () => {
    const store = freshThemeStore();
    store.setMode("dark");
    flushSync();
    store.setCustomPaletteColor("dark", "bg", "javascript:alert(1)");
    store.setCustomPaletteColor("dark", "bg", "red");
    flushSync();
    expect(store.palette).toBe("default");
    expect(
      document.documentElement.style.getPropertyValue("--jb-custom-bg"),
    ).toBe("");
  });

  it("clears custom palette variables when another palette is picked", () => {
    const store = freshThemeStore();
    store.setMode("dark");
    store.setPalette("custom");
    store.setCustomPaletteColor("dark", "bg", "#112233");
    flushSync();
    store.setPalette("forest");
    flushSync();
    const style = document.documentElement.style;
    expect(document.documentElement.dataset.palette).toBe("forest");
    expect(style.getPropertyValue("--jb-custom-bg")).toBe("");
    expect(style.getPropertyValue("--jb-custom-surface")).toBe("");
  });

  it("applies square radius as zero on every radius size", () => {
    const store = freshThemeStore();
    store.setRadiusStyle("square");
    flushSync();
    const style = document.documentElement.style;
    for (const size of ["sm", "md", "lg", "xl", "full"]) {
      expect(style.getPropertyValue(`--jb-radius-${size}`)).toBe("0");
    }
    expect(localStorage.getItem(StorageKeys.themeRadius)).toBe("square");

    store.setRadiusStyle("default");
    flushSync();
    expect(style.getPropertyValue("--jb-radius-md")).toBe("");
  });

  it("applies radius and interface size overrides inline", () => {
    const store = freshThemeStore();
    store.setRadiusStyle("sharp");
    store.setUiSize("compact");
    flushSync();
    const style = document.documentElement.style;
    expect(style.getPropertyValue("--jb-radius-md")).toBe("0.25rem");
    expect(style.getPropertyValue("font-size")).toBe("90%");
    expect(localStorage.getItem(StorageKeys.themeRadius)).toBe("sharp");
    expect(localStorage.getItem(StorageKeys.themeUiSize)).toBe("compact");

    store.setRadiusStyle("default");
    store.setUiSize("default");
    flushSync();
    expect(style.getPropertyValue("--jb-radius-md")).toBe("");
    expect(style.getPropertyValue("font-size")).toBe("");
  });
});
