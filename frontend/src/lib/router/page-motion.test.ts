// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { routeViewKey } from "./page-motion";

describe("routeViewKey", () => {
  it("collapses settings shell routes to one key", () => {
    expect(routeViewKey("/settings", {})).toBe("settings-shell");
    expect(routeViewKey("/settings/:tab", { tab: "playback" })).toBe(
      "settings-shell",
    );
    expect(routeViewKey("/instances", {})).toBe("settings-shell");
  });

  it("includes params for detail routes", () => {
    expect(routeViewKey("/music/album/:albumId", { albumId: "a1" })).toBe(
      "/music/album/:albumId:albumId=a1",
    );
    expect(routeViewKey("/music/album/:albumId", { albumId: "a2" })).toBe(
      "/music/album/:albumId:albumId=a2",
    );
  });

  it("uses the path alone when there are no params", () => {
    expect(routeViewKey("/music/artists", {})).toBe("/music/artists");
  });
});
