// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { getActiveInstanceId } from "$lib/features/instances/context";

export type HiddenHomeKind = "album" | "mix" | "playlist" | "artist";

export interface HiddenHomeItems {
  albums: string[];
  mixes: string[];
  playlists: string[];
  artists: string[];
}

export const HOME_HIDDEN_KEY = "mel-home-hidden";

const KIND_FIELD: Record<HiddenHomeKind, keyof HiddenHomeItems> = {
  album: "albums",
  mix: "mixes",
  playlist: "playlists",
  artist: "artists",
};

export function homeHiddenStorageKey(
  instanceId = getActiveInstanceId(),
): string {
  return `${HOME_HIDDEN_KEY}:${instanceId ?? "default"}`;
}

export function emptyHiddenHome(): HiddenHomeItems {
  return { albums: [], mixes: [], playlists: [], artists: [] };
}

function readIds(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return [
    ...new Set(
      value.filter(
        (id): id is string => typeof id === "string" && id.length > 0,
      ),
    ),
  ];
}

export function parseHiddenHome(raw: string | null): HiddenHomeItems {
  if (!raw) return emptyHiddenHome();
  try {
    const parsed = JSON.parse(raw) as Partial<HiddenHomeItems>;
    return {
      albums: readIds(parsed.albums),
      mixes: readIds(parsed.mixes),
      playlists: readIds(parsed.playlists),
      artists: readIds(parsed.artists),
    };
  } catch {
    return emptyHiddenHome();
  }
}

export function loadHiddenHome(): HiddenHomeItems {
  try {
    return parseHiddenHome(localStorage.getItem(homeHiddenStorageKey()));
  } catch {
    return emptyHiddenHome();
  }
}

export function saveHiddenHome(items: HiddenHomeItems): void {
  try {
    localStorage.setItem(homeHiddenStorageKey(), JSON.stringify(items));
  } catch {
    // ignore quota / private mode
  }
}

export function hideHomeId(
  items: HiddenHomeItems,
  kind: HiddenHomeKind,
  id: string,
): HiddenHomeItems {
  const field = KIND_FIELD[kind];
  if (!id || items[field].includes(id)) return items;
  return { ...items, [field]: [...items[field], id] };
}

export function isHomeIdHidden(
  items: HiddenHomeItems,
  kind: HiddenHomeKind,
  id: string,
): boolean {
  return items[KIND_FIELD[kind]].includes(id);
}

export function filterHiddenById<T extends { id: string }>(
  items: readonly T[],
  hiddenIds: ReadonlySet<string>,
): T[] {
  if (hiddenIds.size === 0) return [...items];
  return items.filter((item) => !hiddenIds.has(item.id));
}
