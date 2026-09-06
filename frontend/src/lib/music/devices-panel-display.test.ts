// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  deviceKindFromUserAgent,
  peerStatusLabel,
  resolveAmbientCoverArt,
} from "./devices-panel-display";

describe("devices-panel-display", () => {
  it("prefers active peer cover for ambient wash", () => {
    expect(
      resolveAmbientCoverArt({
        localCoverArt: "local",
        activePeerCoverArt: "peer",
      }),
    ).toBe("peer");
    expect(
      resolveAmbientCoverArt({
        localCoverArt: "local",
        activePeerCoverArt: "  ",
      }),
    ).toBe("local");
    expect(resolveAmbientCoverArt({})).toBe("");
  });

  it("builds peer status labels", () => {
    expect(
      peerStatusLabel({
        isActivePlayer: true,
        playback: { trackTitle: "Song", positionMs: 0, paused: false },
        sessionId: "s1",
      }),
    ).toBe("Playing · Song · Listening together");
    expect(
      peerStatusLabel({
        isActivePlayer: false,
        playback: null,
        sessionId: "s1",
        isHost: true,
      }),
    ).toBe("Idle · Hosting");
    expect(
      peerStatusLabel({
        isActivePlayer: false,
        playback: null,
      }),
    ).toBe("Idle");
  });

  it("maps user agents to device glyphs", () => {
    expect(deviceKindFromUserAgent("Mozilla/5.0 (iPhone)")).toBe("headphones");
    expect(deviceKindFromUserAgent("Mozilla/5.0 (Windows NT 10.0)")).toBe(
      "monitor",
    );
    expect(deviceKindFromUserAgent("Unknown")).toBe("music");
  });
});
