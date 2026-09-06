// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";
import { flushSync, mount, unmount } from "svelte";
import type { ParsedLyrics } from "$lib/music/lyrics";
import LyricsDisplay from "./LyricsDisplay.svelte";

const syncedLyrics: ParsedLyrics = {
  synced: true,
  offsetMs: 0,
  rawValue: "[00:00.00]Line one\n[00:02.00]Line two",
  lines: [
    { startMs: 0, text: "Line one" },
    { startMs: 2000, text: "Line two" },
  ],
};

function renderLyrics(
  props: Partial<{
    lyrics: ParsedLyrics;
    currentTimeMs: number;
    playing: boolean;
    onSeek: (startMs: number) => void;
  }> = {},
) {
  const target = document.createElement("div");
  document.body.appendChild(target);
  const onSeek = props.onSeek ?? vi.fn();
  const instance = mount(LyricsDisplay, {
    target,
    props: {
      lyrics: syncedLyrics,
      currentTimeMs: 0,
      playing: false,
      onSeek,
      ...props,
    },
  });
  flushSync();
  return {
    target,
    onSeek,
    cleanup: () => {
      unmount(instance);
      target.remove();
    },
  };
}

describe("LyricsDisplay", () => {
  it("highlights the active synced line while playing", () => {
    const { target, cleanup } = renderLyrics({
      currentTimeMs: 2100,
      playing: true,
    });

    const lines = target.querySelectorAll(".lyrics-display__line");
    expect(lines[1]?.classList.contains("lyrics-display__line--active")).toBe(
      true,
    );
    cleanup();
  });

  it("highlights the active synced line while paused", () => {
    const { target, cleanup } = renderLyrics({
      currentTimeMs: 2100,
      playing: false,
    });

    const lines = target.querySelectorAll(".lyrics-display__line");
    expect(lines[1]?.classList.contains("lyrics-display__line--active")).toBe(
      true,
    );
    cleanup();
  });

  it("seeks when a synced line is clicked", () => {
    const onSeek = vi.fn();
    const { target, cleanup } = renderLyrics({ onSeek });

    const buttons = target.querySelectorAll("button.lyrics-display__line");
    (buttons[1] as HTMLButtonElement).click();
    expect(onSeek).toHaveBeenCalledWith(2000);
    cleanup();
  });

  it("applies synced styling when lyrics are timed", () => {
    const { target, cleanup } = renderLyrics();

    expect(target.querySelector(".lyrics-display--synced")).not.toBeNull();
    cleanup();
  });
});
