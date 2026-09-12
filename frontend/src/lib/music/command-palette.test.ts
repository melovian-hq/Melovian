// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";

vi.mock("$lib/theme/theme.svelte", () => ({
  theme: { setMode: vi.fn() },
}));

vi.mock("$lib/config/music.svelte", () => ({
  music: {
    connected: false,
    currentTrack: null,
    togglePlay: vi.fn(),
    next: vi.fn(),
    previous: vi.fn(),
    toggleQueue: vi.fn(),
    searchAll: vi.fn(),
    playTrackById: vi.fn(),
    playLibraryShuffle: vi.fn(),
  },
}));

vi.mock("$lib/router/router.svelte", () => ({
  router: { navigate: vi.fn() },
}));

vi.mock("$lib/ui/command-palette.svelte", () => ({
  commandPalette: { close: vi.fn() },
}));

vi.mock("$lib/ui/keyboard-help.svelte", () => ({
  keyboardHelp: { toggle: vi.fn() },
}));

vi.mock("$lib/components/layout/layout.svelte", () => ({
  layout: { enterTvMode: vi.fn(), exitTvMode: vi.fn(), tvMode: false },
}));

import {
  filterPaletteCommands,
  scorePaletteCommand,
  searchPaletteCommands,
  type PaletteCommand,
} from "./command-palette";

const sample: PaletteCommand[] = [
  {
    id: "a",
    label: "Go to History",
    group: "Navigate",
    run: () => {},
  },
  {
    id: "b",
    label: "Play / Pause",
    group: "Playback",
    keywords: ["toggle"],
    run: () => {},
  },
];

describe("command-palette", () => {
  it("scores and filters commands", () => {
    expect(scorePaletteCommand(sample[0], "history")).toBeGreaterThan(0);
    expect(filterPaletteCommands(sample, "toggle")).toHaveLength(1);
    expect(searchPaletteCommands("").length).toBeGreaterThan(5);
  });

  it("includes the now playing navigation command", () => {
    const commands = searchPaletteCommands("now playing");
    expect(commands.some((command) => command.id === "nav-now-playing")).toBe(
      true,
    );
  });

  it("includes the TV mode command", () => {
    const commands = searchPaletteCommands("tv");
    expect(commands.some((command) => command.id === "tv-mode")).toBe(true);
  });

  it("TV mode command only enters overlay when a track is playing", async () => {
    const { music } = await import("$lib/config/music.svelte");
    const { layout } = await import("$lib/components/layout/layout.svelte");
    const { router } = await import("$lib/router/router.svelte");
    const tv = searchPaletteCommands("tv").find(
      (command) => command.id === "tv-mode",
    );
    expect(tv).toBeTruthy();

    music.currentTrack = null;
    await tv!.run();
    expect(router.navigate).toHaveBeenCalledWith("/music/now-playing");
    expect(layout.enterTvMode).not.toHaveBeenCalled();

    music.currentTrack = { id: "t1", title: "Song" } as never;
    await tv!.run();
    expect(layout.enterTvMode).toHaveBeenCalled();
  });

  it("includes a library shuffle command that starts continuous refill", async () => {
    const { music } = await import("$lib/config/music.svelte");
    const command = searchPaletteCommands("shuffle library").find(
      (entry) => entry.id === "shuffle-library",
    );
    expect(command).toBeTruthy();
    await command!.run();
    expect(music.playLibraryShuffle).toHaveBeenCalled();
  });

  it("does not include stats or radio navigation", () => {
    const commands = searchPaletteCommands("");
    expect(commands.some((command) => command.id === "nav-stats")).toBe(false);
    expect(commands.some((command) => command.id === "nav-radios")).toBe(false);
  });
});
