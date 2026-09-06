// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, beforeEach, afterEach } from "vitest";
import { flushSync, mount, unmount } from "svelte";
import { layout } from "$lib/components/layout/layout.svelte";
import { toast } from "$lib/ui/toast.svelte";
import ToastContainer from "./ToastContainer.svelte";

describe("ToastContainer mobile position", () => {
  beforeEach(() => {
    for (const item of [...toast.items]) {
      toast.dismiss(item.id);
    }
    layout.setMobileViewport(false);
  });

  afterEach(() => {
    layout.setMobileViewport(false);
    for (const item of [...toast.items]) {
      toast.dismiss(item.id);
    }
  });

  it("anchors toasts at the top on mobile", () => {
    layout.setMobileViewport(true);
    toast.info("Hello", { duration: 0 });
    const target = document.createElement("div");
    document.body.appendChild(target);
    const instance = mount(ToastContainer, { target });
    flushSync();
    try {
      const container = target.querySelector(".toast-container") as HTMLElement;
      expect(container).not.toBeNull();
      expect(container.classList.contains("toast-container--top")).toBe(true);
      expect(container.style.top).toContain("safe-area-inset-top");
      expect(container.style.bottom).toBe("");
    } finally {
      unmount(instance);
      target.remove();
    }
  });

  it("keeps bottom anchoring on desktop", () => {
    layout.setMobileViewport(false);
    toast.info("Hello", { duration: 0 });
    const target = document.createElement("div");
    document.body.appendChild(target);
    const instance = mount(ToastContainer, { target });
    flushSync();
    try {
      const container = target.querySelector(".toast-container") as HTMLElement;
      expect(container.classList.contains("toast-container--top")).toBe(false);
      expect(container.style.bottom).not.toBe("");
      expect(container.style.top).toBe("");
    } finally {
      unmount(instance);
      target.remove();
    }
  });
});
