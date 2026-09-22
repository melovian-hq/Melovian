// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { flushSync, mount, unmount } from "svelte";
import { describe, expect, it, vi } from "vitest";
import SettingsNav from "./SettingsNav.svelte";
import { SETTINGS_TABS } from "$lib/settings/tabs";
import {
  setSettingsNavMode,
  settingsNavMode,
} from "$lib/settings/nav-state.svelte";

function renderNav(props: {
  active?: string;
  searching?: boolean;
  onselect?: (id: string) => void;
}) {
  const target = document.createElement("div");
  document.body.appendChild(target);
  const instance = mount(SettingsNav, {
    target,
    props: {
      tabs: SETTINGS_TABS,
      active: (props.active ?? "general") as never,
      onselect: props.onselect ?? vi.fn(),
      searching: props.searching ?? false,
    },
  });
  flushSync();
  return {
    target,
    labels: () =>
      [...target.querySelectorAll(".settings-nav__label")].map(
        (el) => el.textContent,
      ),
    cleanup: () => {
      void unmount(instance);
      target.remove();
    },
  };
}

function modeButton(name: string): HTMLButtonElement {
  const btn = document.querySelector<HTMLButtonElement>(
    `[role="tab"][data-value="${name}"], .settings-nav__mode-btn`,
  );
  const match = [
    ...document.querySelectorAll<HTMLButtonElement>("button"),
  ].find(
    (b) => b.textContent?.trim() === name && b.getAttribute("role") === "tab",
  );
  return (match ?? btn)!;
}

describe("SettingsNav", () => {
  it("defaults to simple mode with recommended tabs only", () => {
    setSettingsNavMode("simple");
    const { labels, cleanup } = renderNav({});
    expect(labels()).toEqual([
      "Profile",
      "General",
      "Sources",
      "Playback",
      "Downloads",
      "About",
    ]);
    expect(modeButton("Simple").getAttribute("data-state")).toBe("active");
    cleanup();
  });

  it("shows every tab in advanced mode", () => {
    setSettingsNavMode("simple");
    const { labels, cleanup } = renderNav({});
    modeButton("Advanced").click();
    flushSync();
    expect(settingsNavMode()).toBe("advanced");
    expect(labels()).toHaveLength(SETTINGS_TABS.length);
    cleanup();
    setSettingsNavMode("simple");
  });

  it("pins the active advanced tab while in simple mode", () => {
    setSettingsNavMode("simple");
    const { labels, cleanup } = renderNav({ active: "extensions" });
    expect(labels().at(-1)).toBe("Extensions");
    cleanup();
  });
});
