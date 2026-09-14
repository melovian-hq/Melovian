// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from "vitest";
import { reportPlayerBarHeight } from "./player-height";

function fakeRect(height: number, top: number): DOMRect {
  return {
    height,
    top,
    bottom: top + height,
    left: 0,
    right: 0,
    width: 0,
    x: 0,
    y: top,
    toJSON: () => ({}),
  } as DOMRect;
}

describe("reportPlayerBarHeight", () => {
  afterEach(() => {
    document.querySelector(".app-root")?.remove();
  });

  it("writes the rendered bar height onto .app-root", () => {
    const root = document.createElement("div");
    root.className = "app-root";
    document.body.appendChild(root);
    const node = document.createElement("div");
    vi.spyOn(node, "getBoundingClientRect").mockReturnValue(fakeRect(92, 700));

    const action = reportPlayerBarHeight(node);
    expect(root.style.getPropertyValue("--jb-player-bar-height")).toBe("92px");

    action.destroy();
    expect(root.style.getPropertyValue("--jb-player-bar-height")).toBe("");
  });

  it("includes the viewport offset below a floating bar", () => {
    const root = document.createElement("div");
    root.className = "app-root";
    document.body.appendChild(root);
    const node = document.createElement("div");
    // innerHeight is 768 in jsdom. A bar top at 668 covers 100px total.
    vi.spyOn(node, "getBoundingClientRect").mockReturnValue(fakeRect(80, 668));

    const action = reportPlayerBarHeight(node, { bottomOffset: true });
    const covered = window.innerHeight - 668;
    expect(root.style.getPropertyValue("--jb-player-bar-height")).toBe(
      `${covered}px`,
    );
    action.destroy();
  });

  it("does not write a zero or negative height", () => {
    const root = document.createElement("div");
    root.className = "app-root";
    document.body.appendChild(root);
    const node = document.createElement("div");
    vi.spyOn(node, "getBoundingClientRect").mockReturnValue(fakeRect(0, 0));

    const action = reportPlayerBarHeight(node);
    expect(root.style.getPropertyValue("--jb-player-bar-height")).toBe("");
    action.destroy();
  });
});
