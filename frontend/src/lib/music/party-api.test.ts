// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { partyStreamUrl, partyCoverUrl } from "./party-api";

describe("party media urls", () => {
  it("builds stream and cover paths", () => {
    expect(partyStreamUrl("lt-1", "trk_a")).toContain(
      "/api/party/lt-1/stream/trk_a",
    );
    expect(partyCoverUrl("lt-1", "cover-1")).toContain(
      "/api/party/lt-1/cover/cover-1",
    );
  });
});
