// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";
import { flushSync, mount, unmount } from "svelte";
import ProgressSeek from "./ProgressSeek.svelte";

function renderSeek(onSeek = vi.fn()) {
  const target = document.createElement("div");
  document.body.appendChild(target);
  const instance = mount(ProgressSeek, {
    target,
    props: {
      currentTime: 10,
      duration: 100,
      progressPercent: 10,
      onSeek,
    },
  });
  flushSync();
  const input = target.querySelector(
    ".progress-seek__input",
  ) as HTMLInputElement;
  return {
    target,
    input,
    onSeek,
    cleanup: () => {
      unmount(instance);
      target.remove();
    },
  };
}

function setRangeValue(input: HTMLInputElement, seconds: number) {
  input.value = String(seconds);
  input.dispatchEvent(new Event("input", { bubbles: true }));
}

describe("ProgressSeek", () => {
  it("does not seek while dragging via input events", () => {
    const { input, onSeek, cleanup } = renderSeek();
    try {
      setRangeValue(input, 25);
      setRangeValue(input, 40);
      setRangeValue(input, 55);
      flushSync();
      expect(onSeek).not.toHaveBeenCalled();
    } finally {
      cleanup();
    }
  });

  it("seeks once on change with the preview value", () => {
    const { input, onSeek, cleanup } = renderSeek();
    try {
      setRangeValue(input, 42.5);
      input.dispatchEvent(new Event("change", { bubbles: true }));
      flushSync();
      expect(onSeek).toHaveBeenCalledTimes(1);
      expect(onSeek).toHaveBeenCalledWith(42.5);
    } finally {
      cleanup();
    }
  });

  it("seeks once on pointerup when scrubbing", () => {
    const { input, onSeek, cleanup } = renderSeek();
    try {
      setRangeValue(input, 33);
      input.dispatchEvent(new Event("pointerup", { bubbles: true }));
      flushSync();
      expect(onSeek).toHaveBeenCalledTimes(1);
      expect(onSeek).toHaveBeenCalledWith(33);
    } finally {
      cleanup();
    }
  });

  it("does not double-seek when change and pointerup both fire", () => {
    const { input, onSeek, cleanup } = renderSeek();
    try {
      setRangeValue(input, 60);
      input.dispatchEvent(new Event("change", { bubbles: true }));
      input.dispatchEvent(new Event("pointerup", { bubbles: true }));
      flushSync();
      expect(onSeek).toHaveBeenCalledTimes(1);
      expect(onSeek).toHaveBeenCalledWith(60);
    } finally {
      cleanup();
    }
  });

  it("updates the fill from preview while scrubbing", () => {
    const { target, input, cleanup } = renderSeek();
    try {
      setRangeValue(input, 50);
      flushSync();
      const root = target.querySelector(".progress-seek") as HTMLElement;
      expect(root.style.getPropertyValue("--progress")).toBe("0.5");
    } finally {
      cleanup();
    }
  });
});
