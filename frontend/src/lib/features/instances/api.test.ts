// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { readErrorMessage } from "./api";

describe("readErrorMessage", () => {
  it("prefers JSON error field", async () => {
    const response = new Response(
      JSON.stringify({ error: "tls handshake timeout" }),
      {
        status: 400,
        headers: { "Content-Type": "application/json" },
      },
    );
    await expect(
      readErrorMessage(response, "Connection test failed"),
    ).resolves.toBe("tls handshake timeout");
  });

  it("falls back to plain text", async () => {
    const response = new Response("empty request body", { status: 400 });
    await expect(
      readErrorMessage(response, "Connection test failed"),
    ).resolves.toBe("empty request body");
  });

  it("uses fallback when body is empty", async () => {
    const response = new Response("", { status: 400 });
    await expect(
      readErrorMessage(response, "Connection test failed"),
    ).resolves.toBe("Connection test failed");
  });
});
