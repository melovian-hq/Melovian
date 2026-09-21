// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";
import {
  contextMenu,
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

describe("contextMenu action", () => {
  function mount(handler: (pos: { x: number; y: number }) => void) {
    const node = document.createElement("div");
    document.body.appendChild(node);
    const action = contextMenu(node, handler);
    return {
      node,
      destroy: () => {
        action.destroy?.();
        node.remove();
      },
    };
  }

  it("opens on contextmenu events", () => {
    const handler = vi.fn();
    const { node, destroy } = mount(handler);
    node.dispatchEvent(
      new MouseEvent("contextmenu", { clientX: 5, clientY: 7, bubbles: true }),
    );
    expect(handler).toHaveBeenCalledWith({ x: 5, y: 7 });
    destroy();
  });

  it("opens after a 500ms long-press and suppresses the trailing click", () => {
    vi.useFakeTimers();
    const handler = vi.fn();
    const { node, destroy } = mount(handler);
    const start = new Event("touchstart", { bubbles: true });
    Object.defineProperty(start, "touches", {
      value: [{ clientX: 30, clientY: 40 }],
    });
    node.dispatchEvent(start);
    vi.advanceTimersByTime(499);
    expect(handler).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(handler).toHaveBeenCalledWith({ x: 30, y: 40 });

    const click = new MouseEvent("click", { bubbles: true, cancelable: true });
    node.dispatchEvent(click);
    expect(click.defaultPrevented).toBe(true);
    destroy();
    vi.useRealTimers();
  });

  it("cancels the long-press when the finger moves", () => {
    vi.useFakeTimers();
    const handler = vi.fn();
    const { node, destroy } = mount(handler);
    const start = new Event("touchstart", { bubbles: true });
    Object.defineProperty(start, "touches", {
      value: [{ clientX: 10, clientY: 10 }],
    });
    node.dispatchEvent(start);
    const move = new Event("touchmove", { bubbles: true });
    Object.defineProperty(move, "touches", {
      value: [{ clientX: 40, clientY: 10 }],
    });
    node.dispatchEvent(move);
    vi.advanceTimersByTime(1000);
    expect(handler).not.toHaveBeenCalled();
    destroy();
    vi.useRealTimers();
  });

  it("skips long-press on editable targets", () => {
    vi.useFakeTimers();
    const handler = vi.fn();
    const { node, destroy } = mount(handler);
    const input = document.createElement("input");
    node.appendChild(input);
    const start = new Event("touchstart", { bubbles: true });
    Object.defineProperty(start, "touches", {
      value: [{ clientX: 10, clientY: 10 }],
    });
    Object.defineProperty(start, "target", { value: input });
    node.dispatchEvent(start);
    vi.advanceTimersByTime(1000);
    expect(handler).not.toHaveBeenCalled();
    destroy();
    vi.useRealTimers();
  });
});
