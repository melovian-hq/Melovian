// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, beforeEach } from "vitest";
import {
  isMixCacheFresh,
  loadMixCache,
  mixCacheKey,
  saveMixCache,
  shouldRebuildMixes,
} from "./mix-storage";
import type { GeneratedMix } from "./mix-generator";

const mix = (id: string): GeneratedMix => ({
  id,
  title: id,
  subtitle: "test",
  tracks: [{ id: `${id}-track`, title: "Track" }],
  gradient: "linear-gradient(red, blue)",
});

describe("mix-cache", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("normalizes server urls for lookup", () => {
    expect(mixCacheKey("https://music.example/")).toBe(
      mixCacheKey("https://music.example"),
    );
  });

  it("loads slim mixes for a server", () => {
    saveMixCache("https://music.example", "2026-06-23", [mix("discover")]);
    const cached = loadMixCache("https://music.example/");
    expect(cached?.mixes).toHaveLength(1);
    expect(cached?.mixes[0]?.id).toBe("discover");
    expect(cached?.mixes[0]?.trackIds).toEqual(["discover-track"]);
    expect(cached?.mixes[0]?.tracks).toEqual([]);
  });

  it("returns null for a different server", () => {
    saveMixCache("https://music.example", "2026-06-23", [mix("discover")]);
    expect(loadMixCache("https://other.example")).toBeNull();
  });

  it("checks freshness by day seed and age", () => {
    const entry = {
      serverUrl: "https://music.example",
      daySeed: "2026-06-23",
      mixes: [mix("discover")],
      cachedAt: Date.now(),
    };
    expect(isMixCacheFresh(entry, "2026-06-23")).toBe(true);
    expect(isMixCacheFresh(entry, "2026-06-24")).toBe(false);
    expect(
      isMixCacheFresh(
        { ...entry, cachedAt: Date.now() - 7 * 60 * 60 * 1000 },
        "2026-06-23",
      ),
    ).toBe(false);
  });

  it("short-circuits Subsonic rebuild when day cache is fresh", () => {
    const entry = {
      serverUrl: "https://music.example",
      daySeed: "2026-06-23:0",
      mixes: [mix("discover")],
      cachedAt: Date.now(),
    };
    expect(shouldRebuildMixes(false, entry, "2026-06-23:0")).toBe(false);
    expect(shouldRebuildMixes(true, entry, "2026-06-23:0")).toBe(true);
    expect(shouldRebuildMixes(false, null, "2026-06-23:0")).toBe(true);
    expect(shouldRebuildMixes(false, entry, "2026-06-24:0")).toBe(true);
  });

  it("rebuilds when cache still has legacy genre or decade mix ids", () => {
    const entry = {
      serverUrl: "https://music.example",
      daySeed: "2026-06-23:0",
      mixes: [mix("genre-mix"), mix("decade-mix")],
      cachedAt: Date.now(),
    };
    expect(shouldRebuildMixes(false, entry, "2026-06-23:0")).toBe(true);
  });
});
