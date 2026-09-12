// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from "vitest";
import { getListenProgressBatch } from "./api";

describe("getListenProgressBatch", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("returns an empty map for no ids", async () => {
    await expect(getListenProgressBatch([])).resolves.toEqual(new Map());
    await expect(getListenProgressBatch(["", "  "])).resolves.toEqual(
      new Map(),
    );
  });

  it("deduplicates ids and parses batch payload", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers(),
      json: async () => ({
        t1: {
          trackId: "t1",
          trackTitle: "One",
          artistName: "Artist",
          albumId: "a1",
          albumTitle: "Album",
          positionMs: 0,
          durationMs: 1000,
          played: false,
          playCount: 0,
          listenedMs: 0,
          lastPlayedAt: "2026-01-01T00:00:00Z",
          coverArtId: "a1",
        },
        t2: {
          trackId: "t2",
          trackTitle: "Two",
          artistName: "Artist",
          albumId: "a1",
          albumTitle: "Album",
          positionMs: 0,
          durationMs: 1000,
          played: false,
          playCount: 0,
          listenedMs: 0,
          lastPlayedAt: "2026-01-01T00:00:00Z",
          coverArtId: "a1",
        },
      }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const result = await getListenProgressBatch(["t1", "t1", "t2"]);

    expect(fetchMock).toHaveBeenCalledOnce();
    expect(fetchMock.mock.calls[0]?.[0]).toContain("ids=t1%2Ct2");
    expect(result.get("t1")?.trackTitle).toBe("One");
    expect(result.get("t2")?.trackTitle).toBe("Two");
  });
});
