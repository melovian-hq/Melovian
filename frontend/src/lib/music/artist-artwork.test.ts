// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";
import {
  hasServerArtistArt,
  normalizeExternalMediaUrl,
  resolveServerArtistArtUrl,
} from "./artist-media";

vi.mock("$lib/features/instances/context", () => ({
  getActiveInstanceId: () => "inst-1",
}));

describe("artist-artwork", () => {
  const config = {
    serverUrl: "/api/subsonic",
    clientName: "melovian",
    version: "1.16.1",
  };

  it("proxies relative Navidrome paths through the Subsonic proxy", () => {
    const url = resolveServerArtistArtUrl(
      config,
      {
        coverArt: "ar-1",
        artistImageUrl: "/share/img/token?size=600",
      },
      240,
      (u) => `resolved:${u}`,
    );
    expect(url).toBe(
      "resolved:/api/subsonic/share/img/token?size=600&_instance=inst-1",
    );
  });

  it("proxies localhost artistImageUrl through melovian not the browser loopback", () => {
    const url = resolveServerArtistArtUrl(
      config,
      {
        coverArt: "ar-1",
        artistImageUrl:
          "http://localhost:4533/share/img/eyJhbGciOiJIUzI1NiJ9.token?size=600",
      },
      240,
      (u) => u,
    );
    expect(url).toBe(
      "/api/subsonic/share/img/eyJhbGciOiJIUzI1NiJ9.token?size=600&_instance=inst-1",
    );
    expect(url).not.toContain("localhost");
    expect(url).not.toContain("127.0.0.1");
  });

  it("proxies 127.0.0.1 share urls the same way", () => {
    expect(
      normalizeExternalMediaUrl(
        config,
        "http://127.0.0.1:4533/share/img/abc?size=300",
      ),
    ).toBe("/api/subsonic/share/img/abc?size=300&_instance=inst-1");
  });

  it("still proxies when config.serverUrl itself is loopback ND", () => {
    expect(
      normalizeExternalMediaUrl(
        { serverUrl: "http://localhost:4533" },
        "http://localhost:4533/share/img/tok?size=600",
      ),
    ).toBe("/api/subsonic/share/img/tok?size=600&_instance=inst-1");
  });

  it("falls back to Subsonic coverArt when image URL is missing", () => {
    const url = resolveServerArtistArtUrl(
      config,
      { coverArt: "ar-9" },
      160,
      (u) => u,
    );
    expect(url).toContain("getCoverArt");
    expect(url).toContain("ar-9");
  });

  it("detects server art from either field", () => {
    expect(hasServerArtistArt({ coverArt: "x" })).toBe(true);
    expect(hasServerArtistArt({ artistImageUrl: " http://a " })).toBe(true);
    expect(hasServerArtistArt({})).toBe(false);
  });

  it("leaves public absolute urls unchanged", () => {
    expect(
      normalizeExternalMediaUrl(config, "https://cdn.example/art.jpg"),
    ).toBe("https://cdn.example/art.jpg");
  });
});
