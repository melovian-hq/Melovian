// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { ListenEntry, ListenStats } from "$lib/subsonic/types";

export type StatsPeriod = "all" | "7d" | "30d" | "90d" | `year:${number}`;

export const STATS_PERIOD_LABELS: Record<
  Exclude<StatsPeriod, `year:${number}`>,
  string
> = {
  all: "All time",
  "7d": "This week",
  "30d": "This month",
  "90d": "3 months",
};

export function statsYearLabel(year: number): string {
  return String(year);
}

export function statsPeriodYear(period: StatsPeriod): number | null {
  if (!period.startsWith("year:")) return null;
  const year = Number(period.slice(5));
  return Number.isFinite(year) ? year : null;
}

export function statsPeriodForYear(year: number): StatsPeriod {
  return `year:${year}`;
}

const PERIOD_MS: Record<"7d" | "30d" | "90d", number> = {
  "7d": 7 * 86_400_000,
  "30d": 30 * 86_400_000,
  "90d": 90 * 86_400_000,
};

export function filterHistoryByPeriod(
  history: readonly ListenEntry[],
  period: StatsPeriod,
): ListenEntry[] {
  const year = statsPeriodYear(period);
  if (year !== null) return filterHistoryByYear(history, year);
  if (period === "all") return [...history];
  const cutoff = Date.now() - PERIOD_MS[period as keyof typeof PERIOD_MS];
  return history.filter(
    (entry) => new Date(entry.lastPlayedAt).getTime() >= cutoff,
  );
}

export function filterHistoryByYear(
  history: readonly ListenEntry[],
  year: number,
): ListenEntry[] {
  const start = new Date(year, 0, 1).getTime();
  const end = new Date(year + 1, 0, 1).getTime();
  return history.filter((entry) => {
    const playedAt = new Date(entry.lastPlayedAt).getTime();
    return playedAt >= start && playedAt < end;
  });
}

export function uniqueRecentHistory(
  history: readonly ListenEntry[],
): ListenEntry[] {
  const seen = new Set<string>();
  const unique: ListenEntry[] = [];
  const sorted = [...history].sort(
    (a, b) =>
      new Date(b.lastPlayedAt).getTime() - new Date(a.lastPlayedAt).getTime(),
  );
  for (const entry of sorted) {
    if (seen.has(entry.trackId)) continue;
    seen.add(entry.trackId);
    unique.push(entry);
  }
  return unique;
}

export function computeStatsFromHistory(
  history: readonly ListenEntry[],
  limit = 12,
): ListenStats {
  const totalPlays = history.reduce((sum, e) => sum + e.playCount, 0);
  const uniqueTracks = new Set(history.map((e) => e.trackId)).size;
  const totalListeningMs = history.reduce((sum, e) => {
    if ((e.listenedMs ?? 0) > 0) return sum + e.listenedMs;
    return sum + e.durationMs * e.playCount;
  }, 0);

  const artistCounts = new Map<string, { label: string; count: number }>();
  const trackCounts = new Map<string, { label: string; count: number }>();
  const albumCounts = new Map<string, { label: string; count: number }>();

  for (const entry of history) {
    const weight =
      (entry.listenedMs ?? 0) > 0
        ? Math.max(1, Math.round(entry.listenedMs / 1000))
        : entry.playCount;
    if (entry.artistName) {
      const current = artistCounts.get(entry.artistName) ?? {
        label: entry.artistName,
        count: 0,
      };
      current.count += weight;
      artistCounts.set(entry.artistName, current);
    }
    if (entry.trackTitle) {
      const current = trackCounts.get(entry.trackId) ?? {
        label: entry.trackTitle,
        count: 0,
      };
      current.count += weight;
      trackCounts.set(entry.trackId, current);
    }
    if (entry.albumTitle && entry.albumId) {
      const current = albumCounts.get(entry.albumId) ?? {
        label: entry.albumTitle,
        count: 0,
      };
      current.count += weight;
      albumCounts.set(entry.albumId, current);
    }
  }

  const sortTop = (entries: [string, { label: string; count: number }][]) =>
    entries
      .sort((a, b) => b[1].count - a[1].count)
      .slice(0, limit)
      .map(([key, value]) => ({ key, label: value.label, count: value.count }));

  return {
    totalPlays,
    uniqueTracks,
    totalListeningMs,
    topArtists: sortTop([...artistCounts.entries()]),
    topTracks: sortTop([...trackCounts.entries()]),
    topAlbums: sortTop([...albumCounts.entries()]),
  };
}
