// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  clearBrokenArtwork,
  isArtworkBroken,
  markArtworkBroken,
} from "./artwork-status";

describe("artwork-status", () => {
  beforeEach(() => {
    localStorage.clear();
    sessionStorage.clear();
    clearBrokenArtwork();
  });

  it("reports marked URLs and ignores unmarked ones", () => {
    markArtworkBroken("https://server/rest/getCoverArt?id=a");
    expect(isArtworkBroken("https://server/rest/getCoverArt?id=a")).toBe(true);
    expect(isArtworkBroken("https://server/rest/getCoverArt?id=b")).toBe(false);
  });

  it("does not track data or blob urls", () => {
    markArtworkBroken("data:image/svg+xml;base64,abc");
    markArtworkBroken("blob:https://app/x");
    expect(isArtworkBroken("data:image/svg+xml;base64,abc")).toBe(false);
    expect(isArtworkBroken("blob:https://app/x")).toBe(false);
  });

  it("handles empty input without throwing", () => {
    markArtworkBroken(null);
    markArtworkBroken(undefined);
    expect(isArtworkBroken(null)).toBe(false);
    expect(isArtworkBroken(undefined)).toBe(false);
    expect(isArtworkBroken("")).toBe(false);
  });

  it("clears all entries", () => {
    markArtworkBroken("https://server/rest/getCoverArt?id=a");
    clearBrokenArtwork();
    expect(isArtworkBroken("https://server/rest/getCoverArt?id=a")).toBe(false);
  });

  it("survives a reload so missing art is not re-probed", async () => {
    markArtworkBroken("https://server/rest/getCoverArt?id=a");
    vi.resetModules();
    const fresh = await import("./artwork-status");
    expect(fresh.isArtworkBroken("https://server/rest/getCoverArt?id=a")).toBe(
      true,
    );
  });

  it("expires the blob so art added upstream gets retried", async () => {
    vi.useFakeTimers();
    try {
      markArtworkBroken("https://server/rest/getCoverArt?id=old");
      vi.setSystemTime(Date.now() + 25 * 60 * 60 * 1000);
      vi.resetModules();
      const fresh = await import("./artwork-status");
      expect(
        fresh.isArtworkBroken("https://server/rest/getCoverArt?id=old"),
      ).toBe(false);
    } finally {
      vi.useRealTimers();
    }
  });
});
