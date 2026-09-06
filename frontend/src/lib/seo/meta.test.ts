// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it } from "vitest";
import { APP_NAME, APP_DESCRIPTION } from "$lib/brand";
import { applyPageMeta, metaFromRoute } from "./meta";

describe("seo meta", () => {
  beforeEach(() => {
    document.head.innerHTML = "";
    document.title = "";
  });

  it("builds route meta with fallbacks", () => {
    const meta = metaFromRoute(
      { path: "/music/albums", title: "Albums", description: "Browse albums." },
      "/music/albums",
    );
    expect(meta.title).toBe("Albums");
    expect(meta.description).toBe("Browse albums.");
    expect(meta.path).toBe("/music/albums");
  });

  it("applies title description and open graph tags", () => {
    applyPageMeta({
      title: "Albums",
      description: "Browse albums in the library.",
      path: "/music/albums",
      image: "/og.png",
    });

    expect(document.title).toBe(`Albums · ${APP_NAME}`);
    expect(
      document.head
        .querySelector('meta[name="description"]')
        ?.getAttribute("content"),
    ).toBe("Browse albums in the library.");
    expect(
      document.head
        .querySelector('meta[property="og:title"]')
        ?.getAttribute("content"),
    ).toBe(`Albums · ${APP_NAME}`);
    expect(
      document.head
        .querySelector('meta[property="og:image"]')
        ?.getAttribute("content"),
    ).toContain("/og.png");
    expect(
      document.head
        .querySelector('link[rel="canonical"]')
        ?.getAttribute("href"),
    ).toContain("/music/albums");
  });

  it("falls back to the app description", () => {
    applyPageMeta({ title: APP_NAME });
    expect(
      document.head
        .querySelector('meta[name="description"]')
        ?.getAttribute("content"),
    ).toBe(APP_DESCRIPTION);
  });
});
