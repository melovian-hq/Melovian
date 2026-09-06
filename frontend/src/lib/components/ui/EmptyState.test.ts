// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { flushSync, mount, unmount } from "svelte";
import EmptyState from "./EmptyState.svelte";

describe("EmptyState mobile containment", () => {
  it("stays within a narrow parent without horizontal overflow", () => {
    const parent = document.createElement("div");
    parent.style.width = "320px";
    parent.style.overflow = "hidden";
    document.body.appendChild(parent);

    const instance = mount(EmptyState, {
      target: parent,
      props: {
        title: "Connection failed",
        message:
          "Could not reach https://very-long-subdomain.example.invalid:4533/api/subsonic/rest/ping.view",
        icon: "alertCircle",
      },
    });
    flushSync();

    try {
      const el = parent.querySelector(".empty-state") as HTMLElement;
      expect(el).not.toBeNull();
      expect(el.scrollWidth).toBeLessThanOrEqual(el.clientWidth + 1);
      expect(getComputedStyle(el).maxWidth).not.toBe("none");
      expect(el.className).toContain("empty-state");
    } finally {
      unmount(instance);
      parent.remove();
    }
  });
});
