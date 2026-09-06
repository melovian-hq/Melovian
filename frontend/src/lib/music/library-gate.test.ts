// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";

vi.mock("$lib/config/music.svelte", () => ({
  music: {
    error: null as string | null,
    loading: false,
    libraryReady: true,
  },
}));

vi.mock("$lib/features/sources/store.svelte", () => ({
  sources: {
    needsSetup: false,
    hasAnySource: true,
  },
}));

import { music } from "$lib/config/music.svelte";
import { sources } from "$lib/features/sources/store.svelte";
import { libraryUnavailable } from "./library-gate";

describe("libraryUnavailable", () => {
  it("is false when a ready library is present", () => {
    sources.needsSetup = false;
    sources.hasAnySource = true;
    music.error = null;
    music.loading = false;
    music.libraryReady = true;
    expect(libraryUnavailable()).toBe(false);
  });

  it("is true when setup is required", () => {
    sources.needsSetup = true;
    sources.hasAnySource = false;
    music.error = null;
    music.libraryReady = true;
    expect(libraryUnavailable()).toBe(true);
  });

  it("is true when connection failed", () => {
    sources.needsSetup = false;
    sources.hasAnySource = true;
    music.error = "Connection refused";
    music.libraryReady = false;
    music.loading = false;
    expect(libraryUnavailable()).toBe(true);
  });

  it("is true when no source is configured", () => {
    sources.needsSetup = false;
    sources.hasAnySource = false;
    music.error = null;
    music.libraryReady = true;
    expect(libraryUnavailable()).toBe(true);
  });

  it("is true when the library is missing and not loading", () => {
    sources.needsSetup = false;
    sources.hasAnySource = true;
    music.error = null;
    music.libraryReady = false;
    music.loading = false;
    expect(libraryUnavailable()).toBe(true);
  });
});
