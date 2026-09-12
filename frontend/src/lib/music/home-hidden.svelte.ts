// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { StorageKeys } from "$lib/brand";
import { getActiveInstanceId } from "$lib/features/instances/context";

export type HiddenHomeKind = "album" | "mix" | "playlist" | "artist";

export interface HiddenHomeItems {
  albums: string[];
  mixes: string[];
  playlists: string[];
  artists: string[];
}

export const HOME_HIDDEN_KEY = StorageKeys.homeHidden;

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

class HomeHiddenStore {
  albums = $state<Set<string>>(new Set());
  mixes = $state<Set<string>>(new Set());
  playlists = $state<Set<string>>(new Set());
  artists = $state<Set<string>>(new Set());

  constructor() {
    this.reload();
  }

  reload() {
    const items = loadHiddenHome();
    this.albums = new Set(items.albums);
    this.mixes = new Set(items.mixes);
    this.playlists = new Set(items.playlists);
    this.artists = new Set(items.artists);
  }

  hide(kind: HiddenHomeKind, id: string) {
    if (!id) return;
    const next = hideHomeId(
      {
        albums: [...this.albums],
        mixes: [...this.mixes],
        playlists: [...this.playlists],
        artists: [...this.artists],
      },
      kind,
      id,
    );
    this.albums = new Set(next.albums);
    this.mixes = new Set(next.mixes);
    this.playlists = new Set(next.playlists);
    this.artists = new Set(next.artists);
    saveHiddenHome(next);
  }

  isShortcutHidden(item: { kind: string; id: string }): boolean {
    if (item.kind === "album") {
      const id = item.id.startsWith("newest:")
        ? item.id.slice("newest:".length)
        : item.id.startsWith("album:")
          ? item.id.slice("album:".length)
          : item.id;
      return this.albums.has(id);
    }
    if (item.kind === "mix") {
      return this.mixes.has(item.id.replace(/^mix:/, ""));
    }
    if (item.kind === "playlist") {
      return this.playlists.has(item.id.replace(/^playlist:/, ""));
    }
    if (item.kind === "artist") {
      return this.artists.has(item.id.replace(/^artist:/, ""));
    }
    return false;
  }

  filterAlbums<T extends { id: string }>(items: readonly T[]): T[] {
    return filterHiddenById(items, this.albums);
  }

  filterMixes<T extends { id: string }>(items: readonly T[]): T[] {
    return filterHiddenById(items, this.mixes);
  }

  filterPlaylists<T extends { id: string }>(items: readonly T[]): T[] {
    return filterHiddenById(items, this.playlists);
  }

  filterArtists<T extends { id: string }>(items: readonly T[]): T[] {
    return filterHiddenById(items, this.artists);
  }

  clearAll() {
    const empty = emptyHiddenHome();
    this.albums = new Set();
    this.mixes = new Set();
    this.playlists = new Set();
    this.artists = new Set();
    saveHiddenHome(empty);
  }
}

export const homeHidden = new HomeHiddenStore();
