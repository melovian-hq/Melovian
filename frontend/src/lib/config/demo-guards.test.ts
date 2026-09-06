// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { canMutateInDemo } from "./demo-guards";

describe("demo-guards", () => {
  it("allows mutations when demo mode is off", () => {
    expect(canMutateInDemo()).toBe(true);
  });
});
