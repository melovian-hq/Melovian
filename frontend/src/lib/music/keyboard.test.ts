// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";
import { handleKeydown, type PlaybackControls } from "./keyboard";

function makeControls(): PlaybackControls {
  return {
    togglePlay: vi.fn(),
    next: vi.fn(),
    previous: vi.fn(),
    adjustVolume: vi.fn(),
    seekBy: vi.fn(),
  };
}

function press(
  code: string,
  init: Partial<KeyboardEventInit> & { target?: EventTarget } = {},
): KeyboardEvent {
  const { target, ...rest } = init;
  const event = new KeyboardEvent("keydown", { code, ...rest });
  if (target) Object.defineProperty(event, "target", { value: target });
  return event;
}

describe("handleKeydown", () => {
  it("toggles play on Space", () => {
    const controls = makeControls();
    expect(handleKeydown(press("Space"), controls)).toBe(true);
    expect(controls.togglePlay).toHaveBeenCalledTimes(1);
  });

  it("adjusts volume with arrow up/down", () => {
    const controls = makeControls();
    handleKeydown(press("ArrowUp"), controls);
    handleKeydown(press("ArrowDown"), controls);
    expect(controls.adjustVolume).toHaveBeenNthCalledWith(1, 0.05);
    expect(controls.adjustVolume).toHaveBeenNthCalledWith(2, -0.05);
  });

  it("seeks with arrows and skips tracks with shift", () => {
    const controls = makeControls();
    handleKeydown(press("ArrowRight"), controls);
    expect(controls.seekBy).toHaveBeenCalledWith(5);

    handleKeydown(press("ArrowLeft"), controls);
    expect(controls.seekBy).toHaveBeenCalledWith(-5);

    handleKeydown(press("ArrowRight", { shiftKey: true }), controls);
    expect(controls.next).toHaveBeenCalledTimes(1);

    handleKeydown(press("ArrowLeft", { shiftKey: true }), controls);
    expect(controls.previous).toHaveBeenCalledTimes(1);
  });

  it("ignores shortcuts with modifier keys", () => {
    const controls = makeControls();
    expect(handleKeydown(press("Space", { metaKey: true }), controls)).toBe(
      false,
    );
    expect(controls.togglePlay).not.toHaveBeenCalled();
  });

  it("ignores shortcuts while typing in an input", () => {
    const controls = makeControls();
    const input = document.createElement("input");
    expect(handleKeydown(press("Space", { target: input }), controls)).toBe(
      false,
    );
    expect(controls.togglePlay).not.toHaveBeenCalled();
  });

  it("opens command palette with ctrl+k", () => {
    const controls = {
      ...makeControls(),
      togglePalette: vi.fn(),
    };
    expect(
      handleKeydown(press("KeyK", { ctrlKey: true, key: "k" }), controls),
    ).toBe(true);
    expect(controls.togglePalette).toHaveBeenCalledTimes(1);
  });

  it("closes palette with escape when open", () => {
    const controls = {
      ...makeControls(),
      closePalette: vi.fn(),
    };
    expect(
      handleKeydown(press("Escape"), controls, { paletteOpen: true }),
    ).toBe(true);
    expect(controls.closePalette).toHaveBeenCalledTimes(1);
  });

  it("toggles help with question mark and closes with escape", () => {
    const controls = {
      ...makeControls(),
      toggleHelp: vi.fn(),
      closeHelp: vi.fn(),
    };
    expect(
      handleKeydown(press("Slash", { shiftKey: true, key: "?" }), controls),
    ).toBe(true);
    expect(controls.toggleHelp).toHaveBeenCalledTimes(1);

    expect(handleKeydown(press("Escape"), controls, { helpOpen: true })).toBe(
      true,
    );
    expect(controls.closeHelp).toHaveBeenCalledTimes(1);
  });
});
