// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, beforeEach, afterEach } from "vitest";
import { flushSync, mount, unmount } from "svelte";
import { layout } from "$lib/components/layout/layout.svelte";
import HomeTipsBanner from "./HomeTipsBanner.svelte";

describe("HomeTipsBanner", () => {
  beforeEach(() => {
    localStorage.clear();
    layout.setMobileViewport(false);
  });

  afterEach(() => {
    layout.setMobileViewport(false);
  });

  it("renders on desktop when not dismissed", () => {
    const target = document.createElement("div");
    document.body.appendChild(target);
    const instance = mount(HomeTipsBanner, { target });
    flushSync();
    try {
      expect(target.querySelector(".home-tips")).not.toBeNull();
    } finally {
      unmount(instance);
      target.remove();
    }
  });

  it("hides on mobile viewport", () => {
    layout.setMobileViewport(true);
    const target = document.createElement("div");
    document.body.appendChild(target);
    const instance = mount(HomeTipsBanner, { target });
    flushSync();
    try {
      expect(target.querySelector(".home-tips")).toBeNull();
    } finally {
      unmount(instance);
      target.remove();
    }
  });
});
