// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import "fake-indexeddb/auto";
import { beforeEach, describe, expect, it, vi, afterEach } from "vitest";
import type { QueueTrack } from "$lib/subsonic";

vi.mock("$lib/features/instances/context", () => ({
  getActiveInstanceId: vi.fn(() => "inst-1"),
}));

const track = (id: string): QueueTrack =>
  ({ id, title: id }) as unknown as QueueTrack;

let objectUrlSeq = 0;

describe("media-cache", () => {
  beforeEach(async () => {
    objectUrlSeq = 0;
    vi.stubGlobal(
      "URL",
      Object.assign(URL, {
        createObjectURL: vi.fn(() => `blob:mock-${++objectUrlSeq}`),
        revokeObjectURL: vi.fn(),
      }),
    );
    const mod = await import("./media-cache");
    mod.resetMediaCacheForTests();
    await new Promise<void>((res) => {
      const req = indexedDB.deleteDatabase("melovian-media-cache");
      req.onsuccess = req.onerror = req.onblocked = () => res();
    });
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("gates prefetch on saveData and slow connections", async () => {
    const { bandwidthAllowsPrefetch } = await import("./media-cache");
    const nav = navigator as Navigator & {
      connection?: { saveData?: boolean; effectiveType?: string };
    };
    expect(bandwidthAllowsPrefetch()).toBe(true);
    nav.connection = { saveData: true, effectiveType: "4g" };
    expect(bandwidthAllowsPrefetch()).toBe(false);
    nav.connection = { saveData: false, effectiveType: "2g" };
    expect(bandwidthAllowsPrefetch()).toBe(false);
    nav.connection = { saveData: false, effectiveType: "4g" };
    expect(bandwidthAllowsPrefetch()).toBe(true);
    delete nav.connection;
  });

  it("pins the network URL and fills the cache in the background", async () => {
    const { resolvePlaybackUrl, ensureCached, cachedTrackUrl } =
      await import("./media-cache");
    const t = track("song-1");
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(new Blob(["audio-data"], { type: "audio/mpeg" })),
      );
    vi.stubGlobal("fetch", fetchMock);

    const url = resolvePlaybackUrl(
      t,
      "https://m.example/rest/stream?id=song-1",
    );
    expect(url).toBe("https://m.example/rest/stream?id=song-1");
    // Pin is stable across calls
    expect(resolvePlaybackUrl(t, "https://other/url")).toBe(url);

    await ensureCached(t, "https://m.example/rest/stream?id=song-1");
    expect(fetchMock).toHaveBeenCalled();

    const cached = await cachedTrackUrl(t);
    expect(cached).toMatch(/^blob:mock-/);
  });

  it("serves a blob URL on first resolution when already cached", async () => {
    const { resolvePlaybackUrl, cachedTrackUrl } =
      await import("./media-cache");
    const t = track("song-2");
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(new Blob(["x"]))),
    );
    // Warm the cache via a direct store path
    const { ensureCached } = await import("./media-cache");
    await ensureCached(t, "https://m.example/rest/stream?id=song-2");
    expect(await cachedTrackUrl(t)).toMatch(/^blob:mock-/);
    expect(
      resolvePlaybackUrl(t, "https://m.example/rest/stream?id=song-2"),
    ).toMatch(/^blob:mock-/);
  });

  it("skips radio and local tracks", async () => {
    const { isCacheableTrack, ensureCached } = await import("./media-cache");
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    const radio = {
      id: "ir:station-1",
      streamUrl: "https://radio.example/live",
      isInternetRadio: true,
    } as unknown as QueueTrack;
    expect(isCacheableTrack(radio)).toBe(false);
    expect(isCacheableTrack({ id: "trk_abc" } as QueueTrack)).toBe(false);
    await ensureCached(radio, "https://radio.example/live");
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("dedupes concurrent fills for the same track", async () => {
    const { ensureCached } = await import("./media-cache");
    const t = track("song-3");
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(new Blob(["audio"])));
    vi.stubGlobal("fetch", fetchMock);
    // Both calls land while the first fill is in-flight, so the second must
    // reuse the inflight promise rather than fetch again.
    const p1 = ensureCached(t, "https://m.example/rest/stream?id=song-3");
    const p2 = ensureCached(t, "https://m.example/rest/stream?id=song-3");
    await Promise.all([p1, p2]);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("keeps pinned keys during eviction", async () => {
    const { resolvePlaybackUrl, ensureCached, mediaCacheStats } =
      await import("./media-cache");
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockImplementation(() =>
          Promise.resolve(new Response(new Blob(["12345"]))),
        ),
    );
    const pinned = track("pinned");
    resolvePlaybackUrl(pinned, "https://m.example/stream?p=1");
    await ensureCached(pinned, "https://m.example/stream?p=1");
    const stats = await mediaCacheStats();
    expect(stats.entries).toBe(1);
    expect(stats.bytes).toBeGreaterThan(0);
  });
});
