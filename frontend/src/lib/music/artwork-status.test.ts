// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it } from "vitest";
import {
  clearBrokenArtwork,
  isArtworkBroken,
  markArtworkBroken,
} from "./artwork-status";

describe("artwork-status", () => {
  beforeEach(() => {
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
});
