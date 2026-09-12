// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { pinOutgoingPage, resetPageScroll, routeViewKey } from "./page-motion";

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

describe("pinOutgoingPage", () => {
  function scrolledShell(scrollTop: number) {
    const scroller = document.createElement("div");
    scroller.className = "app-shell__content";
    const outlet = document.createElement("div");
    outlet.className = "route-outlet";
    const page = document.createElement("div");
    page.className = "route-page";
    outlet.appendChild(page);
    scroller.appendChild(outlet);
    document.body.appendChild(scroller);
    scroller.scrollTop = scrollTop;
    return { scroller, page };
  }

  it("offsets the pinned page by the shell scroll position", () => {
    const { scroller, page } = scrolledShell(240);
    pinOutgoingPage(page);
    expect(page.style.position).toBe("absolute");
    expect(page.style.top).toBe("-240px");
    scroller.remove();
  });

  it("pins at the top when the shell is not scrolled", () => {
    const { scroller, page } = scrolledShell(0);
    pinOutgoingPage(page);
    expect(page.style.top).toBe("0px");
    scroller.remove();
  });

  it("pins at the top outside a shell scroll container", () => {
    const page = document.createElement("div");
    document.body.appendChild(page);
    pinOutgoingPage(page);
    expect(page.style.top).toBe("0px");
    page.remove();
  });
});

describe("resetPageScroll", () => {
  it("clears scroll on the enclosing shell container", () => {
    const scroller = document.createElement("div");
    scroller.className = "app-shell__content";
    const page = document.createElement("div");
    scroller.appendChild(page);
    document.body.appendChild(scroller);
    scroller.scrollTop = 300;
    scroller.scrollLeft = 12;
    resetPageScroll(page);
    expect(scroller.scrollTop).toBe(0);
    expect(scroller.scrollLeft).toBe(0);
    scroller.remove();
  });
});
