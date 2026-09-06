// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, beforeEach } from "vitest";
import {
  ALL_MIX_IDS,
  defaultMixSettings,
  isMixEnabled,
  loadMixSettings,
  mergeMixSettings,
  parsePreferredLanguages,
  saveMixSettings,
} from "./mix-settings";

describe("mix-settings", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("defaults to a 30-track minimum per mix", () => {
    expect(defaultMixSettings().minTracksPerMix).toBe(30);
  });

  it("caps minimum tracks at the configured maximum", () => {
    const merged = mergeMixSettings({
      minTracksPerMix: 40,
      maxTracksPerMix: 25,
    });
    expect(merged.minTracksPerMix).toBe(25);
    expect(merged.maxTracksPerMix).toBe(25);
  });

  it("merges partial settings with defaults", () => {
    const merged = mergeMixSettings({
      discoverRecentDays: 14,
      maxTracksPerMix: 200,
    });
    expect(merged.discoverRecentDays).toBe(14);
    expect(merged.maxTracksPerMix).toBe(100);
    expect(merged.languageBias).toBe(defaultMixSettings().languageBias);
  });

  it("persists settings in localStorage", () => {
    saveMixSettings({ ...defaultMixSettings(), languageBias: "strict" });
    expect(loadMixSettings().languageBias).toBe("strict");
  });

  it("treats empty enabled list as all mixes enabled", () => {
    const settings = defaultMixSettings();
    expect(isMixEnabled("discover", settings)).toBe(true);
  });

  it("filters enabled mixes when configured", () => {
    const settings = mergeMixSettings({
      enabledMixIds: ["discover", "on-repeat"],
    });
    expect(isMixEnabled("discover", settings)).toBe(true);
    expect(isMixEnabled("replay", settings)).toBe(false);
  });

  it("expands legacy genre-mix and decade-mix toggles into slot ids", () => {
    const settings = mergeMixSettings({
      enabledMixIds: ["genre-mix", "decade-mix"] as never,
    });
    expect(settings.enabledMixIds).toEqual([
      "genre-mix-1",
      "genre-mix-2",
      "genre-mix-3",
      "decade-mix-1",
      "decade-mix-2",
      "decade-mix-3",
    ]);
    expect(isMixEnabled("genre-mix-2", settings)).toBe(true);
    expect(isMixEnabled("decade-mix-1", settings)).toBe(true);
  });

  it("keeps ALL_MIX_IDS aligned with mix generator priority order", async () => {
    const { MIX_PRIORITY } = await import("./mix-generator/priority");
    expect([...ALL_MIX_IDS]).toEqual([...MIX_PRIORITY]);
  });

  it("parses preferred languages", () => {
    expect(parsePreferredLanguages("ru, en")).toEqual(["ru", "en"]);
  });
});
