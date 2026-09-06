// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from "vitest";
import { flushSync, mount, unmount } from "svelte";

vi.hoisted(() => {
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

vi.mock("$lib/config/music.svelte", () => ({
  music: {
    currentTrack: {
      id: "trk_1",
      title: "Weathergirl",
      artist: "Flavor Foley",
      album: "Weathergirl",
      albumId: "alb_1",
      coverArt: "cov_1",
      duration: 258,
    },
    playerLayout: "full",
    playerVisible: true,
    playing: true,
    currentTime: 42,
    duration: 258,
    smoothProgress: 16,
    onPlayRoute: false,
    togglePlay: vi.fn(),
    next: vi.fn(),
    seek: vi.fn(),
    restorePlayer: vi.fn(),
    config: {
      serverUrl: "/api/subsonic",
      username: "u",
      password: "p",
      version: "1.16.1",
      clientName: "melovian",
    },
  },
}));

vi.mock("$lib/music/device-sync.svelte", () => ({
  deviceSync: {
    dispatchTransport: () => false,
  },
}));

vi.mock("$lib/components/ui/EnhancedCoverArt.svelte", async () => ({
  default: (await import("../../../test-fixtures/Passthrough.svelte")).default,
}));

vi.mock("$lib/router/Link.svelte", async () => ({
  default: (await import("../../../test-fixtures/Passthrough.svelte")).default,
}));

import MusicSlimPlayer from "./MusicSlimPlayer.svelte";

describe("MusicSlimPlayer", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
  });

  it("renders progress seek and elapsed time on the mobile bar", () => {
    const target = document.createElement("div");
    document.body.appendChild(target);
    const instance = mount(MusicSlimPlayer, { target });
    flushSync();
    try {
      expect(target.querySelector(".progress-seek")).not.toBeNull();
      expect(target.querySelector(".slim-player__time")?.textContent).toMatch(
        /\d+:\d+\s*\/\s*\d+:\d+/,
      );
    } finally {
      unmount(instance);
      target.remove();
    }
  });
});
