// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { flushSync, mount, unmount } from "svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  expectNoA11yViolations,
  removeLeftoverPortalNodes,
} from "../../../test-fixtures/a11y";
import Toggle from "./Toggle.svelte";

const cleanups: Array<() => Promise<void> | void> = [];

afterEach(async () => {
  for (const cleanup of cleanups.splice(0)) {
    await cleanup();
  }
  removeLeftoverPortalNodes();
});

function renderToggle(props: Record<string, unknown>) {
  const target = document.createElement("div");
  document.body.appendChild(target);
  const instance = mount(Toggle, { target, props });
  flushSync();
  cleanups.push(async () => {
    await unmount(instance);
    target.remove();
  });
  return target;
}

describe("Toggle accessibility", () => {
  it("has no violations when off", async () => {
    renderToggle({ checked: false, label: "Shuffle", onchange: vi.fn() });
    expect(document.querySelector("[role='switch']")).not.toBeNull();
    await expectNoA11yViolations(document.body);
  });

  it("has no violations when on", async () => {
    renderToggle({ checked: true, label: "Shuffle", onchange: vi.fn() });
    const toggle = document.querySelector("[role='switch']");
    expect(toggle?.getAttribute("aria-checked")).toBe("true");
    await expectNoA11yViolations(document.body);
  });

  it("has no violations when disabled with only an aria label", async () => {
    renderToggle({
      checked: false,
      disabled: true,
      ariaLabel: "Enable offline mode",
      onchange: vi.fn(),
    });
    const toggle = document.querySelector("[role='switch']");
    expect(toggle?.getAttribute("aria-label")).toBe("Enable offline mode");
    await expectNoA11yViolations(document.body);
  });
});
