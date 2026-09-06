// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { readAPIError, requireOk } from "./errors";

describe("readAPIError", () => {
  it("prefers JSON error field", async () => {
    const response = new Response(
      JSON.stringify({ error: "tls handshake timeout" }),
      {
        status: 400,
        headers: { "Content-Type": "application/json" },
      },
    );
    await expect(
      readAPIError(response, "Connection test failed"),
    ).resolves.toBe("tls handshake timeout");
  });

  it("falls back to plain text", async () => {
    const response = new Response("empty request body", { status: 400 });
    await expect(
      readAPIError(response, "Connection test failed"),
    ).resolves.toBe("empty request body");
  });

  it("uses fallback when body is empty", async () => {
    const response = new Response("", { status: 400 });
    await expect(
      readAPIError(response, "Connection test failed"),
    ).resolves.toBe("Connection test failed");
  });
});

describe("requireOk", () => {
  it("resolves for successful responses", async () => {
    await expect(
      requireOk(new Response("", { status: 200 }), "failed"),
    ).resolves.toBeUndefined();
  });

  it("throws the API error for failed responses", async () => {
    const response = new Response(JSON.stringify({ error: "nope" }), {
      status: 500,
      headers: { "Content-Type": "application/json" },
    });
    await expect(requireOk(response, "failed")).rejects.toThrow("nope");
  });
});
