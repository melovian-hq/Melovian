// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, beforeEach } from "vitest";
import { getActiveInstanceId, setActiveInstanceId } from "./context";

describe("instance context", () => {
  beforeEach(() => {
    setActiveInstanceId(null);
  });

  it("stores and reads active instance id", () => {
    setActiveInstanceId("abc123");
    expect(getActiveInstanceId()).toBe("abc123");
  });

  it("clears active instance id", () => {
    setActiveInstanceId("abc123");
    setActiveInstanceId(null);
    expect(getActiveInstanceId()).toBeNull();
  });
});
