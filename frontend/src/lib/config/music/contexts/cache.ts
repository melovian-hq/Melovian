// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicCacheContext } from "../cache-ops";
import type { MusicStoreHost } from "../types";

export function createCacheContext(store: MusicStoreHost): MusicCacheContext {
  return {
    get downloadedIds() {
      return store.downloadedIds;
    },
    set downloadedIds(v) {
      store.downloadedIds = v;
    },
    get cacheSettings() {
      return store.cacheSettings;
    },
    set cacheSettings(v) {
      store.cacheSettings = v;
    },
    get cacheUsedBytes() {
      return store.cacheUsedBytes;
    },
    set cacheUsedBytes(v) {
      store.cacheUsedBytes = v;
    },
    get cacheTrackCount() {
      return store.cacheTrackCount;
    },
    set cacheTrackCount(v) {
      store.cacheTrackCount = v;
    },
    get cacheInFlight() {
      return store.cacheInFlight;
    },
    get cacheStrategyRunning() {
      return store.cacheStrategyRunning;
    },
    set cacheStrategyRunning(v) {
      store.cacheStrategyRunning = v;
    },
    get connected() {
      return store.connected;
    },
    get shuffle() {
      return store.shuffle;
    },
    get repeat() {
      return store.repeat;
    },
    get queue() {
      return store.queue;
    },
    get listenHistory() {
      return store.listenHistory;
    },
    get frequentAlbums() {
      return store.frequentAlbums;
    },
    get playlists() {
      return store.playlists;
    },
    get library() {
      return store.library;
    },
    get offlineDownloadProgress() {
      return store.offlineDownloadProgress;
    },
    set offlineDownloadProgress(v) {
      store.offlineDownloadProgress = v;
    },
    get offlineDownloadAbort() {
      return store.offlineDownloadAbort;
    },
    set offlineDownloadAbort(v) {
      store.offlineDownloadAbort = v;
    },
    sequentialNextIndex: () => store.sequentialNextIndex(),
    runCacheStrategy: () => store.cacheOps.runCacheStrategy(),
    downloadCurrentOr: (track) => store.cacheOps.downloadCurrentOr(track),
    loadCacheSettings: () => store.cacheOps.loadCacheSettings(),
    collectStrategyTracks: () => store.cacheOps.collectStrategyTracks(),
    prefetchCache: (track) => store.cacheOps.prefetchCache(track),
    removeDownload: (trackId) => store.cacheOps.removeDownload(trackId),
    cancelOfflineDownload: () => store.cacheOps.cancelOfflineDownload(),
  };
}
