// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { artistIdFromName } from "./local-ids";

describe("artistIdFromName", () => {
  it("matches backend localmusic.ArtistID hashes", () => {
    expect(artistIdFromName("OG Buda")).toBe("art_6e8c05a9f72985e1");
    expect(artistIdFromName("Artist A")).toBe("art_3a8b2ccfc1e02291");
    expect(artistIdFromName("artist a")).toBe("art_3a8b2ccfc1e02291");
    expect(artistIdFromName("Drake")).toBe("art_d22885245140da09");
    expect(artistIdFromName("Travis Scott")).toBe("art_494924982b2e9f61");
    expect(artistIdFromName("Artist2")).toBe("art_d4d4b6292e0d2ae8");
    expect(artistIdFromName("OG Buda & Artist2")).toBe("art_585e934f873e4671");
  });

  it("returns empty string for blank names", () => {
    expect(artistIdFromName("   ")).toBe("");
  });
});
