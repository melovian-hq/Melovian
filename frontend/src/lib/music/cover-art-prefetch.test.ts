// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import {
  isCoverArtPrefetched,
  prefetchCoverArt,
  trackCoverArtUrl,
} from "./cover-art-prefetch";
import type { SubsonicConfig, SubsonicSong } from "$lib/subsonic";

describe("cover-art-prefetch", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "Image",
      class {
        decoding = "async";
        onload: (() => void) | null = null;
        onerror: (() => void) | null = null;
        private _src = "";
        set src(value: string) {
          this._src = value;
          queueMicrotask(() => this.onload?.());
        }
        get src() {
          return this._src;
        }
      },
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("builds track cover urls with track id for unified routing", () => {
    const config = {
      serverUrl: "http://example.com",
      version: "1.16.1",
      clientName: "melovian",
    } satisfies SubsonicConfig;
    const track = {
      id: "trk_1",
      title: "Song",
      albumId: "alb_1",
      coverArt: "alb_1",
    } satisfies SubsonicSong;

    expect(trackCoverArtUrl(config, track, 256)).toContain("alb_1");
    expect(trackCoverArtUrl(config, track, 256)).toContain("size=256");
  });

  it("deduplicates prefetch requests", async () => {
    prefetchCoverArt("http://example.com/cover.jpg");
    prefetchCoverArt("http://example.com/cover.jpg");
    await Promise.resolve();
    expect(isCoverArtPrefetched("http://example.com/cover.jpg")).toBe(true);
  });
});
