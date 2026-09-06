// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { SubsonicSearchResult } from "$lib/subsonic/types";
import {
  createLocalLibraryAdapter,
  createUnifiedLibraryAdapter,
  type MusicLibraryAdapter,
} from "$lib/music/library-adapter";

describe("local library adapter", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url.endsWith("/api/local-music/search?q=alpha&limit=5")) {
          return new Response(
            JSON.stringify({
              artists: [],
              albums: [],
              songs: [{ id: "trk_1", title: "Alpha Song", artist: "Artist" }],
            }),
            { status: 200, headers: { "Content-Type": "application/json" } },
          );
        }
        return new Response("not found", { status: 404 });
      }),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("maps local search responses into subsonic search results", async () => {
    const library = createLocalLibraryAdapter();
    const result = await library.search3("alpha", 5);
    expect(result.songs).toEqual([
      expect.objectContaining({ id: "trk_1", title: "Alpha Song" }),
    ]);
  });

  it("queries subsonic and local search in parallel", async () => {
    vi.useFakeTimers();
    const order: string[] = [];
    const empty: SubsonicSearchResult = { artists: [], albums: [], songs: [] };
    const subsonic = {
      search3: vi.fn(
        () =>
          new Promise<SubsonicSearchResult>((resolve) => {
            setTimeout(() => {
              order.push("subsonic");
              resolve(empty);
            }, 50);
          }),
      ),
    } as unknown as MusicLibraryAdapter;
    const local = {
      search3: vi.fn(
        () =>
          new Promise<SubsonicSearchResult>((resolve) => {
            setTimeout(() => {
              order.push("local");
              resolve({
                artists: [],
                albums: [],
                songs: [{ id: "trk_1", title: "Alpha Song", artist: "Artist" }],
              });
            }, 10);
          }),
      ),
    } as unknown as MusicLibraryAdapter;

    const library = createUnifiedLibraryAdapter(subsonic, local);
    const resultPromise = library.search3("alpha", 5);
    await vi.advanceTimersByTimeAsync(50);
    const result = await resultPromise;

    expect(order).toEqual(["local", "subsonic"]);
    expect(result.songs).toEqual([
      expect.objectContaining({ id: "trk_1", title: "Alpha Song" }),
    ]);
    vi.useRealTimers();
  });
});
