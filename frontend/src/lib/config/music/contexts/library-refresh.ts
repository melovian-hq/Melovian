// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicLibraryRefreshContext } from "../library-refresh-ops";
import type { MusicStoreHost } from "../types";

export function createLibraryRefreshContext(
  store: MusicStoreHost,
): MusicLibraryRefreshContext {
  return {
    get libraryRefreshing() {
      return store.libraryRefreshing;
    },
    set libraryRefreshing(v) {
      store.libraryRefreshing = v;
    },
    get connected() {
      return store.connected;
    },
    get homeCoreFetchedAt() {
      return store.homeCoreFetchedAt;
    },
    set homeCoreFetchedAt(v) {
      store.homeCoreFetchedAt = v;
    },
    get personalizationFetchedAt() {
      return store.personalizationFetchedAt;
    },
    set personalizationFetchedAt(v) {
      store.personalizationFetchedAt = v;
    },
    get libraryStats() {
      return store.libraryStats;
    },
    set libraryStats(v) {
      store.libraryStats = v;
    },
    get libraryRevision() {
      return store.libraryRevision;
    },
    set libraryRevision(v) {
      store.libraryRevision = v;
    },
    get lastLibrarySongCount() {
      return store.lastLibrarySongCount;
    },
    set lastLibrarySongCount(v) {
      store.lastLibrarySongCount = v;
    },
    get lastLibraryScanning() {
      return store.lastLibraryScanning;
    },
    set lastLibraryScanning(v) {
      store.lastLibraryScanning = v;
    },
    get libraryWatchTimer() {
      return store.libraryWatchTimer;
    },
    set libraryWatchTimer(v) {
      store.libraryWatchTimer = v;
    },
    invalidateArtistIndex: () => store.libraryBrowseOps.invalidateArtistIndex(),
    refreshHomeCore: () => store.refreshHomeCore(),
    refreshLibraryStats: (options) => store.refreshLibraryStats(options),
    refreshFavorites: () => store.refreshFavorites(),
    refreshServerPlaylists: () => store.refreshServerPlaylists(),
    loadArtists: (options) => store.loadArtists(options),
    schedulePersonalizationRefresh: (force) =>
      store.schedulePersonalizationRefresh(force),
    refreshLibrary: (options) => store.refreshLibrary(options),
    stopLibraryWatch: () => store.stopLibraryWatch(),
    scheduleLibraryWatch: (delayMs) => store.scheduleLibraryWatch(delayMs),
    pollLibraryChanges: () => store.pollLibraryChanges(),
  };
}
