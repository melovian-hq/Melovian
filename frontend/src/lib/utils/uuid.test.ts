// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";
import { installInsecureContextPolyfills, randomUUID } from "./uuid";

describe("randomUUID", () => {
  it("returns a v4 UUID", () => {
    const id = randomUUID();
    expect(id).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/,
    );
  });

  it("falls back when randomUUID is unavailable in insecure contexts", () => {
    vi.stubGlobal("isSecureContext", false);
    vi.stubGlobal("crypto", {});

    expect(randomUUID()).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/,
    );
  });

  it("polyfills crypto.randomUUID for third-party callers", () => {
    vi.stubGlobal("isSecureContext", false);
    vi.stubGlobal("crypto", {
      getRandomValues: (bytes: Uint8Array) => bytes.fill(9) && bytes,
    });

    installInsecureContextPolyfills();

    expect(typeof globalThis.crypto.randomUUID).toBe("function");
    expect(globalThis.crypto.randomUUID()).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/,
    );
  });
});
