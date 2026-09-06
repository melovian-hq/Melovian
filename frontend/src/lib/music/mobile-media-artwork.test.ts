// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { absoluteMobileArtworkUrl } from "./mobile-media-artwork";

describe("absoluteMobileArtworkUrl", () => {
  it("prefixes relative API paths with the WebView origin", () => {
    expect(
      absoluteMobileArtworkUrl(
        "/api/subsonic/rest/getCoverArt.view?id=1",
        "https://wails.localhost",
      ),
    ).toBe("https://wails.localhost/api/subsonic/rest/getCoverArt.view?id=1");
  });

  it("keeps absolute http URLs", () => {
    expect(
      absoluteMobileArtworkUrl(
        "https://music.example/cover.jpg",
        "https://wails.localhost",
      ),
    ).toBe("https://music.example/cover.jpg");
  });

  it("drops SVG data URLs that native cannot decode", () => {
    expect(
      absoluteMobileArtworkUrl(
        "data:image/svg+xml,%3Csvg%3E",
        "https://wails.localhost",
      ),
    ).toBe("");
  });

  it("returns empty for blank input", () => {
    expect(absoluteMobileArtworkUrl("  ", "https://wails.localhost")).toBe("");
  });
});
