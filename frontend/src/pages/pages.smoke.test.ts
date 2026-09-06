// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from "vitest";

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
});

vi.mock("$lib/components/layout/AppShell.svelte", async () => ({
  default: (await import("../test-fixtures/Passthrough.svelte")).default,
}));

vi.mock("$lib/config/music.svelte", () => ({
  music: {
    connected: true,
    loading: false,
    error: null,
    config: {
      serverUrl: "http://localhost",
      version: "1.16.1",
      clientName: "melovian",
    },
    connect: vi.fn().mockResolvedValue(true),
    getGenreSongs: vi.fn().mockResolvedValue([]),
    playGenre: vi.fn(),
    playTracks: vi.fn(),
    playTracksNext: vi.fn(),
    addTracksToQueue: vi.fn(),
    shuffle: false,
    isFavorite: vi.fn().mockReturnValue(false),
    playRandomRadio: vi.fn(),
    playLibraryShuffle: vi.fn(),
    playRandomTracks: vi.fn(),
    libraryReady: true,
    continuousBusy: "off",
    continuousMode: "off",
    searchAll: vi.fn().mockResolvedValue({
      query: "",
      result: { artists: [], albums: [], songs: [] },
      suggestion: null,
      similar: [],
    }),
    loadArtists: vi.fn().mockResolvedValue([]),
    recentAlbums: [],
    frequentAlbums: [],
    recommendations: [],
    personalMixes: [],
    randomTracks: [],
    favoriteTracks: [],
    favoriteAlbums: [],
    favoriteArtists: [],
    resumeTracks: [],
    playlists: [],
    serverPlaylists: [],
    internetRadios: [],
    stats: { topArtists: [], topTracks: [], topAlbums: [] },
    libraryStats: null,
    libraryRefreshing: false,
    libraryRevision: 0,
    refreshLibrary: vi.fn().mockResolvedValue(undefined),
    hideUnknownMetadata: false,
    setHideUnknownMetadata: vi.fn(),
    metadataEnhancementSettings: {
      enabled: true,
      artists: true,
      albums: true,
      tracks: true,
      preferServerArtistArt: true,
    },
    updateMetadataEnhancementSettings: vi.fn(),
    genres: [],
    listenHistory: [],
    playAllFavorites: vi.fn(),
    playFavoriteAlbums: vi.fn(),
    playFavoriteArtists: vi.fn(),
    favoriteToSong: vi.fn((entry: { trackId: string }) => ({
      id: entry.trackId,
    })),
    refreshFavorites: vi.fn().mockResolvedValue(undefined),
    refreshMixes: vi.fn().mockResolvedValue(undefined),
    regenerateMixes: vi.fn().mockResolvedValue(undefined),
    getMix: vi.fn().mockReturnValue(undefined),
    refreshLibraryStats: vi.fn().mockResolvedValue(undefined),
    refreshHistory: vi.fn().mockResolvedValue(undefined),
    loadLyricsSettings: vi.fn().mockResolvedValue(undefined),
    loadCacheSettings: vi.fn().mockResolvedValue(undefined),
    mixesRegenerating: false,
    updateMixSettings: vi.fn(),
    mixSettings: { preferredLanguages: [], enabledMixes: [] },
    serverName: "Test Server",
    disconnect: vi.fn(),
    setRoutePath: vi.fn(),
    currentTrack: null,
    queue: [],
    queueIndex: 0,
    currentLyrics: null,
    lyricsLoading: false,
    lyricsFetching: false,
    playing: false,
    currentTime: 0,
    library: {
      getSimilarSongs: vi.fn().mockResolvedValue([]),
    },
    loadCurrentLyrics: vi.fn().mockResolvedValue(undefined),
    fetchCurrentLyrics: vi.fn().mockResolvedValue(null),
    clearQueue: vi.fn(),
    moveInQueue: vi.fn(),
    removeFromQueue: vi.fn(),
    playQueueIndex: vi.fn(),
    seekToLyricLine: vi.fn(),
  },
}));

vi.mock("$lib/features/auth/store.svelte", () => ({
  auth: {
    enabled: false,
    authenticated: false,
    demoMode: false,
    needsAccountLogin: false,
    loading: false,
    statusLoaded: true,
    error: null,
    init: vi.fn().mockResolvedValue(undefined),
  },
}));

vi.mock("$lib/features/instances/store.svelte", () => ({
  instances: {
    ready: true,
    needsLogin: false,
    items: [],
    active: null,
    init: vi.fn().mockResolvedValue(undefined),
  },
}));

vi.mock("$lib/features/local-libraries/store.svelte", () => ({
  localLibraries: {
    ready: true,
    enabled: false,
    items: [],
    scanningItems: [],
    init: vi.fn().mockResolvedValue(undefined),
    bindEvents: vi.fn(() => () => {}),
  },
}));

vi.mock("$lib/music/connection.svelte", () => ({
  connection: {
    loadRemoteSettings: vi.fn().mockResolvedValue(undefined),
    init: vi.fn(),
    dispose: vi.fn(),
    settings: {},
  },
}));

