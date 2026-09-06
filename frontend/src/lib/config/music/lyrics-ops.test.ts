// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ParsedLyrics } from "$lib/music/lyrics";
import type { QueueTrack } from "$lib/subsonic";
import type { MusicLyricsContext } from "./lyrics-ops";
import { fetchCurrentLyrics, loadCurrentLyrics } from "./lyrics-ops";

vi.mock("$lib/music/api", () => ({
  getTrackLyrics: vi.fn(),
  fetchTrackLyrics: vi.fn(),
  getLyricsSettings: vi.fn(),
  saveLyricsSettingsRemote: vi.fn(),
}));

vi.mock("$lib/ui/toast.svelte", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

vi.mock("$lib/core/logger", () => ({
  logger: {
    error: vi.fn(),
    warn: vi.fn(),
    info: vi.fn(),
    debug: vi.fn(),
  },
}));

import * as musicApi from "$lib/music/api";
import { toast } from "$lib/ui/toast.svelte";

const track: QueueTrack = {
  id: "trk_1",
  title: "Song",
  artist: "Artist",
  album: "Album",
  duration: 200,
};

function makeCtx(
  overrides: Partial<MusicLyricsContext> = {},
): MusicLyricsContext {
  return {
    lyricsOpen: true,
    currentTrack: track,
    currentLyrics: null,
    lyricsLoading: false,
    lyricsFetching: false,
    lyricsRequestId: 0,
    pendingLyricsReload: false,
    lyricsSettings: null,
    favoriteTracks: [],
    listenHistory: [],
    resumeTracks: [],
    randomTracks: [],
    library: {
      getLyricsForSong: vi.fn().mockResolvedValue(null),
    } as unknown as MusicLyricsContext["library"],
    config: {} as MusicLyricsContext["config"],
    seek: vi.fn(),
    favoriteToSong: vi.fn(),
    entryToSong: vi.fn(),
    ...overrides,
  };
}

describe("lyrics-ops regression", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("toasts synced lyrics distinctly after a successful refetch", async () => {
    const synced: ParsedLyrics = {
      synced: true,
      offsetMs: 0,
      rawValue: "[00:01.00]Hello",
      lines: [{ startMs: 1000, text: "Hello" }],
    };
    vi.mocked(musicApi.fetchTrackLyrics).mockResolvedValue(synced);
    const ctx = makeCtx();

    const result = await fetchCurrentLyrics(ctx);

    expect(result?.synced).toBe(true);
    expect(ctx.currentLyrics?.synced).toBe(true);
    expect(toast.success).toHaveBeenCalledWith("Synced lyrics fetched");
  });

  it("toasts plain lyrics when refetch returns unsynced text", async () => {
    const plain: ParsedLyrics = {
      synced: false,
      offsetMs: 0,
      rawValue: "Hello",
      lines: [{ text: "Hello" }],
    };
    vi.mocked(musicApi.fetchTrackLyrics).mockResolvedValue(plain);
    const ctx = makeCtx();

    await fetchCurrentLyrics(ctx);

    expect(toast.success).toHaveBeenCalledWith("Lyrics fetched");
  });

  it("reparses embedded LRC from cached remote payloads on load", async () => {
    vi.mocked(musicApi.getTrackLyrics).mockResolvedValue({
      synced: false,
      offsetMs: 0,
      rawValue: "[00:01.00]First\n[00:03.00]Second",
      lines: [{ text: "First" }, { text: "Second" }],
    });
    const ctx = makeCtx();

    await loadCurrentLyrics(ctx);

    expect(ctx.currentLyrics?.synced).toBe(true);
    expect(ctx.currentLyrics?.lines[0]?.startMs).toBe(1000);
    expect(ctx.currentLyrics?.lines[0]?.text).toBe("First");
  });

  it("replaces current plain lyrics when refetch returns synced", async () => {
    const plain: ParsedLyrics = {
      synced: false,
      offsetMs: 0,
      rawValue: "Old plain",
      lines: [{ text: "Old plain" }],
    };
    const synced: ParsedLyrics = {
      synced: true,
      offsetMs: 0,
      rawValue: "[00:00.00]New",
      lines: [{ startMs: 0, text: "New" }],
    };
    vi.mocked(musicApi.fetchTrackLyrics).mockResolvedValue(synced);
    const ctx = makeCtx({ currentLyrics: plain });

    await fetchCurrentLyrics(ctx);

    expect(ctx.currentLyrics?.synced).toBe(true);
    expect(ctx.currentLyrics?.lines[0]?.text).toBe("New");
  });

  it("keeps fetched lyrics when a slower load finishes afterward", async () => {
    const stale: ParsedLyrics = {
      synced: false,
      offsetMs: 0,
      rawValue: "Stale",
      lines: [{ text: "Stale" }],
    };
    const fresh: ParsedLyrics = {
      synced: true,
      offsetMs: 0,
      rawValue: "[00:00.00]Fresh",
      lines: [{ startMs: 0, text: "Fresh" }],
    };

    let resolveLoad: (value: ParsedLyrics) => void = () => {};
    const loadPromise = new Promise<ParsedLyrics>((resolve) => {
      resolveLoad = resolve;
    });
    vi.mocked(musicApi.getTrackLyrics).mockReturnValue(loadPromise);
    vi.mocked(musicApi.fetchTrackLyrics).mockResolvedValue(fresh);

    const ctx = makeCtx({ currentLyrics: stale });
    const loading = loadCurrentLyrics(ctx);
    await fetchCurrentLyrics(ctx);

    expect(ctx.currentLyrics?.lines[0]?.text).toBe("Fresh");

    resolveLoad(stale);
    await loading;

    expect(ctx.currentLyrics?.lines[0]?.text).toBe("Fresh");
    expect(ctx.currentLyrics?.synced).toBe(true);
  });

  it("skips auto-load while a fetch is already in flight", async () => {
    let resolveFetch: (value: ParsedLyrics) => void = () => {};
    const fetchPromise = new Promise<ParsedLyrics>((resolve) => {
      resolveFetch = resolve;
    });
    vi.mocked(musicApi.fetchTrackLyrics).mockReturnValue(fetchPromise);
    vi.mocked(musicApi.getTrackLyrics).mockResolvedValue({
      synced: false,
      offsetMs: 0,
      rawValue: "Should not win",
      lines: [{ text: "Should not win" }],
    });

    const ctx = makeCtx();
    const fetching = fetchCurrentLyrics(ctx);
    await loadCurrentLyrics(ctx);

    expect(musicApi.getTrackLyrics).not.toHaveBeenCalled();
    expect(ctx.pendingLyricsReload).toBe(true);

    resolveFetch({
      synced: true,
      offsetMs: 0,
      rawValue: "[00:00.00]Fetched",
      lines: [{ startMs: 0, text: "Fetched" }],
    });
    await fetching;

    expect(musicApi.getTrackLyrics).not.toHaveBeenCalled();
    expect(ctx.currentLyrics?.lines[0]?.text).toBe("Fetched");
  });

  it("reloads lyrics after track change during an in-flight fetch", async () => {
    let resolveFetch: (value: ParsedLyrics) => void = () => {};
    const fetchPromise = new Promise<ParsedLyrics>((resolve) => {
      resolveFetch = resolve;
    });
    vi.mocked(musicApi.fetchTrackLyrics).mockReturnValue(fetchPromise);
    vi.mocked(musicApi.getTrackLyrics).mockResolvedValue({
      synced: true,
      offsetMs: 0,
      rawValue: "[00:00.00]Next",
      lines: [{ startMs: 0, text: "Next" }],
    });

    const ctx = makeCtx();
    const fetching = fetchCurrentLyrics(ctx);
    ctx.currentTrack = { ...track, id: "trk_2", title: "Other" };
    await loadCurrentLyrics(ctx);
    expect(ctx.pendingLyricsReload).toBe(true);

    resolveFetch({
      synced: false,
      offsetMs: 0,
      rawValue: "Old track",
      lines: [{ text: "Old track" }],
    });
    await fetching;
    await Promise.resolve();

    expect(musicApi.getTrackLyrics).toHaveBeenCalled();
    expect(ctx.currentLyrics?.lines[0]?.text).toBe("Next");
  });
});
