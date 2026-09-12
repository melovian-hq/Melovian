// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicTrackBoundaryContext } from "../track-boundary-ops";
import type { MusicStoreHost } from "../types";

export function createTrackBoundaryContext(
  store: MusicStoreHost,
): MusicTrackBoundaryContext {
  return {
    get trackBoundaryBusy() {
      return store.trackBoundaryBusy;
    },
    set trackBoundaryBusy(v) {
      store.trackBoundaryBusy = v;
    },
    get crossfadeHandled() {
      return store.crossfadeHandled;
    },
    set crossfadeHandled(v) {
      store.crossfadeHandled = v;
    },
    get currentTrack() {
      return store.currentTrack;
    },
    get playing() {
      return store.playing;
    },
    set playing(v) {
      store.playing = v;
    },
    get engine() {
      return store.engine;
    },
    get currentTime() {
      return store.currentTime;
    },
    set currentTime(v) {
      store.currentTime = v;
    },
    get playbackEpoch() {
      return store.playbackEpoch;
    },
    get smoothProgress() {
      return store.smoothProgress;
    },
    set smoothProgress(v) {
      store.smoothProgress = v;
    },
    get lastSavedPosition() {
      return store.lastSavedPosition;
    },
    set lastSavedPosition(v) {
      store.lastSavedPosition = v;
    },
    get playbackRestored() {
      return store.playbackRestored;
    },
    set playbackRestored(v) {
      store.playbackRestored = v;
    },
    get queue() {
      return store.queue;
    },
    set queue(v) {
      store.queue = v;
    },
    get queueIndex() {
      return store.queueIndex;
    },
    set queueIndex(v) {
      store.queueIndex = v;
    },
    get shuffle() {
      return store.shuffle;
    },
    set shuffle(v) {
      store.shuffle = v;
    },
    get autoplay() {
      return store.autoplay;
    },
    set autoplay(v) {
      store.autoplay = v;
    },
    get continuousMode() {
      return store.continuousMode;
    },
    set continuousMode(v) {
      store.continuousMode = v;
    },
    get repeat() {
      return store.repeat;
    },
    get shuffleUpcoming() {
      return store.shuffleUpcoming;
    },
    set shuffleUpcoming(v) {
      store.shuffleUpcoming = v;
    },
    get shuffleHistory() {
      return store.shuffleHistory;
    },
    set shuffleHistory(v) {
      store.shuffleHistory = v;
    },
    get queueSettings() {
      return store.queueSettings;
    },
    get continuousRefillInFlight() {
      return store.continuousRefillInFlight;
    },
    set continuousRefillInFlight(v) {
      store.continuousRefillInFlight = v;
    },
    get libraryPool() {
      return store.libraryPool;
    },
    get personalRadio() {
      return store.personalRadio;
    },
    get listenHistory() {
      return store.listenHistory;
    },
    get stats() {
      return store.stats;
    },
    get library() {
      return store.library;
    },
    get nativePlayback() {
      return store.nativePlayback;
    },
    get transcodedTrackIds() {
      return store.transcodedTrackIds;
    },
    get failedTrackSkips() {
      return store.failedTrackSkips;
    },
    set failedTrackSkips(v) {
      store.failedTrackSkips = v;
    },
    get error() {
      return store.error;
    },
    set error(v) {
      store.error = v;
    },
    get playerLayout() {
      return store.playerLayout;
    },
    set playerLayout(v) {
      store.playerLayout = v;
    },
    get internetRadios() {
      return store.internetRadios;
    },
    isDownloaded: (trackId) => store.isDownloaded(trackId),
    trackStreamUrl: (track) => store.trackStreamUrl(track),
    loadTrackSource: (token, track, url) =>
      store.loadTrackSource(token, track, url),
    startProgressTracking: () => store.startProgressTracking(),
    startSmoothProgress: () => store.startSmoothProgress(),
    stopSmoothProgress: () => store.stopSmoothProgress(),
    syncMediaSession: () => store.syncMediaSession(),
    suspendForReconnect: () => store.suspendForReconnect(),
    isSupersededPlaybackError: (message) =>
      store.isSupersededPlaybackError(message),
    markTrackTranscoded: (trackId) => store.markTrackTranscoded(trackId),
    maybeRefillContinuousQueue: () => store.maybeRefillContinuousQueue(),
    refillLibraryQueue: (count) => store.refillLibraryQueue(count),
    refillPersonalQueue: (count) => store.refillPersonalQueue(count),
    appendRandomSongsToQueue: (count) => store.appendRandomSongsToQueue(count),
    appendTracksToQueue: (tracks) => store.appendTracksToQueue(tracks),
    advanceTrack: () => store.advanceTrack(),
    stopAtQueueEnd: () => store.stopAtQueueEnd(),
    recordPlayCompletion: (track) => store.recordPlayCompletion(track),
    personalRadioOptions: (coldStart) => store.personalRadioOptions(coldStart),
    entryToSong: (entry) => store.entryToSong(entry),
    persistPlaybackState: () => store.persistPlaybackState(),
    seedShuffleUpcoming: () => store.seedShuffleUpcoming(),
    initEngine: () => store.initEngine(),
    resolveTracksForRestore: (trackIds) =>
      store.resolveTracksForRestore(trackIds),
    refreshInternetRadios: () => store.refreshInternetRadios(),
  };
}
