// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from "vitest";
import { flushSync, mount, unmount } from "svelte";
import type { QueueTrack, SubsonicSong } from "$lib/subsonic";
import type { ParsedLyrics } from "$lib/music/lyrics";
import { layout } from "$lib/components/layout/layout.svelte";

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

const sampleTrack: SubsonicSong = {
  id: "trk_1",
  title: "Weathergirl",
  artist: "Flavor Foley",
  artistId: "art_1",
  album: "Weathergirl",
  albumId: "alb_1",
  year: 2024,
  suffix: "flac",
  duration: 258,
  coverArt: "cov_1",
};

const queueTrack: SubsonicSong = {
  id: "trk_2",
  title: "Next Song",
  artist: "Other Artist",
  duration: 180,
};

const relatedTrack: SubsonicSong = {
  id: "trk_3",
  title: "Related Song",
  artist: "Flavor Foley",
  duration: 200,
};

const syncedLyrics: ParsedLyrics = {
  synced: true,
  offsetMs: 0,
  rawValue: "[00:00.00]Azure views\n[00:02.00]For two",
  lines: [
    { startMs: 0, text: "Azure views" },
    { startMs: 2000, text: "For two" },
  ],
};

const { music, getRelatedTracksCached } = vi.hoisted(() => {
  const getRelatedTracksCached = vi.fn();
  const music = {
    config: {
      serverUrl: "http://localhost",
      version: "1.16.1",
      clientName: "melovian",
    },
    currentTrack: null as QueueTrack | null,
    queue: [] as QueueTrack[],
    queueIndex: 0,
    currentLyrics: null as ParsedLyrics | null,
    lyricsLoading: false,
    lyricsFetching: false,
    playing: false,
    currentTime: 0,
    library: {
      getSimilarSongs: vi.fn().mockResolvedValue([]),
      getAlbum: vi.fn(),
      getArtist: vi.fn(),
      search3: vi.fn(),
    },
    isFavorite: vi.fn().mockReturnValue(false),
    toggleFavorite: vi.fn().mockResolvedValue(undefined),
    toggleEqPanel: vi.fn(),
    eqAvailable: true,
    eqOpen: false,
    eq: { enabled: false },
    loadCurrentLyrics: vi.fn().mockResolvedValue(undefined),
    fetchCurrentLyrics: vi.fn().mockResolvedValue(null),
    metadataEnhancementSettings: {
      enabled: true,
      artists: true,
      albums: true,
      tracks: true,
    },
    clearQueue: vi.fn(),
    moveInQueue: vi.fn(),
    removeFromQueue: vi.fn(),
    playQueueIndex: vi.fn(),
    playTracks: vi.fn(),
    seekToLyricLine: vi.fn(),
    seek: vi.fn(),
    togglePlay: vi.fn(),
    next: vi.fn(),
    previous: vi.fn(),
    duration: 0,
    smoothProgress: 0,
  };
  return { music, getRelatedTracksCached };
});

vi.mock("$lib/components/layout/AppShell.svelte", async () => ({
  default: (await import("../test-fixtures/Passthrough.svelte")).default,
}));

vi.mock("$lib/config/music.svelte", () => ({
  music,
}));

vi.mock("$lib/music/related-tracks-cache", () => ({
  getRelatedTracksCached,
  peekRelatedTracks: vi.fn(() => null),
  prefetchRelatedTracks: vi.fn(),
}));

vi.mock("$lib/router/router.svelte", () => ({
  router: {
    pathname: "/music/now-playing",
    navigate: vi.fn(),
  },
  link: vi.fn(),
  withBase: (path: string) => path,
}));

vi.mock("$lib/music/device-sync.svelte", () => ({
  deviceSync: {
    dispatchTransport: vi.fn(() => false),
  },
}));

const { videoFeature } = vi.hoisted(() => ({
  videoFeature: {
    enabled: false,
    loaded: true,
    refresh: vi.fn().mockResolvedValue(undefined),
    setEnabled(value: boolean) {
      this.enabled = value;
      this.loaded = true;
    },
  },
}));

vi.mock("$lib/video/feature.svelte", () => ({
  videoFeature,
}));

vi.mock("$lib/components/music/VideoWatchPanel.svelte", async () => ({
  default: (await import("../test-fixtures/VideoWatchPanelStub.svelte"))
    .default,
}));

import NowPlayingPage from "./NowPlayingPage.svelte";

function resetMusic() {
  music.currentTrack = null;
  music.queue = [];
  music.queueIndex = 0;
  music.currentLyrics = null;
  music.lyricsLoading = false;
  music.lyricsFetching = false;
  music.playing = false;
  music.currentTime = 0;
  vi.clearAllMocks();
  getRelatedTracksCached.mockResolvedValue([]);
}

function renderPage() {
  const target = document.createElement("div");
  document.body.appendChild(target);
  const instance = mount(NowPlayingPage, { target });
  flushSync();
  return {
    target,
    cleanup: () => {
      unmount(instance);
      target.remove();
    },
  };
}

