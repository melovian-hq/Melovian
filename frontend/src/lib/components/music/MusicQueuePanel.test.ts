// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { flushSync, mount, unmount } from "svelte";
import type { QueueTrack } from "$lib/subsonic";
import {
  loadQueuePanelPosition,
  saveQueuePanelPosition,
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

  class ResizeObserverPolyfill {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  globalThis.ResizeObserver =
    ResizeObserverPolyfill as unknown as typeof ResizeObserver;

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

const tracks: QueueTrack[] = [
  { id: "trk_1", title: "One", artist: "A", duration: 180 },
  { id: "trk_2", title: "Two", artist: "B", duration: 200 },
];

const { music } = vi.hoisted(() => {
  const music = {
    queue: [] as QueueTrack[],
    queueIndex: 0,
    queueOpen: true,
    playing: false,
    continuousMode: "off" as const,
    onPlayRoute: false,
    onNowPlayingRoute: false,
    config: { serverUrl: "", username: "", password: "", token: "", salt: "" },
    moveInQueue: vi.fn(),
    playQueueIndex: vi.fn(),
    removeFromQueue: vi.fn(),
    clearQueue: vi.fn(),
    clearContinuousMode: vi.fn(),
  };
  return { music };
});

vi.mock("$lib/config/music.svelte", () => ({
  music,
  CONTINUOUS_MODE_LABELS: {
    off: "Off",
    random: "Random radio",
    library: "Library shuffle",
    personal: "Personal radio",
  },
}));

describe("MusicQueuePanel drag regression", () => {
  let MusicQueuePanel: typeof import("./MusicQueuePanel.svelte").default;

  beforeEach(async () => {
    localStorage.clear();
    music.queue = [...tracks];
    music.queueOpen = true;
    music.onPlayRoute = false;
    music.onNowPlayingRoute = false;
    music.moveInQueue.mockClear();
    music.clearQueue.mockClear();

    Element.prototype.setPointerCapture = vi.fn();
    Element.prototype.releasePointerCapture = vi.fn();
    Element.prototype.hasPointerCapture = vi.fn(() => false);

    MusicQueuePanel = (await import("./MusicQueuePanel.svelte")).default;
  });

  afterEach(() => {
    localStorage.clear();
  });

  function renderPanel() {
    const target = document.createElement("div");
    document.body.appendChild(target);
    const instance = mount(MusicQueuePanel, { target });
    flushSync();
    const panel = target.querySelector(".queue-panel") as HTMLElement;
    const header = target.querySelector(".queue-panel__header") as HTMLElement;

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
    expect(header.classList.contains("queue-panel__header--draggable")).toBe(
      true,
    );
    expect(header.getAttribute("aria-label")).toBe("Move queue panel");
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
    expect(panel.classList.contains("queue-panel--positioned")).toBe(true);
    expect(panel.classList.contains("queue-panel--dragging")).toBe(true);

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
    expect(panel.classList.contains("queue-panel--dragging")).toBe(false);
    expect(loadQueuePanelPosition()).toEqual({ x: 100, y: 140 });
    cleanup();
  });

  it("ignores pointer downs on action buttons", () => {
    saveQueuePanelPosition({ x: 50, y: 60 });
    const { panel, header, cleanup } = renderPanel();
    const close = header.querySelector(
      ".queue-panel__close",
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
    expect(panel.classList.contains("queue-panel--dragging")).toBe(false);
    cleanup();
  });

  it("renders on the now playing route when the queue is open", () => {
    music.onNowPlayingRoute = true;
    const { panel, cleanup } = renderPanel();
    expect(panel).toBeTruthy();
    cleanup();
  });
});
