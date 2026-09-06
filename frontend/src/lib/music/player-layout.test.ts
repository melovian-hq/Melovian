// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  isVideoPlaybackPath,
  isNowPlayingPath,
  playerLayoutForPath,
} from "./player-layout";

describe("isVideoPlaybackPath", () => {
  it("matches play routes", () => {
    expect(isVideoPlaybackPath("/play/abc123")).toBe(true);
    expect(isVideoPlaybackPath("/play/movie-id?start=1000")).toBe(true);
  });

  it("does not match other routes", () => {
    expect(isVideoPlaybackPath("/music")).toBe(false);
    expect(isVideoPlaybackPath("/item/abc123")).toBe(false);
    expect(isVideoPlaybackPath("/")).toBe(false);
  });
});

describe("isNowPlayingPath", () => {
  it("matches the now playing route", () => {
    expect(isNowPlayingPath("/music/now-playing")).toBe(true);
  });

  it("does not match other routes", () => {
    expect(isNowPlayingPath("/music")).toBe(false);
    expect(isNowPlayingPath("/music/lyrics")).toBe(false);
  });
});

describe("playerLayoutForPath", () => {
  it("uses full layout on app routes", () => {
    expect(playerLayoutForPath("/music", "mini")).toBe("full");
    expect(playerLayoutForPath("/music/album/abc", "mini")).toBe("full");
    expect(playerLayoutForPath("/settings/profile", "mini")).toBe("full");
    expect(playerLayoutForPath("/music/metadata", "mini")).toBe("full");
  });

  it("preserves dismissed layout", () => {
    expect(playerLayoutForPath("/music", "dismissed")).toBe("dismissed");
    expect(playerLayoutForPath("/settings/profile", "dismissed")).toBe(
      "dismissed",
    );
  });
});
