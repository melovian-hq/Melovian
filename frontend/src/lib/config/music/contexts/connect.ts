// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicConnectContext } from "../connect-ops";
import type { MusicStoreHost } from "../types";

export function createConnectContext(
  store: MusicStoreHost,
  homeStaleMs: number,
): MusicConnectContext {
  return {
    get connectInFlight() {
      return store.connectInFlight;
    },
    set connectInFlight(v) {
      store.connectInFlight = v;
    },
    get connected() {
      return store.connected;
    },
    set connected(v) {
      store.connected = v;
    },
    get loading() {
      return store.loading;
    },
    set loading(v) {
      store.loading = v;
    },
    get error() {
      return store.error;
    },
    set error(v) {
      store.error = v;
    },
    get status() {
      return store.status;
    },
    set status(v) {
      store.status = v;
    },
    get libraryWarmup() {
      return store.libraryWarmup;
    },
    set libraryWarmup(v) {
      store.libraryWarmup = v;
    },
    get playbackRestored() {
      return store.playbackRestored;
    },
    set playbackRestored(v) {
      store.playbackRestored = v;
    },
    get playing() {
      return store.playing;
    },
    set playing(v) {
      store.playing = v;
    },
    get queue() {
      return store.queue;
    },
    get queueIndex() {
      return store.queueIndex;
    },
    get reconnectResumePending() {
      return store.reconnectResumePending;
    },
    set reconnectResumePending(v) {
      store.reconnectResumePending = v;
    },
    get reconnectPositionMs() {
      return store.reconnectPositionMs;
    },
    set reconnectPositionMs(v) {
      store.reconnectPositionMs = v;
    },
    get currentTime() {
      return store.currentTime;
    },
    get currentTrack() {
      return store.currentTrack;
    },
    get engine() {
      return store.engine;
    },
    get playbackEpoch() {
      return store.playbackEpoch;
    },
    get homeCoreFetchedAt() {
      return store.homeCoreFetchedAt;
    },
    get homeStaleMs() {
      return homeStaleMs;
    },
    restoreCachedMixes: () => store.restoreCachedMixes(),
    initEngine: () => store.initEngine(),
    loadEqSettings: (authEnabled) => store.loadEqSettings(authEnabled),
    refreshHomeCore: () => store.refreshHomeCore(),
    loadCacheSettings: () => store.loadCacheSettings(),
    loadLyricsSettings: () => store.loadLyricsSettings(),
    restorePlayback: () => store.restorePlayback(),
    startLibraryWatch: () => store.startLibraryWatch(),
    stopLibraryWatch: () => store.stopLibraryWatch(),
    refreshHistory: (limit) => store.refreshHistory(limit),
    refreshStats: (limit) => store.refreshStats(limit),
    refreshPlaylists: () => store.refreshPlaylists(),
    refreshServerPlaylists: () => store.refreshServerPlaylists(),
    refreshInternetRadios: () => store.refreshInternetRadios(),
    refreshFavorites: () => store.refreshFavorites(),
    refreshLibraryStats: () => store.refreshLibraryStats(),
    loadDownloads: () => store.loadDownloads(),
    schedulePersonalizationRefresh: (force) =>
      store.schedulePersonalizationRefresh(force),
    runCacheStrategy: () => store.runCacheStrategy(),
    isDownloaded: (trackId) => store.isDownloaded(trackId),
    loadTrackSource: (token, track, url) =>
      store.loadTrackSource(token, track, url),
    trackStreamUrl: (track) => store.trackStreamUrl(track),
    startProgressTracking: () => store.startProgressTracking(),
    startSmoothProgress: () => store.startSmoothProgress(),
    stopSmoothProgress: () => store.stopSmoothProgress(),
    syncMediaSession: () => store.syncMediaSession(),
    handlePlaybackNetworkFailure: (token, options) =>
      store.handlePlaybackNetworkFailure(token, options),
    playCurrent: () => store.playCurrent(),
    seek: (seconds) => store.seek(seconds),
  };
}
