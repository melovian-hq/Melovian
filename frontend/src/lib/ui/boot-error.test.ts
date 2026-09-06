// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { formatError } from "./boot-error";

describe("formatError", () => {
  it("formats Error instances", () => {
    const err = new Error("boom");
    const details = formatError(err, "test");
    expect(details.message).toBe("boom");
    expect(details.source).toBe("test");
    expect(details.stack).toContain("boom");
  });

  it("formats string errors", () => {
    const details = formatError("load failed", "import");
    expect(details.message).toBe("load failed");
    expect(details.source).toBe("import");
  });
});
