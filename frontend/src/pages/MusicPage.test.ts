// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from "vitest";
import { flushSync, mount, unmount } from "svelte";

vi.hoisted(() => {
  const storage = new Map<string, string>();
  globalThis.localStorage = {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => {
      storage.set(key, value);
    },
    removeItem: (key: string) => {
      storage.delete(key);
    },
    clear: () => storage.clear(),
    key: () => null,
    length: 0,
  } as Storage;
  globalThis.matchMedia = ((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  })) as typeof window.matchMedia;
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as typeof ResizeObserver;
});

vi.mock("$lib/components/layout/AppShell.svelte", async () => ({
  default: (await import("../test-fixtures/Passthrough.svelte")).default,
}));

const { music } = vi.hoisted(() => {
  const music = {
    connected: true,
    loading: false,
    libraryReady: true,
    error: null,
    serverName: "Navidrome",
    config: {
      serverUrl: "http://localhost",
      version: "1.16.1",
      clientName: "melovian",
    },
    continuousBusy: "off" as const,
    continuousMode: "off" as const,
    mixesRegenerating: false,
    resumeTracks: [] as Array<{
      trackId: string;
      trackTitle: string;
      artistName: string;
      albumId: string;
      albumTitle: string;
      positionMs: number;
      durationMs: number;
      played: boolean;
      playCount: number;
      listenedMs: number;
      lastPlayedAt: string;
      coverArtId: string;
    }>,
    listenHistory: [] as Array<{
      trackId: string;
      trackTitle: string;
      artistName: string;
      albumId: string;
      albumTitle: string;
      positionMs: number;
      durationMs: number;
      played: boolean;
      playCount: number;
      listenedMs: number;
      lastPlayedAt: string;
      coverArtId: string;
    }>,
    recentAlbums: [] as Array<{
      id: string;
      name: string;
      artist?: string;
      coverArt?: string;
    }>,
    frequentAlbums: [] as Array<{
      id: string;
      name: string;
      artist?: string;
      coverArt?: string;
    }>,
    recommendations: [] as Array<{
      id: string;
      name: string;
      artist?: string;
      coverArt?: string;
    }>,
    favoriteAlbums: [] as Array<{
      id: string;
      name: string;
      artist?: string;
      coverArt?: string;
    }>,
    favoriteArtists: [] as Array<{
      id: string;
      name: string;
      coverArt?: string;
    }>,
    favoriteTracks: [] as Array<{ trackId: string }>,
    personalMixes: [] as Array<{
      id: string;
      title: string;
      subtitle: string;
      tracks: [];
      trackIds: string[];
      coverArtId?: string;
      gradient: string;
    }>,
    playlists: [] as Array<{
      id: string;
      name: string;
      createdAt: string;
      updatedAt: string;
      trackCount: number;
      coverArtIds?: string[];
    }>,
    serverPlaylists: [] as Array<{
      id: string;
      name: string;
      songCount?: number;
      coverArt?: string;
    }>,
    playLibraryShuffle: vi.fn(),
    regenerateMixes: vi.fn().mockResolvedValue(undefined),
    playMix: vi.fn(),
    shuffle: false,
    hideUnknownMetadata: false,
    metadataEnhancementSettings: {
      enabled: false,
      artists: false,
      albums: false,
      tracks: false,
      preferServerArtistArt: true,
    },
    library: {
      getAlbum: vi.fn().mockResolvedValue(null),
    },
  };
  return { music };
});

vi.mock("$lib/config/music.svelte", () => ({ music }));

vi.mock("$lib/features/sources/store.svelte", () => ({
  sources: { needsSetup: false, hasAnySource: true },
}));

vi.mock("$lib/features/local-libraries/store.svelte", () => ({
  localLibraries: { active: null },
}));

vi.mock("$lib/router/router.svelte", () => ({
  router: { pathname: "/music", navigate: vi.fn() },
  link: vi.fn(),
  withBase: (path: string) => path,
}));

import MusicPage from "./MusicPage.svelte";

function renderPage() {
  const target = document.createElement("div");
  document.body.appendChild(target);
  const instance = mount(MusicPage, { target });
  flushSync();
  return {
    html: target.innerHTML,
    text: target.textContent ?? "",
    cleanup: () => {
      void unmount(instance);
      target.remove();
    },
  };
}

describe("MusicPage home feed", () => {
  beforeEach(() => {
    music.loading = false;
    music.libraryReady = true;
    music.error = null;
    music.resumeTracks = [];
    music.listenHistory = [];
    music.recentAlbums = [];
    music.frequentAlbums = [];
    music.recommendations = [];
    music.favoriteAlbums = [];
    music.favoriteArtists = [];
    music.favoriteTracks = [];
    music.personalMixes = [];
    music.playlists = [];
    music.serverPlaylists = [];
  });

  it("shows a time-based greeting and empty state for a blank library", () => {
    const { text, cleanup } = renderPage();
    expect(text).toMatch(/Good (morning|afternoon|evening)/);
    expect(text).toContain("Nothing to play yet");
    expect(text).not.toContain("Personal radio");
    expect(text).not.toContain("Random radio");
    cleanup();
  });

  it("renders Spotify-style shelves from library data", () => {
    music.favoriteTracks = [{ trackId: "t1" }];
    music.personalMixes = [
      {
        id: "discover",
        title: "Discover Weekly",
        subtitle: "New finds",
        tracks: [],
        trackIds: ["a"],
        coverArtId: "mix-1",
        gradient: "linear-gradient(#000, #111)",
      },
    ];
    music.resumeTracks = [
      {
        trackId: "t1",
        trackTitle: "Alison",
        artistName: "Slowdive",
        albumId: "souvlaki",
        albumTitle: "Souvlaki",
        positionMs: 0,
        durationMs: 200000,
        played: true,
        playCount: 3,
        listenedMs: 180000,
        lastPlayedAt: "2026-08-15T12:00:00Z",
        coverArtId: "c1",
      },
    ];
    music.recentAlbums = [
      { id: "souvlaki", name: "Souvlaki", artist: "Slowdive" },
      { id: "just", name: "Just for a Day", artist: "Slowdive" },
      { id: "newest", name: "Newest Record", artist: "Other" },
    ];
    music.favoriteArtists = [{ id: "art-1", name: "Slowdive" }];
    music.playlists = [
      {
        id: "pl1",
        name: "Night drives",
        createdAt: "2026-01-01T00:00:00Z",
        updatedAt: "2026-01-01T00:00:00Z",
        trackCount: 8,
      },
    ];
    music.recommendations = [
      { id: "rec-1", name: "Loveless", artist: "My Bloody Valentine" },
    ];

    const { text, html, cleanup } = renderPage();
    expect(text).toContain("Jump back in");
    expect(text).toContain("Made for you");
    expect(html).toContain("home-shelf__items");
    expect(html).not.toContain("home-shelf__items--two-rows");
    expect(html).not.toContain("home-shelf__scroller");
    expect(text).toContain("Discover Weekly");
    expect(text).toContain("Because you listened to Slowdive");
    expect(text).toContain("Your playlists");
    expect(text).toContain("Night drives");
    expect(text).toContain("Your favorite artists");
    expect(text).toContain("New in your library");
    expect(text).toContain("Recommended for you");
    expect(text).toContain("Liked tracks");
    expect(text).not.toContain("Shuffle library");
    expect(html).not.toContain("/music/radios");
    expect(html).not.toContain("/music/stats");
    cleanup();
  });
});
