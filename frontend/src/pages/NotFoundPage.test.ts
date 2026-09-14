// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, afterEach } from "vitest";
import { flushSync, mount, unmount } from "svelte";
import NotFoundPage from "./NotFoundPage.svelte";
import { router } from "$lib/router/router.svelte";

describe("NotFoundPage", () => {
  const originalPathname = router.pathname;

  afterEach(() => {
    router.pathname = originalPathname;
  });

  it("renders a not-found state naming the attempted path", () => {
    router.pathname = "/definitely-not-a-page";

    const target = document.createElement("div");
    document.body.appendChild(target);
    const instance = mount(NotFoundPage, { target });
    flushSync();

    expect(target.innerHTML).toContain("Page not found");
    expect(target.innerHTML).toContain("/definitely-not-a-page");

    const home = target.querySelector<HTMLAnchorElement>(
      "a.not-found-page__primary",
    );
    expect(home?.getAttribute("href")).toBe("/music");
    expect(home?.textContent).toContain("Back to music");

    void unmount(instance);
    target.remove();
  });
});
