// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from "vitest";
import {
  attemptStaleReload,
  configureStaleChunkHandling,
  isStaleChunkError,
  resetStaleChunkForTests,
} from "./stale-chunk";

describe("isStaleChunkError", () => {
  it.each([
    "Failed to fetch dynamically imported module: https://x/assets/SettingsPage-C_Cl7oGw.js",
    "error loading dynamically imported module: https://x/assets/HomePage-A1B2.js",
    "Importing a module script failed.",
    "Unable to preload CSS for /assets/index-abc.css",
  ])("matches %s", (message) => {
    expect(isStaleChunkError(new TypeError(message))).toBe(true);
  });

  it("matches a ChunkLoadError name", () => {
    const err = new Error("Loading chunk 42 failed");
    err.name = "ChunkLoadError";
    expect(isStaleChunkError(err)).toBe(true);
  });

  it.each([
    "Failed to fetch",
    "NetworkError when attempting to fetch resource.",
    "Could not reach the server",
    "401 Unauthorized",
  ])("rejects %s", (message) => {
    expect(isStaleChunkError(new TypeError(message))).toBe(false);
  });

  it("rejects nullish and non-error values", () => {
    expect(isStaleChunkError(null)).toBe(false);
    expect(isStaleChunkError(undefined)).toBe(false);
    expect(isStaleChunkError({})).toBe(false);
  });
});

describe("attemptStaleReload", () => {
  afterEach(() => {
    vi.useRealTimers();
    resetStaleChunkForTests();
  });

  function armed() {
    const reload = vi.fn();
    configureStaleChunkHandling({ reload });
    return reload;
  }

  it("schedules a reload and reports reloading", () => {
    vi.useFakeTimers();
    const reload = armed();
    expect(attemptStaleReload()).toBe("reloading");
    expect(reload).not.toHaveBeenCalled();
    vi.runAllTimers();
    expect(reload).toHaveBeenCalledTimes(1);
  });

  it("stays scheduled on repeat calls in the same view", () => {
    vi.useFakeTimers();
    const reload = armed();
    attemptStaleReload();
    expect(attemptStaleReload()).toBe("reloading");
    vi.runAllTimers();
    expect(reload).toHaveBeenCalledTimes(1);
  });

  it("defers while audio is playing", () => {
    vi.useFakeTimers();
    const reload = vi.fn();
    configureStaleChunkHandling({ reload, isPlaying: () => true });
    expect(attemptStaleReload()).toBe("deferred");
    vi.runAllTimers();
    expect(reload).not.toHaveBeenCalled();
  });

  it("is limited when a reload ran within the guard window", () => {
    vi.useFakeTimers();
    const reload = vi.fn();
    configureStaleChunkHandling({ reload });
    // A prior page view reloaded for a stale chunk seconds ago.
    sessionStorage.setItem("melovian-stale-reload", String(Date.now()));
    expect(attemptStaleReload()).toBe("limited");
    vi.runAllTimers();
    expect(reload).not.toHaveBeenCalled();
  });

  it("allows another reload after the guard window passes", () => {
    vi.useFakeTimers();
    const reload = vi.fn();
    configureStaleChunkHandling({ reload });
    sessionStorage.setItem(
      "melovian-stale-reload",
      String(Date.now() - 61_000),
    );
    expect(attemptStaleReload()).toBe("reloading");
    vi.runAllTimers();
    expect(reload).toHaveBeenCalledTimes(1);
  });
});
