// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  contextMenuPositionFromEvent,
  getPlayerBarInsetPx,
  isContextMenuItem,
  isEditableContextTarget,
} from "./context-menu";

describe("context menu helpers", () => {
  it("returns zero player inset when player chrome is hidden", () => {
    expect(getPlayerBarInsetPx()).toBe(0);
  });

  it("treats separators as non-items", () => {
    expect(isContextMenuItem({ id: "sep", separator: true })).toBe(false);
    expect(
      isContextMenuItem({
        id: "play",
        label: "Play now",
        onclick: () => {},
      }),
    ).toBe(true);
  });

  it("ignores editable fields for custom menus", () => {
    const input = document.createElement("input");
    document.body.appendChild(input);
    expect(isEditableContextTarget(input)).toBe(true);
    input.remove();

    const button = document.createElement("button");
    expect(isEditableContextTarget(button)).toBe(false);
  });

  it("captures a position from a contextmenu event", () => {
    const button = document.createElement("button");
    const event = new MouseEvent("contextmenu", {
      clientX: 40,
      clientY: 80,
      bubbles: true,
      cancelable: true,
    });
    Object.defineProperty(event, "target", { value: button });
    const pos = contextMenuPositionFromEvent(event);
    expect(event.defaultPrevented).toBe(true);
    expect(pos).toEqual({ x: 40, y: 80 });
  });
});
