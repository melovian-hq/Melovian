// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { GeneratedMix } from "./types";

export const MIX_PRIORITY = [
  "on-repeat",
  "replay",
  "for-you-50",
  "for-you-100",
  "daily-mix-1",
  "daily-mix-2",
  "daily-mix-3",
  "genre-mix-1",
  "genre-mix-2",
  "genre-mix-3",
  "discover",
  "deep-cuts",
  "release-radar",
  "throwback",
  "year-rewind",
  "popular",
  "your-artists",
  "overlooked",
  "favorites-mix",
  "decade-mix-1",
  "decade-mix-2",
  "decade-mix-3",
  "fresh-finds",
];

export function sortMixesByPriority(
  mixes: readonly GeneratedMix[],
): GeneratedMix[] {
  return [...mixes].sort(
    (a, b) =>
      (MIX_PRIORITY.indexOf(a.id) + 1 || 99) -
      (MIX_PRIORITY.indexOf(b.id) + 1 || 99),
  );
}

export function upsertMix(
  mixes: readonly GeneratedMix[],
  mix: GeneratedMix,
): GeneratedMix[] {
  const index = mixes.findIndex((entry) => entry.id === mix.id);
  if (index < 0) return sortMixesByPriority([...mixes, mix]);
  const next = [...mixes];
  next[index] = mix;
  return sortMixesByPriority(next);
}
