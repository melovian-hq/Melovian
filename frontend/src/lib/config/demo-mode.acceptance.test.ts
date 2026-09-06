// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";
import { canMutateInDemo } from "$lib/config/demo-guards";

vi.mock("$lib/config/runtime", async (importOriginal) => {
  const actual = await importOriginal<typeof import("$lib/config/runtime")>();
  return {
    ...actual,
    isDemoMode: () => true,
  };
});

describe("demo-mode acceptance", () => {
  it("accepts demo mode blocking mutations", () => {
    expect(canMutateInDemo()).toBe(false);
  });
});
