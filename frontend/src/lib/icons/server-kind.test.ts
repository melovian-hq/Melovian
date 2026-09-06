// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { detectServerKind, serverIconPath } from "./server-kind";

describe("detectServerKind", () => {
  it("detects Navidrome from server name", () => {
    expect(detectServerKind("Navidrome", "0.54.5")).toBe("navidrome");
  });

  it("detects Navidrome from version string", () => {
    expect(detectServerKind("My Music", "navidrome-0.54.5")).toBe("navidrome");
  });

  it("detects OpenSubsonic type field", () => {
    expect(detectServerKind("navidrome", "0.58.0")).toBe("navidrome");
  });

  it("defaults to Subsonic for other servers", () => {
    expect(detectServerKind("Subsonic", "6.1.6")).toBe("subsonic");
    expect(detectServerKind("Airsonic", "10.8.0")).toBe("subsonic");
  });
});

describe("serverIconPath", () => {
  it("uses the blue Navidrome icon", () => {
    expect(serverIconPath("navidrome")).toBe("/icons/navidrome.svg");
  });

  it("uses a single Subsonic icon", () => {
    expect(serverIconPath("subsonic")).toBe("/icons/subsonic.svg");
  });
});
