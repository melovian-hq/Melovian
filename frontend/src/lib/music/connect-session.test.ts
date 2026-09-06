// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  shouldAwaitConnectInFlight,
  shouldSkipMusicConnect,
} from "$lib/music/connect-session";

describe("connect-session", () => {
  it("skips connect when already connected without force", () => {
    expect(shouldSkipMusicConnect(true, false)).toBe(true);
  });

  it("does not skip when force reconnect is requested", () => {
    expect(shouldSkipMusicConnect(true, true)).toBe(false);
  });

  it("does not skip when disconnected", () => {
    expect(shouldSkipMusicConnect(false, false)).toBe(false);
  });

  it("awaits in-flight connect unless forced", () => {
    const inFlight = Promise.resolve(true);
    expect(shouldAwaitConnectInFlight(inFlight, false)).toBe(true);
    expect(shouldAwaitConnectInFlight(inFlight, true)).toBe(false);
    expect(shouldAwaitConnectInFlight(null, false)).toBe(false);
  });
});
