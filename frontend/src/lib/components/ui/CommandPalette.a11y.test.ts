// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { flushSync, mount, unmount } from "svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  expectNoA11yViolations,
  removeLeftoverPortalNodes,
} from "../../../test-fixtures/a11y";

// The real search module pulls in the music, router, theme, and extension
// stores. The palette markup only needs commands back, so stub the search.
vi.mock("$lib/music/command-palette", () => ({
  searchPaletteWithMusic: vi.fn(async () => [
    {
      id: "nav-home",
      label: "Go to Home",
      group: "Navigate",
      icon: "music",
      run: () => {},
    },
    {
      id: "play-pause",
      label: "Play / Pause",
      group: "Playback",
      icon: "play",
      run: () => {},
    },
  ]),
}));

import { commandPalette } from "$lib/ui/command-palette.svelte";
import CommandPalette from "./CommandPalette.svelte";

const cleanups: Array<() => Promise<void> | void> = [];

afterEach(async () => {
  commandPalette.close();
  for (const cleanup of cleanups.splice(0)) {
    await cleanup();
  }
  removeLeftoverPortalNodes();
});

describe("CommandPalette accessibility", () => {
  it("has no violations for the open palette with grouped results", async () => {
    const target = document.createElement("div");
    document.body.appendChild(target);
    const instance = mount(CommandPalette, { target });
    flushSync();
    cleanups.push(async () => {
      await unmount(instance);
      target.remove();
    });

    commandPalette.openPalette();
    flushSync();

    // Results load after the component's debounce timer
    await vi.waitFor(() => {
      expect(
        document.querySelectorAll("[role='option']").length,
      ).toBeGreaterThan(0);
    });

    expect(document.querySelector("[role='dialog']")).not.toBeNull();
    await expectNoA11yViolations(document.body);
  });
});
