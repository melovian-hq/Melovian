// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { persistPlaybackSnapshot, flushPlaybackState } = vi.hoisted(() => ({
  persistPlaybackSnapshot: vi.fn(),
  flushPlaybackState: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("$lib/config/music.svelte", () => ({
  music: {
    persistPlaybackSnapshot,
    flushPlaybackState,
  },
}));

import { bindPlaybackLifecycle } from "./playback-lifecycle";

describe("playback-lifecycle", () => {
  let originalVisibilityState: DocumentVisibilityState;

  beforeEach(() => {
    originalVisibilityState = document.visibilityState;
    vi.clearAllMocks();
  });

  afterEach(() => {
    Object.defineProperty(document, "visibilityState", {
      configurable: true,
      value: originalVisibilityState,
    });
    vi.useRealTimers();
  });

  it("flushes playback state on pagehide", () => {
    const unbind = bindPlaybackLifecycle();
    window.dispatchEvent(new Event("pagehide"));

    expect(persistPlaybackSnapshot).toHaveBeenCalledOnce();
    expect(flushPlaybackState).toHaveBeenCalledOnce();
    unbind();
  });

  it("flushes playback state on beforeunload", () => {
    const unbind = bindPlaybackLifecycle();
    window.dispatchEvent(new Event("beforeunload"));

    expect(persistPlaybackSnapshot).toHaveBeenCalledOnce();
    expect(flushPlaybackState).toHaveBeenCalledOnce();
    unbind();
  });

  it("flushes when the document becomes hidden", () => {
    const unbind = bindPlaybackLifecycle();
    Object.defineProperty(document, "visibilityState", {
      configurable: true,
      value: "hidden",
    });
    document.dispatchEvent(new Event("visibilitychange"));

    expect(persistPlaybackSnapshot).toHaveBeenCalledOnce();
    expect(flushPlaybackState).toHaveBeenCalledOnce();
    unbind();
  });

  it("does not flush when the document becomes visible", () => {
    const unbind = bindPlaybackLifecycle();
    Object.defineProperty(document, "visibilityState", {
      configurable: true,
      value: "visible",
    });
    document.dispatchEvent(new Event("visibilitychange"));

    expect(persistPlaybackSnapshot).not.toHaveBeenCalled();
    expect(flushPlaybackState).not.toHaveBeenCalled();
    unbind();
  });

  it("stops flushing after cleanup", () => {
    const unbind = bindPlaybackLifecycle();
    unbind();

    window.dispatchEvent(new Event("pagehide"));

    expect(persistPlaybackSnapshot).not.toHaveBeenCalled();
    expect(flushPlaybackState).not.toHaveBeenCalled();
  });
});
