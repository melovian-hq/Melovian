// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { flushSync, mount, unmount } from "svelte";
import type { ParsedLyrics } from "$lib/music/lyrics";
import type { QueueTrack } from "$lib/subsonic";
import {
  loadLyricsPanelPosition,
  saveLyricsPanelPosition,
} from "$lib/music/prefs";

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

  if (typeof globalThis.PointerEvent === "undefined") {
    class PointerEventPolyfill extends MouseEvent {
      pointerId: number;
      constructor(type: string, init: PointerEventInit = {}) {
        super(type, init);
        this.pointerId = init.pointerId ?? 0;
      }
    }
    globalThis.PointerEvent =
      PointerEventPolyfill as unknown as typeof PointerEvent;
  }
});

const syncedLyrics: ParsedLyrics = {
  synced: true,
  offsetMs: 0,
  title: "Song",
  artist: "Artist",
  rawValue: "[00:00.00]One\n[00:02.00]Two",
  lines: [
    { startMs: 0, text: "One" },
    { startMs: 2000, text: "Two" },
  ],
};

const track: QueueTrack = {
  id: "trk_1",
  title: "Song",
  artist: "Artist",
  duration: 180,
};

const { music } = vi.hoisted(() => {
  const music = {
    currentTrack: null as QueueTrack | null,
    currentLyrics: null as ParsedLyrics | null,
    lyricsLoading: false,
    lyricsFetching: false,
    playing: false,
    currentTime: 0,
    playerVisible: true,
    playerLayout: "compact" as const,
    fetchCurrentLyrics: vi.fn().mockResolvedValue(null),
    toggleLyricsPanel: vi.fn(),
    seekToLyricLine: vi.fn(),
  };
  return { music };
});

vi.mock("$lib/config/music.svelte", () => ({ music }));
vi.mock("$lib/router/router.svelte", () => ({
  router: {
    pathname: "/music",
    navigate: vi.fn(),
  },
  link: () => {},
  withBase: (path: string) => path,
}));

describe("LyricsPanel drag regression", () => {
  let LyricsPanel: typeof import("./LyricsPanel.svelte").default;

  beforeEach(async () => {
    localStorage.clear();
    music.currentTrack = track;
    music.currentLyrics = syncedLyrics;
    music.lyricsLoading = false;
    music.lyricsFetching = false;
    music.fetchCurrentLyrics.mockClear();
    music.toggleLyricsPanel.mockClear();

    Element.prototype.setPointerCapture = vi.fn();
    Element.prototype.releasePointerCapture = vi.fn();
    Element.prototype.hasPointerCapture = vi.fn(() => false);

    LyricsPanel = (await import("./LyricsPanel.svelte")).default;
  });

  afterEach(() => {
    localStorage.clear();
  });

  function renderPanel() {
    const target = document.createElement("div");
    document.body.appendChild(target);
    const instance = mount(LyricsPanel, { target });
    flushSync();
    const panel = target.querySelector(".lyrics-sidebar") as HTMLElement;
    const header = target.querySelector(
      ".lyrics-sidebar__header",
    ) as HTMLElement;

    Object.defineProperty(panel, "getBoundingClientRect", {
      configurable: true,
      value: () => ({
        left: 40,
        top: 80,
        right: 392,
        bottom: 528,
        width: 352,
        height: 448,
        x: 40,
        y: 80,
        toJSON: () => ({}),
      }),
    });
    Object.defineProperty(panel, "offsetWidth", {
      configurable: true,
      value: 352,
    });
    Object.defineProperty(panel, "offsetHeight", {
      configurable: true,
      value: 448,
    });

    return {
      target,
      panel,
      header,
      cleanup: () => {
        unmount(instance);
        target.remove();
      },
    };
  }

  it("marks the header as a desktop drag surface", () => {
    const { header, cleanup } = renderPanel();
    expect(header.classList.contains("lyrics-sidebar__header--draggable")).toBe(
      true,
    );
    expect(header.getAttribute("aria-label")).toBe("Move lyrics panel");
    cleanup();
  });

  it("moves and persists position while dragging the header", () => {
    const { panel, header, cleanup } = renderPanel();

    header.dispatchEvent(
      new PointerEvent("pointerdown", {
        bubbles: true,
        clientX: 60,
        clientY: 100,
        button: 0,
        pointerId: 1,
      }),
    );
    flushSync();
    expect(panel.classList.contains("lyrics-sidebar--positioned")).toBe(true);
    expect(panel.classList.contains("lyrics-sidebar--dragging")).toBe(true);

    header.dispatchEvent(
      new PointerEvent("pointermove", {
        bubbles: true,
        clientX: 120,
        clientY: 160,
        pointerId: 1,
      }),
    );
    flushSync();
    expect(panel.style.left).toBe("100px");
    expect(panel.style.top).toBe("140px");

    header.dispatchEvent(
      new PointerEvent("pointerup", {
        bubbles: true,
        clientX: 120,
        clientY: 160,
        pointerId: 1,
      }),
    );
    flushSync();
    expect(panel.classList.contains("lyrics-sidebar--dragging")).toBe(false);
    expect(loadLyricsPanelPosition()).toEqual({ x: 100, y: 140 });
    cleanup();
  });

  it("ignores pointer downs on action buttons", () => {
    saveLyricsPanelPosition({ x: 50, y: 60 });
    const { panel, header, cleanup } = renderPanel();
    const close = header.querySelector(
      ".lyrics-sidebar__close",
    ) as HTMLButtonElement;

    close.dispatchEvent(
      new PointerEvent("pointerdown", {
        bubbles: true,
        clientX: 80,
        clientY: 90,
        button: 0,
        pointerId: 1,
      }),
    );
    flushSync();
    expect(panel.classList.contains("lyrics-sidebar--dragging")).toBe(false);
    cleanup();
  });
});
