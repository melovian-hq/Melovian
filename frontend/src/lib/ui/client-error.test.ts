// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { mapClientError } from "./client-error";

describe("mapClientError", () => {
  it("maps network failures", () => {
    const view = mapClientError(new Error("Failed to fetch"));
    expect(view.message).toContain("Could not reach the server");
  });

  it("maps auth failures", () => {
    const view = mapClientError(new Error("401 Unauthorized"));
    expect(view.message).toContain("Sign-in failed");
  });

  it("returns short messages unchanged", () => {
    const view = mapClientError(new Error("Scan failed"));
    expect(view.message).toBe("Scan failed");
  });
});
