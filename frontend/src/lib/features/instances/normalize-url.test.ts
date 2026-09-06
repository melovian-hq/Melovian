// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { displayNameFromServerUrl, normalizeServerUrl } from "./normalize-url";

describe("normalizeServerUrl", () => {
  it("adds https when the scheme is missing", () => {
    expect(normalizeServerUrl("music.example.com")).toBe(
      "https://music.example.com",
    );
  });

  it("keeps an existing scheme and strips trailing slashes", () => {
    expect(normalizeServerUrl("http://localhost:4533/")).toBe(
      "http://localhost:4533",
    );
  });

  it("returns empty for blank input", () => {
    expect(normalizeServerUrl("  ")).toBe("");
  });
});

describe("displayNameFromServerUrl", () => {
  it("uses the hostname without www", () => {
    expect(displayNameFromServerUrl("https://www.navidrome.home/")).toBe(
      "navidrome.home",
    );
  });

  it("returns empty for blank input", () => {
    expect(displayNameFromServerUrl("")).toBe("");
  });
});
