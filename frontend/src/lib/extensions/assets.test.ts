// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { extensionAssetUrl, isSafeExtensionAssetPath } from "./assets";
import { clearExtensionStylesForTests, syncExtensionStyles } from "./styles";

describe("extensionAssetUrl", () => {
  it("builds API asset URLs", () => {
    expect(extensionAssetUrl("demo-pack", "assets/icon.png")).toBe(
      "/api/extensions/demo-pack/assets/assets/icon.png",
    );
  });

  it("rejects traversal", () => {
    expect(isSafeExtensionAssetPath("../secret")).toBe(false);
    expect(extensionAssetUrl("x", "../secret")).toBe("");
  });
});

describe("syncExtensionStyles", () => {
  it("injects and removes link tags", () => {
    clearExtensionStylesForTests();
    syncExtensionStyles([
      {
        id: "demo-pack",
        styles: ["assets/theme-app.css"],
      },
    ]);
    const links = document.querySelectorAll(
      "link[data-melovian-extension-style]",
    );
    expect(links).toHaveLength(1);
    expect(links[0]?.getAttribute("href")).toBe(
      "/api/extensions/demo-pack/assets/assets/theme-app.css",
    );

    syncExtensionStyles([]);
    expect(
      document.querySelectorAll("link[data-melovian-extension-style]"),
    ).toHaveLength(0);
  });
});
