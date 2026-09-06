// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  DEFAULT_CACHE_LIMIT_BYTES,
  defaultCacheSettings,
  formatBytes,
  mergeCacheSettings,
} from "./cache-settings";

describe("cache-settings", () => {
  it("defaults to enabled with a 2GB limit", () => {
    expect(defaultCacheSettings()).toEqual({
      enabled: true,
      limitBytes: DEFAULT_CACHE_LIMIT_BYTES,
      strategy: "playback",
    });
  });

  it("merges partial settings safely", () => {
    expect(mergeCacheSettings({ enabled: false })).toEqual({
      enabled: false,
      limitBytes: DEFAULT_CACHE_LIMIT_BYTES,
      strategy: "playback",
    });
  });

  it("formats byte sizes", () => {
    expect(formatBytes(0)).toBe("0 B");
    expect(formatBytes(1024)).toBe("1.0 KB");
    expect(formatBytes(DEFAULT_CACHE_LIMIT_BYTES)).toBe("2.0 GB");
  });
});
