// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from "vitest";
import { flushSync, mount, unmount } from "svelte";

vi.mock("$lib/components/layout/Sidebar.svelte", async () => ({
  default: (await import("../../../test-fixtures/SidebarStub.svelte")).default,
}));

vi.mock("$lib/components/layout/TopBar.svelte", async () => ({
  default: (await import("../../../test-fixtures/TopBarStub.svelte")).default,
}));

vi.mock("$lib/components/layout/BottomNav.svelte", async () => ({
  default: (await import("../../../test-fixtures/ChromeStub.svelte")).default,
}));

vi.mock("$lib/components/ui/OfflineBanner.svelte", async () => ({
  default: (await import("../../../test-fixtures/ChromeStub.svelte")).default,
}));

vi.mock("$lib/config/music.svelte", () => ({
  music: {
    currentTrack: null,
    playerVisible: false,
    playerLayout: "full",
  },
}));

vi.mock("$lib/extensions/features.svelte", () => ({
  extensionFeatures: {
    availableThemes: [],
    syncFromTrackDecoration: vi.fn(),
  },
}));

vi.mock("$lib/extensions/registry", () => ({
  decorateTrack: vi.fn(() => ({})),
}));

vi.mock("$lib/core/logger", () => ({
  logClientError: vi.fn(),
}));

import { router } from "$lib/router/router.svelte";
import type { RouteDefinition } from "$lib/router/router.svelte";
import ShellHarness from "../../../test-fixtures/ShellHarness.svelte";
import RoutePage from "../../../test-fixtures/RoutePage.svelte";

const routes: RouteDefinition[] = [
  { path: "/a", component: RoutePage, content: "compact" },
  { path: "/b", component: RoutePage },
  { path: "/c", component: RoutePage, content: "fill" },
  { path: "/login", component: RoutePage, bare: true },
  {
    path: "/lazy",
    load: () => Promise.resolve({ default: RoutePage }),
  },
];

function mountHarness() {
  const target = document.createElement("div");
  document.body.appendChild(target);
  const instance = mount(ShellHarness, {
    target,
    props: { routes },
  });
  flushSync();
  return {
    target,
    cleanup: () => {
      void unmount(instance);
      target.remove();
    },
  };
}

describe("app shell persistence", () => {
  beforeEach(() => {
    router.pathname = "/a";
    router.search = "";
  });

  it("keeps the sidebar and content scroller mounted across navigation", async () => {
    const { target, cleanup } = mountHarness();
    const shell = target.querySelector(".app-shell");
    const sidebar = target.querySelector(".sidebar-stub");
    const topbar = target.querySelector(".topbar-stub");
    const content = target.querySelector(".app-shell__content");

    expect(shell).not.toBeNull();
    expect(sidebar).not.toBeNull();
    expect(topbar).not.toBeNull();
    expect(target.querySelector(".route-page-fixture")?.textContent).toBe("/a");

    router.navigate("/b");
    flushSync();
    await vi.waitFor(() => {
      expect(target.querySelector(".route-page-fixture")?.textContent).toBe(
        "/b",
      );
    });

    expect(target.querySelector(".app-shell")).toBe(shell);
    expect(target.querySelector(".sidebar-stub")).toBe(sidebar);
    expect(target.querySelector(".topbar-stub")).toBe(topbar);
    expect(target.querySelector(".app-shell__content")).toBe(content);
    expect(target.querySelectorAll(".sidebar-stub").length).toBe(1);
    cleanup();
  });

  it("drops chrome on bare routes without remounting the shell", () => {
    const { target, cleanup } = mountHarness();
    const shell = target.querySelector(".app-shell");
    const main = target.querySelector(".app-shell__main");

    router.navigate("/login");
    flushSync();

    expect(target.querySelector(".app-shell")).toBe(shell);
    expect(target.querySelector(".app-shell__main")).toBe(main);
    expect(target.querySelector(".sidebar-stub")).toBeNull();
    expect(target.querySelector(".topbar-stub")).toBeNull();
    expect(target.querySelector(".app-shell__content--bare")).not.toBeNull();
    cleanup();
  });

  it("restores chrome when navigating back from a bare route", () => {
    const { target, cleanup } = mountHarness();
    router.navigate("/login");
    flushSync();
    expect(target.querySelector(".sidebar-stub")).toBeNull();

    router.navigate("/a");
    flushSync();

    expect(target.querySelector(".sidebar-stub")).not.toBeNull();
    expect(target.querySelector(".topbar-stub")).not.toBeNull();
    expect(target.querySelector(".app-shell__content--bare")).toBeNull();
    cleanup();
  });

  it("applies the route content layout class", () => {
    const { target, cleanup } = mountHarness();
    expect(target.querySelector(".app-shell__content--compact")).not.toBeNull();

    router.navigate("/b");
    flushSync();
    expect(target.querySelector(".app-shell__content--compact")).toBeNull();
    expect(target.querySelector(".app-shell__content--fill")).toBeNull();

    router.navigate("/c");
    flushSync();
    expect(target.querySelector(".app-shell__content--fill")).not.toBeNull();
    cleanup();
  });

  it("resets the shell scroll position when the page swaps", async () => {
    const { target, cleanup } = mountHarness();
    const content = target.querySelector(".app-shell__content") as HTMLElement;
    content.scrollTop = 240;

    router.navigate("/b");
    flushSync();
    await vi.waitFor(() => {
      expect(target.querySelector(".route-page-fixture")?.textContent).toBe(
        "/b",
      );
    });
    await new Promise((r) => setTimeout(r, 50));

    expect(content.scrollTop).toBe(0);
    cleanup();
  });

  it("keeps the shell mounted while a lazy route loads", async () => {
    const { target, cleanup } = mountHarness();
    const sidebar = target.querySelector(".sidebar-stub");

    router.navigate("/lazy");
    flushSync();
    await vi.waitFor(() => {
      expect(target.querySelector(".route-page-fixture")?.textContent).toBe(
        "/lazy",
      );
    });

    expect(target.querySelector(".sidebar-stub")).toBe(sidebar);
    cleanup();
  });
});
