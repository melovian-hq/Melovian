// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  filterSettingsTabs,
  isSettingsPath,
  isSettingsTabId,
  SETTINGS_TABS,
  settingsTabFromPath,
  settingsTabPath,
  visibleSettingsTabs,
} from "./tabs";

describe("settings tabs", () => {
  it("parses settings paths", () => {
    expect(settingsTabFromPath("/settings")).toBe("");
    expect(settingsTabFromPath("/settings/profile")).toBe("profile");
    expect(settingsTabFromPath("/settings/playback")).toBe("playback");
    expect(settingsTabFromPath("/settings/about")).toBe("about");
    expect(settingsTabFromPath("/settings/tasks")).toBe("tasks");
    expect(isSettingsPath("/settings/mixes")).toBe(true);
    expect(isSettingsPath("/music")).toBe(false);
  });

  it("validates tab ids", () => {
    expect(isSettingsTabId("profile")).toBe(true);
    expect(isSettingsTabId("lyrics")).toBe(true);
    expect(isSettingsTabId("video")).toBe(true);
    expect(isSettingsTabId("about")).toBe(true);
    expect(isSettingsTabId("tasks")).toBe(true);
    expect(isSettingsTabId("nope")).toBe(false);
    expect(settingsTabPath("profile")).toBe("/settings/profile");
    expect(settingsTabPath("general")).toBe("/settings/general");
    expect(settingsTabPath("video")).toBe("/settings/video");
    expect(settingsTabPath("about")).toBe("/settings/about");
    expect(settingsTabPath("tasks")).toBe("/settings/tasks");
  });

  it("includes about and tasks in visible tabs", () => {
    const tabs = visibleSettingsTabs({ lyrics: true });
    expect(tabs.some((tab) => tab.id === "about")).toBe(true);
    expect(tabs.some((tab) => tab.id === "tasks")).toBe(true);
    expect(tabs.find((tab) => tab.id === "about")?.tier).toBe("recommended");
    expect(tabs.find((tab) => tab.id === "tasks")?.tier).toBe("advanced");
  });

  it("filters tabs by label description and keywords", () => {
    expect(filterSettingsTabs(SETTINGS_TABS, "")).toHaveLength(
      SETTINGS_TABS.length,
    );
    expect(filterSettingsTabs(SETTINGS_TABS, "  ").map((t) => t.id)).toEqual(
      SETTINGS_TABS.map((t) => t.id),
    );
    expect(
      filterSettingsTabs(SETTINGS_TABS, "playback").map((t) => t.id),
    ).toEqual(["playback"]);
    const playHits = filterSettingsTabs(SETTINGS_TABS, "play").map((t) => t.id);
    expect(playHits).toContain("playback");
    expect(playHits.length).toBeGreaterThanOrEqual(1);
    expect(
      filterSettingsTabs(SETTINGS_TABS, "lrclib").map((t) => t.id),
    ).toEqual(["lyrics"]);
    expect(filterSettingsTabs(SETTINGS_TABS, "theme").map((t) => t.id)).toEqual(
      ["general"],
    );
    expect(filterSettingsTabs(SETTINGS_TABS, "zzzz")).toEqual([]);
  });
});
