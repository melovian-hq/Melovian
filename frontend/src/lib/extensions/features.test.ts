// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, beforeEach, afterEach } from "vitest";
import {
  EXT_LYRICS,
  EXT_METADATA,
  extensionFeatures,
} from "./features.svelte";
import { visibleSettingsTabs } from "$lib/settings/tabs";

const EXT_DEMO_THEME = "demo-theme";

describe("extensionFeatures", () => {
  beforeEach(() => {
    extensionFeatures.resetForTests([EXT_LYRICS, EXT_METADATA]);
  });

  afterEach(() => {
    extensionFeatures.resetForTests([EXT_LYRICS, EXT_METADATA], "");
  });

  it("reports bundled feature toggles", () => {
    expect(extensionFeatures.lyrics).toBe(true);
    expect(extensionFeatures.metadata).toBe(true);
    extensionFeatures.applyFromItems([
      { id: EXT_LYRICS, enabled: false },
      { id: EXT_METADATA, enabled: true },
    ]);
    expect(extensionFeatures.lyrics).toBe(false);
    expect(extensionFeatures.metadata).toBe(true);
  });

  it("hides lyrics settings tab when lyrics extension is off", () => {
    const withLyrics = visibleSettingsTabs({ lyrics: true });
    expect(withLyrics.some((tab) => tab.id === "lyrics")).toBe(true);
    const without = visibleSettingsTabs({ lyrics: false });
    expect(without.some((tab) => tab.id === "lyrics")).toBe(false);
  });

  it("offers a theme when an extension is enabled but does not activate it", () => {
    extensionFeatures.applyFromItems([
      { id: EXT_LYRICS, enabled: true },
      {
        id: EXT_DEMO_THEME,
        enabled: true,
        appTheme: "neon",
      },
    ]);
    expect(extensionFeatures.availableThemes.has("neon")).toBe(true);
    expect(extensionFeatures.appTheme).toBe("");
    expect(document.documentElement.dataset.extensionTheme).toBeUndefined();
  });

  it("activates chrome only for matching track decoration", () => {
    extensionFeatures.applyFromItems([
      {
        id: EXT_DEMO_THEME,
        enabled: true,
        appTheme: "neon",
      },
    ]);
    extensionFeatures.syncFromTrackDecoration({ playerTheme: "neon" });
    expect(extensionFeatures.appTheme).toBe("neon");
    expect(document.documentElement.dataset.extensionTheme).toBe("neon");

    extensionFeatures.syncFromTrackDecoration({});
    expect(extensionFeatures.appTheme).toBe("");
    expect(document.documentElement.dataset.extensionTheme).toBeUndefined();
  });

  it("ignores decoration themes that no enabled extension offers", () => {
    extensionFeatures.applyFromItems([
      {
        id: EXT_DEMO_THEME,
        enabled: true,
        appTheme: "neon",
      },
    ]);
    extensionFeatures.syncFromTrackDecoration({ playerTheme: "neon-alt" });
    expect(extensionFeatures.appTheme).toBe("");
    expect(document.documentElement.dataset.extensionTheme).toBeUndefined();
  });

  it("clears active theme when the extension is disabled", () => {
    extensionFeatures.applyFromItems([
      {
        id: EXT_DEMO_THEME,
        enabled: true,
        appTheme: "neon",
      },
    ]);
    extensionFeatures.syncFromTrackDecoration({ playerTheme: "neon" });
    extensionFeatures.applyFromItems([
      {
        id: EXT_DEMO_THEME,
        enabled: false,
        appTheme: "neon",
      },
    ]);
    expect(extensionFeatures.availableThemes.has("neon")).toBe(false);
    expect(extensionFeatures.appTheme).toBe("");
    expect(document.documentElement.dataset.extensionTheme).toBeUndefined();
  });

  it("refreshes available themes from manifests without activating", () => {
    extensionFeatures.applyAppThemeFromManifests([
      { id: EXT_DEMO_THEME, appTheme: "neon" },
    ]);
    expect(extensionFeatures.availableThemes.has("neon")).toBe(true);
    expect(extensionFeatures.appTheme).toBe("");
    expect(document.documentElement.dataset.extensionTheme).toBeUndefined();

    extensionFeatures.applyAppThemeFromManifests([]);
    expect(extensionFeatures.availableThemes.has("neon")).toBe(false);
  });
});
