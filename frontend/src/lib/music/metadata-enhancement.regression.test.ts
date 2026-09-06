// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  artworkNeedsEnhancement,
  metadataNamesMatch,
  normalizeMetadataName,
  upscaleItunesArtwork,
} from "./metadata-enhancement";
import {
  defaultMetadataEnhancementSettings,
  mergeMetadataEnhancementSettings,
} from "./metadata-enhancement-settings";

describe("metadata-enhancement regression", () => {
  it("enables enhancement by default", () => {
    expect(mergeMetadataEnhancementSettings({ albums: true }).enabled).toBe(
      true,
    );
    expect(defaultMetadataEnhancementSettings().enabled).toBe(false);
  });

  it("treats data urls as needing enhancement", () => {
    expect(artworkNeedsEnhancement("data:image/svg+xml,abc")).toBe(true);
    expect(artworkNeedsEnhancement("https://cdn.example/cover.jpg")).toBe(
      false,
    );
  });

  it("keeps name matching stable for common cases", () => {
    expect(normalizeMetadataName("The Beatles")).toBe("beatles");
    expect(metadataNamesMatch("Pink Floyd", "pink floyd")).toBe(true);
    expect(
      upscaleItunesArtwork(
        "https://is1-ssl.mzstatic.com/image/thumb/Features/v4/test/100x100bb.jpg",
      ),
    ).toContain("600x600bb");
  });
});

describe("metadata-enhancement crash safety", () => {
  it("tolerates empty and odd inputs", () => {
    expect(normalizeMetadataName("")).toBe("");
    expect(metadataNamesMatch("", "")).toBe(false);
    expect(artworkNeedsEnhancement(null)).toBe(true);
    expect(artworkNeedsEnhancement(undefined, true)).toBe(true);
    expect(upscaleItunesArtwork("not-a-url")).toBe("not-a-url");
  });
});
