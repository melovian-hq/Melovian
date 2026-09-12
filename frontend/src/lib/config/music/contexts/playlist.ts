// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { sources } from "$lib/features/sources/store.svelte";
import type { MusicPlaylistContext } from "../playlist-ops";
import type { MusicStoreHost } from "../types";

export function createPlaylistContext(
  store: MusicStoreHost,
): MusicPlaylistContext {
  return {
    get connected() {
      return store.connected;
    },
    get hasSubsonicActive() {
      return sources.hasSubsonicActive;
    },
    get client() {
      return store.client;
    },
    get library() {
      return store.library;
    },
    get playlists() {
      return store.playlists;
    },
    set playlists(v) {
      store.playlists = v;
    },
    get serverPlaylists() {
      return store.serverPlaylists;
    },
    set serverPlaylists(v) {
      store.serverPlaylists = v;
    },
    get internetRadios() {
      return store.internetRadios;
    },
    set internetRadios(v) {
      store.internetRadios = v;
    },
  };
}
