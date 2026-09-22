// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  filterSettingsTabs,
  isSettingsPath,
  isSettingsTabId,
  navTabsForMode,
  SETTINGS_TAB_IDS,
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

  it("keeps SETTINGS_TAB_IDS aligned with SETTINGS_TABS order", () => {
    expect([...SETTINGS_TAB_IDS]).toEqual(SETTINGS_TABS.map((tab) => tab.id));
    expect(SETTINGS_TABS.map((tab) => tab.id)).toEqual([
      "profile",
      "general",
      "servers",
      "playback",
      "downloads",
      "about",
      "mixes",
      "connection",
      "lyrics",
      "video",
      "rocksky",
      "listenbrainz",
      "lastfm",
      "extensions",
      "tasks",
    ]);
    const recommendedCount = SETTINGS_TABS.filter(
      (tab) => tab.tier !== "advanced",
    ).length;
    expect(
      SETTINGS_TABS.slice(recommendedCount).every(
        (tab) => tab.tier === "advanced",
      ),
    ).toBe(true);
  });

  it("includes about and tasks in visible tabs", () => {
    const tabs = visibleSettingsTabs({ lyrics: true });
    expect(tabs.some((tab) => tab.id === "about")).toBe(true);
    expect(tabs.some((tab) => tab.id === "tasks")).toBe(true);
    expect(tabs.find((tab) => tab.id === "about")?.tier).toBe("recommended");
    expect(tabs.find((tab) => tab.id === "tasks")?.tier).toBe("advanced");
  });

  it("limits simple mode to recommended tabs", () => {
    const ids = navTabsForMode(SETTINGS_TABS, "simple", "general", false).map(
      (t) => t.id,
    );
    expect(ids).toEqual([
      "profile",
      "general",
      "servers",
      "playback",
      "downloads",
      "about",
    ]);
  });

  it("shows every tab in advanced mode", () => {
    expect(
      navTabsForMode(SETTINGS_TABS, "advanced", "general", false).map(
        (t) => t.id,
      ),
    ).toEqual(SETTINGS_TABS.map((t) => t.id));
  });

  it("pins the active advanced tab in simple mode", () => {
    const ids = navTabsForMode(
      SETTINGS_TABS,
      "simple",
      "extensions",
      false,
    ).map((t) => t.id);
    expect(ids.at(-1)).toBe("extensions");
    expect(ids).toContain("profile");
  });

  it("ignores the mode while searching", () => {
    const hits = filterSettingsTabs(SETTINGS_TABS, "lrclib");
    expect(
      navTabsForMode(hits, "simple", "lyrics", true).map((t) => t.id),
    ).toEqual(["lyrics"]);
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
