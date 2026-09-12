// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { flushSync, mount, unmount } from "svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  expectNoA11yViolations,
  removeLeftoverPortalNodes,
} from "../../../test-fixtures/a11y";
import Select from "./Select.svelte";

const OPTIONS = [
  { value: "newest", label: "Newest first" },
  { value: "oldest", label: "Oldest first" },
  { value: "random", label: "Random", disabled: true },
];

const cleanups: Array<() => Promise<void> | void> = [];

afterEach(async () => {
  for (const cleanup of cleanups.splice(0)) {
    await cleanup();
  }
  removeLeftoverPortalNodes();
});

function renderSelect(props: Record<string, unknown> = {}) {
  const target = document.createElement("div");
  document.body.appendChild(target);
  const instance = mount(Select, {
    target,
    props: {
      value: "newest",
      options: OPTIONS,
      ariaLabel: "Sort order",
      onchange: vi.fn(),
      ...props,
    },
  });
  flushSync();
  cleanups.push(async () => {
    await unmount(instance);
    target.remove();
  });
  return target;
}

describe("Select accessibility", () => {
  it("has no violations in the closed state", async () => {
    const target = renderSelect();
    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(target.querySelector("[aria-haspopup='listbox']")).not.toBeNull();
    await expectNoA11yViolations(document.body);
  });

  it("has no violations in the open state", async () => {
    const target = renderSelect();
    const trigger = target.querySelector<HTMLElement>(
      "[aria-haspopup='listbox']",
    );
    expect(trigger).not.toBeNull();
    // jsdom has no PointerEvent. Bits opens the select on a left
    // pointerdown, which a bubbling MouseEvent satisfies.
    trigger?.dispatchEvent(
      new MouseEvent("pointerdown", { bubbles: true, button: 0 }),
    );
    flushSync();

    await vi.waitFor(() => {
      expect(document.querySelector("[role='listbox']")).not.toBeNull();
    });
    expect(document.querySelectorAll("[role='option']")).toHaveLength(
      OPTIONS.length,
    );
    await expectNoA11yViolations(document.body);
  });
});
