// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { GeneratedMix } from "./mix-generator";
import {
  fromCachedMix,
  normalizeLoadedMix,
  toCachedMix,
  type CachedMix,
  type SlimPersonalMix,
} from "./mix-storage";

const CACHE_KEY = "mel-music-mixes";

export interface MixCacheEntry {
  serverUrl: string;
  daySeed: string;
  mixes: SlimPersonalMix[];
  cachedAt: number;
}

export function mixCacheKey(serverUrl: string): string {
  return serverUrl.trim().replace(/\/+$/, "").toLowerCase();
}

export function loadMixCache(serverUrl: string): MixCacheEntry | null {
  if (typeof localStorage === "undefined" || !serverUrl) return null;
  try {
    const raw = localStorage.getItem(CACHE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as {
      serverUrl: string;
      daySeed: string;
      mixes: Array<GeneratedMix | CachedMix | SlimPersonalMix>;
      cachedAt: number;
    };
    if (
      !parsed ||
      mixCacheKey(parsed.serverUrl) !== mixCacheKey(serverUrl) ||
      !Array.isArray(parsed.mixes) ||
      parsed.mixes.length === 0
    ) {
      return null;
    }
    return {
      serverUrl: parsed.serverUrl,
      daySeed: parsed.daySeed,
      mixes: parsed.mixes.map((mix) => normalizeLoadedMix(mix)),
      cachedAt: parsed.cachedAt,
    };
  } catch {
    return null;
  }
}

export function saveMixCache(
  serverUrl: string,
  daySeed: string,
  mixes: Array<GeneratedMix | SlimPersonalMix>,
): void {
  if (typeof localStorage === "undefined" || !serverUrl || mixes.length === 0) {
    return;
  }
  const entry: MixCacheEntry = {
    serverUrl,
    daySeed,
    mixes: mixes.map((mix) => fromCachedMix(toCachedMix(mix))),
    cachedAt: Date.now(),
  };
  try {
    localStorage.setItem(CACHE_KEY, JSON.stringify(entry));
  } catch {
    /* storage full or unavailable */
  }
}

export function isMixCacheFresh(
  entry: MixCacheEntry,
  daySeed: string,
  maxAgeMs = 6 * 60 * 60 * 1000,
): boolean {
  if (entry.daySeed !== daySeed) return false;
  return Date.now() - entry.cachedAt <= maxAgeMs;
}

/** True when a cache still has pre-multi-slot genre/decade mix ids. */
export function hasObsoleteMixIds(
  mixes: ReadonlyArray<{ id: string }>,
): boolean {
  return mixes.some((mix) => mix.id === "genre-mix" || mix.id === "decade-mix");
}

/** Fresh day cache must short-circuit Subsonic mix rebuild storms. */
export function shouldRebuildMixes(
  force: boolean,
  cached: MixCacheEntry | null,
  daySeed: string,
): boolean {
  if (force) return true;
  if (cached && hasObsoleteMixIds(cached.mixes)) return true;
  if (cached && isMixCacheFresh(cached, daySeed)) return false;
  return true;
}

export function clearMixCache(serverUrl: string): void {
  if (typeof localStorage === "undefined" || !serverUrl) return;
  const cached = loadMixCache(serverUrl);
  if (!cached) return;
  try {
    localStorage.removeItem(CACHE_KEY);
  } catch {
    /* storage unavailable */
  }
}

export function recentMixTrackIds(serverUrl: string): Set<string> {
  const entry = loadMixCache(serverUrl);
  if (!entry) return new Set();
  const ids = new Set<string>();
  for (const mix of entry.mixes) {
    for (const id of mix.trackIds ?? []) {
      if (id) ids.add(id);
    }
  }
  return ids;
}
