// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { flushSync, mount, unmount } from "svelte";
import { describe, expect, it, vi } from "vitest";
import ContextMenu from "./ContextMenu.svelte";

function renderMenu() {
  const play = vi.fn();
  const queue = vi.fn();
  const target = document.createElement("div");
  document.body.appendChild(target);
  const instance = mount(ContextMenu, {
    target,
    props: {
      x: 20,
      y: 30,
      label: "Album actions",
      onclose: vi.fn(),
      items: [
        { id: "play", label: "Play now", icon: "play", onclick: play },
        {
          id: "queue",
          label: "Add to queue",
          icon: "queueAdd",
          onclick: queue,
        },
      ],
    },
  });
  flushSync();
  return {
    target,
    play,
    queue,
    cleanup: () => {
      void unmount(instance);
      target.remove();
    },
  };
}

describe("ContextMenu", () => {
  it("renders actions and runs the clicked item", () => {
    const { play, cleanup } = renderMenu();
    // Bits UI Portal renders the menu into document.body, not the mount target
    expect(document.body.textContent).toContain("Play now");
    expect(document.body.textContent).toContain("Add to queue");
    const buttons = document.querySelectorAll("[role='menuitem']");
    expect(buttons).toHaveLength(2);
    (buttons[0] as HTMLButtonElement).click();
    expect(play).toHaveBeenCalledTimes(1);
    cleanup();
  });
});
