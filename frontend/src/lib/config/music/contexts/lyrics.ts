// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicLyricsContext } from "../lyrics-ops";
import type { MusicStoreHost } from "../types";

export function createLyricsContext(store: MusicStoreHost): MusicLyricsContext {
  return {
    get lyricsOpen() {
      return store.lyricsOpen;
    },
    set lyricsOpen(v) {
      store.lyricsOpen = v;
    },
    get currentTrack() {
      return store.currentTrack;
    },
    get currentLyrics() {
      return store.currentLyrics;
    },
    set currentLyrics(v) {
      store.currentLyrics = v;
    },
    get lyricsLoading() {
      return store.lyricsLoading;
    },
    set lyricsLoading(v) {
      store.lyricsLoading = v;
    },
    get lyricsFetching() {
      return store.lyricsFetching;
    },
    set lyricsFetching(v) {
      store.lyricsFetching = v;
    },
    get lyricsRequestId() {
      return store.lyricsRequestId;
    },
    set lyricsRequestId(v) {
      store.lyricsRequestId = v;
    },
    get pendingLyricsReload() {
      return store.pendingLyricsReload;
    },
    set pendingLyricsReload(v) {
      store.pendingLyricsReload = v;
    },
    get lyricsSettings() {
      return store.lyricsSettings;
    },
    set lyricsSettings(v) {
      store.lyricsSettings = v;
    },
    get favoriteTracks() {
      return store.favoriteTracks;
    },
    get listenHistory() {
      return store.listenHistory;
    },
    get resumeTracks() {
      return store.resumeTracks;
    },
    get randomTracks() {
      return store.randomTracks;
    },
    get library() {
      return store.library;
    },
    get config() {
      return store.config;
    },
    seek: (seconds) => store.seek(seconds),
    favoriteToSong: (entry) => store.favoriteToSong(entry),
    entryToSong: (entry) => store.entryToSong(entry),
  };
}
