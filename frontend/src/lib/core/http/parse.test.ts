// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import * as v from "valibot";
import { ApiParseError, parseJson, parsePayload } from "./parse";

const itemSchema = v.looseObject({
  id: v.string(),
  count: v.optional(v.number()),
});

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

describe("parseJson", () => {
  it("returns the parsed output for a matching body", async () => {
    const result = await parseJson(
      itemSchema,
      jsonResponse({ id: "a1", count: 3, extra: "kept" }),
      "test item",
    );
    expect(result.id).toBe("a1");
    expect(result.count).toBe(3);
    expect(result).toMatchObject({ extra: "kept" });
  });

  it("throws ApiParseError when a required field is missing", async () => {
    const promise = parseJson(
      itemSchema,
      jsonResponse({ count: 1 }),
      "test item",
    );
    await expect(promise).rejects.toBeInstanceOf(ApiParseError);
    await expect(promise).rejects.toThrow(/test item/);
  });

  it("throws ApiParseError when a field has the wrong type", async () => {
    await expect(
      parseJson(itemSchema, jsonResponse({ id: 42 }), "test item"),
    ).rejects.toBeInstanceOf(ApiParseError);
  });

  it("rejects non-object bodies", async () => {
    await expect(
      parseJson(itemSchema, jsonResponse([1, 2]), "test item"),
    ).rejects.toBeInstanceOf(ApiParseError);
    await expect(
      parseJson(itemSchema, jsonResponse(null), "test item"),
    ).rejects.toBeInstanceOf(ApiParseError);
  });

  it("lets JSON syntax errors propagate", async () => {
    const response = new Response("not json", { status: 200 });
    await expect(parseJson(itemSchema, response)).rejects.toThrow();
  });
});

describe("parsePayload", () => {
  it("parses an already-decoded value", () => {
    expect(parsePayload(itemSchema, { id: "x" })).toEqual({ id: "x" });
  });

  it("exposes issues on the thrown error", () => {
    try {
      parsePayload(itemSchema, { id: 7 });
      expect.unreachable();
    } catch (err) {
      expect(err).toBeInstanceOf(ApiParseError);
      expect((err as ApiParseError).issues.length).toBeGreaterThan(0);
    }
  });
});

describe("ApiParseError", () => {
  it("is an Error with a stable name", () => {
    const result = v.safeParse(v.string(), 42);
    expect(result.success).toBe(false);
    if (result.success) return;
    const err = new ApiParseError("thing", result.issues);
    expect(err).toBeInstanceOf(Error);
    expect(err.name).toBe("ApiParseError");
    expect(err.message).toContain("thing");
    expect(err.issues).toHaveLength(1);
  });
});
