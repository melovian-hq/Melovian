// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { flushSync, mount, unmount } from "svelte";
import { afterEach, describe, expect, it } from "vitest";
import {
  expectNoA11yViolations,
  removeLeftoverPortalNodes,
} from "../../../test-fixtures/a11y";
import { confirmDialog } from "$lib/ui/confirm.svelte";
import ConfirmDialog from "./ConfirmDialog.svelte";

const cleanups: Array<() => Promise<void> | void> = [];

afterEach(async () => {
  confirmDialog.cancel();
  for (const cleanup of cleanups.splice(0)) {
    await cleanup();
  }
  removeLeftoverPortalNodes();
});

function renderDialog() {
  const target = document.createElement("div");
  document.body.appendChild(target);
  const instance = mount(ConfirmDialog, { target });
  flushSync();
  cleanups.push(async () => {
    await unmount(instance);
    target.remove();
  });
  return target;
}

describe("ConfirmDialog accessibility", () => {
  it("has no violations for an open alertdialog with title and message", async () => {
    renderDialog();
    void confirmDialog.confirm({
      title: "Delete playlist",
      message: "This removes the playlist and cannot be undone.",
      confirmLabel: "Delete",
      danger: true,
    });
    flushSync();
    // Bits layers can take a tick to attach portaled content
    await new Promise((resolve) => setTimeout(resolve, 0));

    const dialog = document.querySelector("[role='alertdialog']");
    expect(dialog).not.toBeNull();
    expect(dialog?.textContent).toContain("Delete playlist");
    await expectNoA11yViolations(document.body);
  });
});
