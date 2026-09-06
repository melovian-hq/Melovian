// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { fetchWithRetry, isTransientNetworkError } from "./client";

vi.mock("$lib/features/instances/context", () => ({
  getActiveInstanceId: () => null,
}));

vi.mock("$lib/music/device-id", () => ({
  getOrCreateDeviceId: () => "device-test",
}));

vi.mock("$lib/compat", () => ({
  clientCompatHeaders: () => ({}),
}));

vi.mock("$lib/config/remote-server", () => ({
  isRemoteClient: () => false,
  resolveApiUrl: (url: string) => url,
}));

const loggerInfo = vi.fn();
const loggerWarn = vi.fn();

vi.mock("$lib/core/logger", () => ({
  newRequestId: () => "req-test",
  logger: {
    debug: vi.fn(),
    info: (...args: unknown[]) => loggerInfo(...args),
    warn: (...args: unknown[]) => loggerWarn(...args),
    error: vi.fn(),
  },
}));

describe("isTransientNetworkError", () => {
  it("treats Failed to fetch as transient", () => {
    expect(
      isTransientNetworkError(new TypeError("Failed to fetch")),
    ).toBe(true);
  });

  it("treats AbortError as transient", () => {
    const err = new Error("aborted");
    err.name = "AbortError";
    expect(isTransientNetworkError(err)).toBe(true);
  });

  it("does not treat arbitrary errors as transient", () => {
    expect(isTransientNetworkError(new Error("boom"))).toBe(false);
  });
});

describe("fetchWithRetry", () => {
  beforeEach(() => {
    loggerInfo.mockClear();
    loggerWarn.mockClear();
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new TypeError("Failed to fetch")),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("logs network exhaustion at info, not warn", async () => {
    await expect(
      fetchWithRetry("/api/music/library-stats", undefined, 2),
    ).rejects.toThrow(/Failed to fetch/);

    expect(loggerInfo).toHaveBeenCalledWith(
      "HTTP request failed",
      expect.objectContaining({
        url: "/api/music/library-stats",
        attempts: 2,
      }),
      "http.client",
      "req-test",
    );
    expect(loggerWarn).not.toHaveBeenCalled();
  });
});
