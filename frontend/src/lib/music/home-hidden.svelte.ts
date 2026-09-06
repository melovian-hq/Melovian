// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  emptyHiddenHome,
  filterHiddenById,
  hideHomeId,
  loadHiddenHome,
  saveHiddenHome,
  type HiddenHomeKind,
} from "./home-hidden";

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
