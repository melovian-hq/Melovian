// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  shouldAwaitConnectInFlight,
  shouldSkipMusicConnect,
} from "$lib/music/connect-session";

describe("music connect session (characterization before connect-ops)", () => {
  it("awaits in-flight connect unless force is set", () => {
    const inflight = Promise.resolve(true);
    expect(shouldAwaitConnectInFlight(inflight, false)).toBe(true);
    expect(shouldAwaitConnectInFlight(inflight, true)).toBe(false);
    expect(shouldAwaitConnectInFlight(null, false)).toBe(false);
  });

  it("skips connect when already connected unless forced", () => {
    expect(shouldSkipMusicConnect(true, false)).toBe(true);
    expect(shouldSkipMusicConnect(true, true)).toBe(false);
    expect(shouldSkipMusicConnect(false, false)).toBe(false);
  });
});
