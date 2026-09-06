// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { playerBottomInset } from "./player-bottom-inset";

describe("playerBottomInset", () => {
  it("uses player bar height plus gap for mini layout", () => {
    const value = playerBottomInset(true, "mini");
    expect(value).toContain("--jb-player-bar-height");
    expect(value).toContain("--jb-space-4");
  });

  it("uses player bar height plus gap for full layout", () => {
    const value = playerBottomInset(true, "full");
    expect(value).toContain("--jb-player-bar-height");
    expect(value).toContain("--jb-space-4");
  });

  it("uses page padding when player is hidden", () => {
    expect(playerBottomInset(false, "full")).toContain("--jb-space-6");
  });

  it("stacks mobile bottom chrome when mobileNav is set", () => {
    const withPlayer = playerBottomInset(true, "full", { mobileNav: true });
    expect(withPlayer).toContain("--jb-bottom-chrome-height");
    expect(withPlayer).toContain("--jb-space-4");

    const withoutPlayer = playerBottomInset(false, "full", {
      mobileNav: true,
    });
    expect(withoutPlayer).toContain("--jb-bottom-chrome-height");
  });
});