vi.mock("$lib/features/sources/store.svelte", () => ({
  sources: {
    ready: true,
    needsSetup: false,
    hasAnySource: true,
    hasSubsonicActive: true,
    hasLocalActive: false,
    hasUnifiedMode: false,
    refreshStatus: vi.fn().mockResolvedValue(undefined),
  },
}));

vi.mock("$lib/router/router.svelte", () => ({
  router: {
    pathname: "/music",
    search: "",
    navigate: vi.fn(),
    match: () => null,
  },
  link: vi.fn(),
  withBase: (path: string) => path,
}));

vi.mock("$lib/core/events/ws.svelte", () => ({
  eventSocket: {
    connect: vi.fn(),
    disconnect: vi.fn(),
    connected: false,
    on: vi.fn(() => () => {}),
  },
}));

vi.mock("$lib/subsonic/detail-cache", () => ({
  fetchAlbumWithCache: vi.fn().mockResolvedValue({
    album: {
      id: "alb_1",
      name: "Test Album",
      artist: "Artist",
      songCount: 1,
      duration: 200,
    },
    songs: [{ id: "1", title: "Track One", artist: "Artist" }],
  }),
  fetchArtistWithCache: vi.fn().mockResolvedValue({
    artist: { id: "art_1", name: "Test Artist" },
    albums: [],
  }),
  invalidateAlbumDetailCache: vi.fn(),
  invalidateArtistDetailCache: vi.fn(),
}));

import { flushSync, mount, unmount } from "svelte";
import GenrePage from "./GenrePage.svelte";
import SearchPage from "./SearchPage.svelte";
import SettingsPage from "./SettingsPage.svelte";
import MusicPage from "./MusicPage.svelte";
import AccountLoginPage from "./AccountLoginPage.svelte";
import AlbumPage from "./AlbumPage.svelte";
import NowPlayingPage from "./NowPlayingPage.svelte";
import FavoritesPage from "./FavoritesPage.svelte";
import HistoryPage from "./HistoryPage.svelte";
import MetadataEditorPage from "./MetadataEditorPage.svelte";
import App from "../App.svelte";

function renderPage(Component: unknown, props: Record<string, unknown> = {}) {
  const target = document.createElement("div");
  document.body.appendChild(target);
  const instance = mount(Component as never, { target, props });
  flushSync();
  return {
    html: target.innerHTML,
    cleanup: () => {
      void unmount(instance);
      target.remove();
    },
  };
}

describe("page smoke tests", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url.includes("/api/config")) {
          return new Response(JSON.stringify({ dataDir: "/tmp" }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          });
        }
        if (url.includes("/api/auth/status")) {
          return new Response(
            JSON.stringify({ enabled: false, authenticated: false }),
            { status: 200, headers: { "Content-Type": "application/json" } },
          );
        }
        if (url.includes("/api/instances")) {
          return new Response(JSON.stringify({ instances: [] }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          });
        }
        return new Response("{}", {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }),
    );
  });

  it("renders GenrePage with its decoded genre title", () => {
    const { html, cleanup } = renderPage(GenrePage, { genre: "Synthwave" });
    expect(html).toContain("Synthwave");
    cleanup();
  });

  it("renders SearchPage with its search controls", () => {
    const { html, cleanup } = renderPage(SearchPage, {});
    expect(html).toContain("Search");
    cleanup();
  });

  it("renders SettingsPage", () => {
    const { html, cleanup } = renderPage(SettingsPage, { tab: "general" });
    expect(html).toContain("Settings");
    cleanup();
  });

  it("renders MusicPage", () => {
    const { html, cleanup } = renderPage(MusicPage, {});
    expect(html).toMatch(/Good (morning|afternoon|evening)/);
    expect(html).toContain("Nothing to play yet");
    expect(html).toContain("Source settings");
    expect(html).not.toContain("Personal radio");
    expect(html).not.toContain("Stats");
    cleanup();
  });

  it("renders AccountLoginPage", () => {
    const { html, cleanup } = renderPage(AccountLoginPage, {});
    expect(html).toContain("Sign in");
    cleanup();
  });

  it("renders AlbumPage", () => {
    const { cleanup } = renderPage(AlbumPage, { id: "alb_1" });
    cleanup();
  });

  it("renders NowPlayingPage empty state", () => {
    const { html, cleanup } = renderPage(NowPlayingPage, {});
    expect(html).toContain("Nothing playing");
    cleanup();
  });

  it("renders FavoritesPage collection hero", () => {
    const { html, cleanup } = renderPage(FavoritesPage, {});
    expect(html).toContain("Favorites");
    expect(html).toContain("Songs");
    cleanup();
  });

  it("renders HistoryPage collection hero", () => {
    const { html, cleanup } = renderPage(HistoryPage, {});
    expect(html).toContain("History");
    expect(html).toContain("All time");
    cleanup();
  });

  it("renders MetadataEditorPage", () => {
    const { html, cleanup } = renderPage(MetadataEditorPage, {});
    expect(html).toContain("Metadata editor");
    expect(html).toContain("No active local library");
    cleanup();
  });

  it("renders App shell when auth is not required", async () => {
    const { html, cleanup } = renderPage(App, {});
    await vi.waitFor(() => expect(html.length).toBeGreaterThan(0));
    cleanup();
  });
});
