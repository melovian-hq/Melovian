// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from "vitest";
import { getConnectionSettings, getEqSettings, getMusicStatus } from "./api";
import { defaultConnectionSettings } from "./connection-settings";

function mockFetchResponse(init: {
  ok: boolean;
  status?: number;
  json?: () => Promise<unknown>;
}) {
  return {
    ok: init.ok,
    status: init.status ?? (init.ok ? 200 : 500),
    headers: new Headers(),
    json: init.json ?? (async () => ({})),
  };
}

describe("music api helpers", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("returns offline music status when request fails", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(mockFetchResponse({ ok: false, status: 401 })),
    );

    await expect(getMusicStatus()).resolves.toEqual({
      enabled: false,
      connected: false,
      error: "Music service unavailable",
    });
  });

  it("merges partial connection settings with defaults", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        mockFetchResponse({
          ok: true,
          json: async () => ({ minDelayMs: 4000 }),
        }),
      ),
    );

    await expect(getConnectionSettings()).resolves.toEqual({
      ...defaultConnectionSettings(),
      minDelayMs: 4000,
    });
  });

  it("returns null for empty connection settings payload", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        mockFetchResponse({
          ok: true,
          json: async () => ({}),
        }),
      ),
    );

    await expect(getConnectionSettings()).resolves.toBeNull();
  });

  it("normalizes eq settings from the server", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        mockFetchResponse({
          ok: true,
          json: async () => ({
            enabled: true,
            presetId: "flat",
            bands: Array(10).fill({ frequency: 1000, gain: 0, q: 1 }),
          }),
        }),
      ),
    );

    const settings = await getEqSettings();
    expect(settings?.enabled).toBe(true);
    expect(settings?.bands).toHaveLength(10);
  });

  it("returns null for malformed eq settings", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        mockFetchResponse({
          ok: true,
          json: async () => ({ enabled: true }),
        }),
      ),
    );

    await expect(getEqSettings()).resolves.toBeNull();
  });
});
