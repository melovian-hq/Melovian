// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";
import { flushSync, mount, unmount } from "svelte";

vi.mock("@bindings/melovian/services/index.js", () => ({
  MediaService: {
    UpdatePlayback: vi.fn().mockResolvedValue(undefined),
    PollMediaAction: vi.fn().mockResolvedValue({ action: "", seekMs: 0 }),
  },
}));

vi.mock("$lib/config/runtime", () => ({
  loadRuntimeConfig: vi.fn().mockResolvedValue(undefined),
  isFakeCatalog: vi.fn().mockReturnValue(false),
  nativeDesktopAvailable: vi.fn().mockReturnValue(false),
}));

vi.mock("$lib/theme/theme.svelte", () => ({
  theme: {
    mode: "dark",
    resolved: "dark",
    setMode: vi.fn(),
    toggle: vi.fn(),
  },
}));

vi.mock("$lib/features/instances/api", () => ({
  listInstances: vi.fn().mockResolvedValue([]),
  getActiveInstance: vi.fn().mockResolvedValue(null),
  createInstance: vi.fn(),
  updateInstance: vi.fn(),
  testInstance: vi.fn(),
  activateInstance: vi.fn(),
  deleteInstance: vi.fn(),
}));

import App from "./App.svelte";

describe("App mount", () => {
  it("renders without crashing", async () => {
    const target = document.createElement("div");
    target.id = "app";
    document.body.appendChild(target);
    const instance = mount(App, { target });
    flushSync();
    await new Promise((r) => setTimeout(r, 200));
    flushSync();
    expect(target.innerHTML.length).toBeGreaterThan(0);
    unmount(instance);
    target.remove();
  });
});
