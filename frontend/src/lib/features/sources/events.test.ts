// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

const handlers = vi.hoisted(
  () => new Map<string, Set<(event: { type: string; payload: unknown }) => void>>(),
);
const bustMock = vi.hoisted(() => vi.fn());
const refreshHomeMock = vi.hoisted(() => vi.fn());
const musicMock = vi.hoisted(() => ({ refreshHome: refreshHomeMock }));
const sourcesMock = vi.hoisted(() => ({ mode: "subsonic" as string }));

vi.mock("$lib/core/events/ws.svelte", () => ({
  eventSocket: {
    on(type: string, handler: (event: { type: string; payload: unknown }) => void) {
      let set = handlers.get(type);
      if (!set) {
        set = new Set();
        handlers.set(type, set);
      }
      set.add(handler);
      return () => set?.delete(handler);
    },
  },
}));

vi.mock("$lib/music/api", () => ({ bustLibraryCache: bustMock }));
vi.mock("$lib/config/music.svelte", () => ({ music: musicMock }));
vi.mock("$lib/features/sources/store.svelte", () => ({ sources: sourcesMock }));

import { bindSourceEvents } from "./events";

function emit(type: string, payload: unknown) {
  handlers.get(type)?.forEach((h) => h({ type, payload }));
}

describe("bindSourceEvents", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    handlers.clear();
    bustMock.mockReset().mockResolvedValue(undefined);
    refreshHomeMock.mockReset().mockResolvedValue(undefined);
    sourcesMock.mode = "subsonic";
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("refreshes when a navidrome scan finishes", async () => {
    const unbind = bindSourceEvents();
    emit("source.navidrome.scanStatus", { scanning: true });
    emit("source.navidrome.scanStatus", { scanning: false, count: 10 });
    await vi.advanceTimersByTimeAsync(2100);
    expect(bustMock).toHaveBeenCalledTimes(1);
    expect(refreshHomeMock).toHaveBeenCalledTimes(1);
    unbind();
  });

  it("ignores scan progress ticks that stay scanning", async () => {
    const unbind = bindSourceEvents();
    emit("source.navidrome.scanStatus", { scanning: true });
    emit("source.navidrome.scanStatus", { scanning: true });
    await vi.advanceTimersByTimeAsync(2100);
    expect(bustMock).not.toHaveBeenCalled();
    unbind();
  });

  it("debounces refreshResource bursts", async () => {
    const unbind = bindSourceEvents();
    emit("source.navidrome.refreshResource", { resource: "song" });
    emit("source.navidrome.refreshResource", { resource: "album" });
    emit("source.navidrome.refreshResource", { resource: "playlist" });
    await vi.advanceTimersByTimeAsync(2100);
    expect(bustMock).toHaveBeenCalledTimes(1);
    unbind();
  });

  it("skips refresh while browsing a local-only view", async () => {
    sourcesMock.mode = "local";
    const unbind = bindSourceEvents();
    emit("source.navidrome.serverStart", {});
    await vi.advanceTimersByTimeAsync(2100);
    expect(bustMock).not.toHaveBeenCalled();
    unbind();
  });

  it("unsubscribes cleanly", async () => {
    const unbind = bindSourceEvents();
    unbind();
    emit("source.navidrome.serverStart", {});
    await vi.advanceTimersByTimeAsync(2100);
    expect(bustMock).not.toHaveBeenCalled();
  });
});