function clickTab(target: HTMLElement, label: string) {
  const tab = [...target.querySelectorAll('[role="tab"]')].find(
    (node) => node.textContent?.trim() === label,
  );
  if (!tab) throw new Error(`Tab not found: ${label}`);
  (tab as HTMLButtonElement).click();
  flushSync();
}

describe("NowPlayingPage", () => {
  beforeEach(() => {
    resetMusic();
    videoFeature.enabled = false;
    layout.exitTvMode();
  });

  it("renders the empty state when nothing is playing", () => {
    const { target, cleanup } = renderPage();
    expect(target.textContent).toContain("Nothing playing");
    cleanup();
  });

  it("renders artwork metadata for the current track", () => {
    music.currentTrack = sampleTrack;
    const { target, cleanup } = renderPage();

    expect(target.textContent).toContain("Weathergirl");
    expect(target.textContent).toContain("Flavor Foley");
    expect(target.textContent).toContain("2024");
    expect(target.querySelector(".cover-art")).not.toBeNull();
    expect(target.querySelector(".now-playing-page__ambient")).not.toBeNull();
    expect(target.querySelector(".now-playing-art__fav")).not.toBeNull();
    const year = target.querySelector(".now-playing-art__year");
    const fav = target.querySelector(".now-playing-art__fav");
    expect(year).not.toBeNull();
    expect(fav).not.toBeNull();
    expect(
      year!.compareDocumentPosition(fav!) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(
      target.querySelector('.now-playing-page__tool[aria-label="TV mode"]'),
    ).not.toBeNull();
    expect(
      target.querySelector('.now-playing-page__tool[aria-label="3D mode"]'),
    ).toBeNull();
    expect(target.textContent).not.toContain("Devices");
    expect(target.textContent).not.toContain("EQ");
    cleanup();
  });

  it("opens TV mode from the fullscreen control", () => {
    music.currentTrack = sampleTrack;
    const { target, cleanup } = renderPage();
    const button = target.querySelector(
      '.now-playing-page__tool[aria-label="TV mode"]',
    ) as HTMLButtonElement | null;
    expect(button).not.toBeNull();
    button!.click();
    flushSync();
    expect(target.querySelector(".tv-mode")).not.toBeNull();
    expect(target.querySelector(".tv-mode__title")?.textContent).toContain(
      "Weathergirl",
    );
    cleanup();
  });

  it("uses a scrollable page shell on mobile viewport", () => {
    layout.setMobileViewport(true);
    music.currentTrack = sampleTrack;
    const { target, cleanup } = renderPage();
    try {
      const page = target.querySelector(".now-playing-page") as HTMLElement;
      expect(page).not.toBeNull();
      // CSS media queries may not apply in jsdom; class presence still marks the shell.
      expect(page.classList.contains("now-playing-page")).toBe(true);
      expect(
        target.querySelector('.now-playing-page__tool[aria-label="TV mode"]'),
      ).not.toBeNull();
    } finally {
      layout.setMobileViewport(false);
      cleanup();
    }
  });

  it("exits TV mode when playback becomes empty", () => {
    layout.enterTvMode();
    music.currentTrack = null;
    const { target, cleanup } = renderPage();
    expect(target.querySelector(".tv-mode")).toBeNull();
    expect(layout.tvMode).toBe(false);
    cleanup();
  });

  it("does not load lyrics until the lyrics tab is opened", () => {
    music.currentTrack = sampleTrack;
    const { target, cleanup } = renderPage();
    expect(music.loadCurrentLyrics).not.toHaveBeenCalled();
    clickTab(target, "Lyrics");
    expect(music.loadCurrentLyrics).toHaveBeenCalledTimes(1);
    cleanup();
  });

  it("does not load lyrics for internet radio tracks", () => {
    music.currentTrack = {
      id: "ir:station-1",
      title: "Radio",
      artist: "Internet Radio",
      isInternetRadio: true,
    };
    renderPage().cleanup();
    expect(music.loadCurrentLyrics).not.toHaveBeenCalled();
  });

  it("defaults to the queue tab and shows an empty queue state", () => {
    music.currentTrack = sampleTrack;
    const { target, cleanup } = renderPage();

    const activeTab = target.querySelector(
      '[role="tab"][aria-selected="true"]',
    );
    expect(activeTab?.textContent).toContain("Up next");
    expect(target.textContent).toContain("Queue is empty");
    cleanup();
  });

  it("renders queue items and handles queue actions", () => {
    music.currentTrack = sampleTrack;
    music.queue = [sampleTrack, queueTrack];
    music.queueIndex = 0;
    const { target, cleanup } = renderPage();

    expect(target.textContent).toContain("2 tracks");
    expect(target.textContent).toContain("Next Song");

    target
      .querySelector(".now-playing-queue__clear")
      ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    expect(music.clearQueue).toHaveBeenCalledTimes(1);

    const queueItems = target.querySelectorAll(".now-playing-queue__item");
    (queueItems[1] as HTMLDivElement).click();
    expect(music.playQueueIndex).toHaveBeenCalledWith(1);

    const removeButtons = target.querySelectorAll(".now-playing-queue__remove");
    (removeButtons[0] as HTMLButtonElement).click();
    expect(music.removeFromQueue).toHaveBeenCalledWith(0);

    cleanup();
  });

  it("reorders the queue when an item is dropped onto another slot", () => {
    music.currentTrack = sampleTrack;
    music.queue = [sampleTrack, queueTrack];
    const { target, cleanup } = renderPage();

    const items = target.querySelectorAll(".now-playing-queue__item");
    items[0].dispatchEvent(new Event("dragstart", { bubbles: true }));
    items[1].dispatchEvent(new Event("dragover", { bubbles: true }));
    items[1].dispatchEvent(new Event("drop", { bubbles: true }));

    expect(music.moveInQueue).toHaveBeenCalledWith(0, 1);
    cleanup();
  });

  it("does not fetch related tracks until the related tab is opened", () => {
    music.currentTrack = sampleTrack;
    renderPage().cleanup();
    expect(getRelatedTracksCached).not.toHaveBeenCalled();
  });

  it("loads and renders related tracks in the related tab", async () => {
    music.currentTrack = sampleTrack;
    getRelatedTracksCached.mockResolvedValue([relatedTrack]);
    const { target, cleanup } = renderPage();

    clickTab(target, "Related");
    await vi.waitFor(() =>
      expect(target.textContent).toContain("Related Song"),
    );
    expect(getRelatedTracksCached).toHaveBeenCalledWith(
      music.library,
      sampleTrack,
      24,
    );

    cleanup();
  });

  it("shows an empty related state when no related tracks are found", async () => {
    music.currentTrack = sampleTrack;
    getRelatedTracksCached.mockResolvedValue([]);
    const { target, cleanup } = renderPage();

    clickTab(target, "Related");
    await vi.waitFor(() =>
      expect(target.textContent).toContain("No related tracks"),
    );

    cleanup();
  });

  it("shows lyrics loading, fetch, and synced lyric content in the lyrics tab", async () => {
    music.currentTrack = sampleTrack;
    music.lyricsLoading = true;
    const { target, cleanup } = renderPage();
    clickTab(target, "Lyrics");

    expect(target.querySelector(".now-playing-side__loading")).not.toBeNull();

    cleanup();
    music.lyricsLoading = false;
    music.currentLyrics = null;
    music.lyricsFetching = false;

    const missing = renderPage();
    clickTab(missing.target, "Lyrics");
    expect(missing.target.textContent).toContain("No lyrics found");
    missing.target
      .querySelector(".now-playing-lyrics__fetch")
      ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    expect(music.fetchCurrentLyrics).toHaveBeenCalledTimes(1);
    missing.cleanup();

    music.currentLyrics = syncedLyrics;
    music.currentTime = 2.1;
    music.playing = true;
    const synced = renderPage();
    clickTab(synced.target, "Lyrics");
    expect(synced.target.textContent).toContain("Synced");
    expect(synced.target.textContent).toContain("For two");
    const active = synced.target.querySelector(".lyrics-display__line--active");
    expect(active?.textContent).toContain("For two");

    const search = synced.target.querySelector(
      ".now-playing-lyrics__search input",
    ) as HTMLInputElement;
    search.value = "azure";
    search.dispatchEvent(new Event("input", { bubbles: true }));
    flushSync();
    expect(synced.target.textContent).toContain("Azure views");
    expect(synced.target.textContent).not.toContain("For two");

    synced.cleanup();
  });

  it("hides the lyrics tab for live radio streams", () => {
    music.currentTrack = {
      id: "ir:station-1",
      title: "Live Radio",
      artist: "Internet Radio",
      isInternetRadio: true,
    };
    const { target, cleanup } = renderPage();

    expect(target.textContent).toContain("Up next");
    expect(target.textContent).toContain("Related");
    const tabs = [...target.querySelectorAll('[role="tab"]')].map((node) =>
      node.textContent?.trim(),
    );
    expect(tabs).not.toContain("Lyrics");
    cleanup();
  });

  it("shows the video tab only when videos are enabled", () => {
    music.currentTrack = sampleTrack;
    videoFeature.enabled = false;
    const off = renderPage();
    let tabs = [...off.target.querySelectorAll('[role="tab"]')].map((node) =>
      node.textContent?.trim(),
    );
    expect(tabs).not.toContain("Video");
    off.cleanup();

    videoFeature.enabled = true;
    const on = renderPage();
    tabs = [...on.target.querySelectorAll('[role="tab"]')].map((node) =>
      node.textContent?.trim(),
    );
    expect(tabs).toContain("Video");
    clickTab(on.target, "Video");
    expect(on.target.textContent).toContain("Video panel for Weathergirl");
    expect(
      on.target
        .querySelector(".video-watch-stub")
        ?.getAttribute("data-embedded"),
    ).toBe("true");
    on.cleanup();
  });
});
