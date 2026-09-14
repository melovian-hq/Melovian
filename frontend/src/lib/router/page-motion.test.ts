// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  captureViewSnapshot,
  clipInsets,
  pinOutgoingPage,
  resetPageScroll,
  routeViewKey,
  scrollerClipBox,
  type PinnedViewSnapshot,
} from "./page-motion";

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

describe("clipInsets", () => {
  it("returns zero insets when the view sits inside the clip", () => {
    const rect = { top: 10, right: 100, bottom: 90, left: 10 };
    const clip = { top: 0, right: 120, bottom: 100, left: 0 };
    expect(clipInsets(rect, clip)).toEqual({
      top: 0,
      right: 0,
      bottom: 0,
      left: 0,
    });
  });

  it("clamps the parts of the view outside the clip", () => {
    const rect = { top: -40, right: 200, bottom: 300, left: -5 };
    const clip = { top: 0, right: 160, bottom: 200, left: 0 };
    expect(clipInsets(rect, clip)).toEqual({
      top: 40,
      right: 40,
      bottom: 100,
      left: 5,
    });
  });

  it("never returns negative insets", () => {
    const rect = { top: 50, right: 50, bottom: 50, left: 50 };
    const clip = { top: 0, right: 500, bottom: 500, left: 0 };
    const inset = clipInsets(rect, clip);
    for (const value of Object.values(inset)) {
      expect(value).toBeGreaterThanOrEqual(0);
    }
  });
});

describe("scrollerClipBox", () => {
  it("returns null outside a shell scroll container", () => {
    const node = document.createElement("div");
    document.body.appendChild(node);
    expect(scrollerClipBox(node)).toBeNull();
    node.remove();
  });

  it("returns a clip box inside the shell scroller", () => {
    const scroller = document.createElement("div");
    scroller.className = "app-shell__content";
    const node = document.createElement("div");
    scroller.appendChild(node);
    document.body.appendChild(scroller);
    const clip = scrollerClipBox(node);
    expect(clip).not.toBeNull();
    expect(clip?.right).toBeGreaterThanOrEqual(clip!.left);
    expect(clip?.bottom).toBeGreaterThanOrEqual(clip!.top);
    scroller.remove();
  });
});

describe("captureViewSnapshot", () => {
  it("records the shell scroller position in the snapshot", () => {
    const scroller = document.createElement("div");
    scroller.className = "app-shell__content";
    const view = document.createElement("div");
    scroller.appendChild(view);
    document.body.appendChild(scroller);
    scroller.scrollTop = 240;
    const snapshot = captureViewSnapshot(view);
    expect(snapshot.scrollTop).toBe(240);
    scroller.remove();
  });

  it("falls back to zero scroll outside a shell scroll container", () => {
    const view = document.createElement("div");
    document.body.appendChild(view);
    expect(captureViewSnapshot(view).scrollTop).toBe(0);
    view.remove();
  });

  it("falls back to the view rect as its own clip outside a shell", () => {
    const view = document.createElement("div");
    document.body.appendChild(view);
    const snapshot = captureViewSnapshot(view);
    expect(snapshot.view).toMatchObject({
      top: 0,
      right: 0,
      bottom: 0,
      left: 0,
      width: 0,
      height: 0,
    });
    expect(snapshot.clip).toEqual({
      top: 0,
      right: 0,
      bottom: 0,
      left: 0,
    });
    view.remove();
  });
});

describe("pinOutgoingPage", () => {
  const snapshot: PinnedViewSnapshot = {
    view: {
      top: 80,
      right: 800,
      bottom: 600,
      left: 256,
      width: 544,
      height: 520,
    },
    clip: { top: 80, right: 800, bottom: 600, left: 256 },
  };

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

  it("freezes the page at the snapshot viewport rect", () => {
    const page = document.createElement("div");
    document.body.appendChild(page);
    pinOutgoingPage(page, snapshot);
    expect(page.style.position).toBe("fixed");
    expect(page.style.top).toBe("80px");
    expect(page.style.left).toBe("256px");
    expect(page.style.width).toBe("544px");
    expect(page.style.height).toBe("520px");
    expect(page.style.pointerEvents).toBe("none");
    page.remove();
  });

  it("clips the pinned page to the snapshot clip box", () => {
    const page = document.createElement("div");
    document.body.appendChild(page);
    pinOutgoingPage(page, {
      view: {
        top: -100,
        right: 800,
        bottom: 900,
        left: 256,
        width: 544,
        height: 1000,
      },
      clip: snapshot.clip,
    });
    expect(page.style.clipPath).toBe("inset(180px 0px 300px 0px)");
    page.remove();
  });

  it("shifts the pin by the scroll delta since the snapshot", () => {
    // The snapshot rect is viewport-relative. Scrolling 240px after the
    // capture moved the painted page up by 240px, so the pin must land at
    // 80 - 240, not the stale 80.
    const { scroller, page } = scrolledShell(240);
    pinOutgoingPage(page, {
      view: {
        top: 80,
        right: 800,
        bottom: 600,
        left: 256,
        width: 544,
        height: 520,
      },
      clip: { top: 80, right: 800, bottom: 600, left: 256 },
      scrollTop: 0,
    });
    expect(page.style.top).toBe("-160px");
    expect(page.style.left).toBe("256px");
    expect(page.style.clipPath).toBe("inset(240px 0px 0px 0px)");
    scroller.remove();
  });

  it("does not shift the pin when the scroll position is unchanged", () => {
    const { scroller, page } = scrolledShell(120);
    pinOutgoingPage(page, { ...snapshot, scrollTop: 120 });
    expect(page.style.top).toBe("80px");
    scroller.remove();
  });

  it("falls back to measuring the node without a snapshot", () => {
    const { scroller, page } = scrolledShell(240);
    pinOutgoingPage(page);
    expect(page.style.position).toBe("fixed");
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
