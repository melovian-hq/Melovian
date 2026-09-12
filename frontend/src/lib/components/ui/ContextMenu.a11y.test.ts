// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { flushSync, mount, unmount } from "svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  expectNoA11yViolations,
  removeLeftoverPortalNodes,
} from "../../../test-fixtures/a11y";
import ContextMenu from "./ContextMenu.svelte";
import type { ContextMenuEntry } from "./context-menu";

const cleanups: Array<() => Promise<void> | void> = [];

afterEach(async () => {
  for (const cleanup of cleanups.splice(0)) {
    await cleanup();
  }
  removeLeftoverPortalNodes();
});

function renderMenu(items: ContextMenuEntry[], label = "Album actions") {
  const target = document.createElement("div");
  document.body.appendChild(target);
  const instance = mount(ContextMenu, {
    target,
    props: { x: 20, y: 30, label, onclose: vi.fn(), items },
  });
  flushSync();
  cleanups.push(async () => {
    await unmount(instance);
    target.remove();
  });
  return target;
}

describe("ContextMenu accessibility", () => {
  it("has no violations for a menu with items, a disabled item, a separator, and a danger action", async () => {
    renderMenu([
      { id: "play", label: "Play now", icon: "play", onclick: vi.fn() },
      {
        id: "queue",
        label: "Add to queue",
        icon: "queueAdd",
        disabled: true,
        onclick: vi.fn(),
      },
      { id: "sep-1", separator: true },
      {
        id: "delete",
        label: "Delete album",
        danger: true,
        onclick: vi.fn(),
      },
    ]);
    // Bits layers can take a tick to attach portaled content
    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(document.querySelector("[role='menu']")).not.toBeNull();
    expect(
      document.querySelectorAll("[role='menuitem']").length,
    ).toBeGreaterThanOrEqual(3);
    await expectNoA11yViolations(document.body);
  });
});
