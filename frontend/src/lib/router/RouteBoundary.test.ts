// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";
import { flushSync, mount, unmount } from "svelte";

vi.mock("$lib/core/logger", async (importOriginal) => {
  const actual = await importOriginal<typeof import("$lib/core/logger")>();
  return { ...actual, logClientError: vi.fn() };
});

import { logClientError } from "$lib/core/logger";
import BoundaryHarness from "./test-fixtures/BoundaryHarness.svelte";

describe("RouteBoundary", () => {
  it("renders the error fallback when a child throws and logs it", () => {
    const target = document.createElement("div");
    document.body.appendChild(target);
    const instance = mount(BoundaryHarness, { target });
    flushSync();

    expect(target.innerHTML).toContain("This page hit a problem");
    expect(target.innerHTML).toContain("Route render exploded");
    expect(target.innerHTML).toContain("Try again");
    expect(target.innerHTML).toContain("Back to music");
    expect(logClientError).toHaveBeenCalledWith(expect.any(Error), "route");

    void unmount(instance);
    target.remove();
  });
});
