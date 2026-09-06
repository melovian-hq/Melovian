// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export type CacheStrategy = "playback" | "most_listened" | "playlists" | "all";

export const CACHE_STRATEGY_LABELS: Record<CacheStrategy, string> = {
  playback: "While playing (current + next track)",
  most_listened: "Most listened",
  playlists: "Playlists",
  all: "All (until limit)",
};

export const DEFAULT_CACHE_LIMIT_BYTES = 2 * 1024 * 1024 * 1024;

export interface CacheSettings {
  enabled: boolean;
  limitBytes: number;
  strategy: CacheStrategy;
}

export function defaultCacheSettings(): CacheSettings {
  return {
    enabled: true,
    limitBytes: DEFAULT_CACHE_LIMIT_BYTES,
    strategy: "playback",
  };
}

const VALID_STRATEGIES = new Set<CacheStrategy>([
  "playback",
  "most_listened",
  "playlists",
  "all",
]);

export function mergeCacheSettings(
  partial: Partial<CacheSettings> | null | undefined,
): CacheSettings {
  if (!partial || typeof partial !== "object") {
    return defaultCacheSettings();
  }
  const defaults = defaultCacheSettings();
  const limitBytes =
    typeof partial.limitBytes === "number" && partial.limitBytes > 0
      ? partial.limitBytes
      : defaults.limitBytes;
  const strategy =
    partial.strategy && VALID_STRATEGIES.has(partial.strategy)
      ? partial.strategy
      : defaults.strategy;
  return {
    enabled:
      typeof partial.enabled === "boolean" ? partial.enabled : defaults.enabled,
    limitBytes,
    strategy,
  };
}

export { formatBytes } from "$lib/utils/format-bytes";
