// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicPlaybackTransportContext } from "../playback-transport-ops";
import type { MusicStoreHost } from "../types";

export function createPlaybackTransportContext(
  store: MusicStoreHost,
): MusicPlaybackTransportContext {
  return {
    get engine() {
      return store.engine;
    },
    get playing() {
      return store.playing;
    },
    set playing(v) {
      store.playing = v;
    },
    get volume() {
      return store.volume;
    },
    set volume(v) {
      store.volume = v;
    },
    get repeat() {
      return store.repeat;
    },
    set repeat(v) {
      store.repeat = v;
    },
    get shuffle() {
      return store.shuffle;
    },
    set shuffle(v) {
      store.shuffle = v;
    },
    get queue() {
      return store.queue;
    },
    get queueIndex() {
      return store.queueIndex;
    },
    set queueIndex(v) {
      store.queueIndex = v;
    },
    get currentTrack() {
      return store.currentTrack;
    },
    get currentTime() {
      return store.currentTime;
    },
    set currentTime(v) {
      store.currentTime = v;
    },
    get duration() {
      return store.duration;
    },
    set duration(v) {
      store.duration = v;
    },
    get smoothProgress() {
      return store.smoothProgress;
    },
    set smoothProgress(v) {
      store.smoothProgress = v;
    },
    get continuousMode() {
      return store.continuousMode;
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
    get personalRadio() {
      return store.personalRadio;
    },
    get lastSavedPosition() {
      return store.lastSavedPosition;
    },
    set lastSavedPosition(v) {
      store.lastSavedPosition = v;
    },
    stopSmoothProgress: () => store.stopSmoothProgress(),
    syncMediaSession: () => store.syncMediaSession(),
    initEngine: () => store.initEngine(),
    playCurrent: () => store.playCurrent(),
    requestPlayCurrent: () => store.requestPlayCurrent(),
    advanceShuffleIndex: () => store.advanceShuffleIndex(),
    stopAtQueueEnd: () => store.stopAtQueueEnd(),
    maybeRefillContinuousQueue: () => store.maybeRefillContinuousQueue(),
    persistPlaybackState: () => store.persistPlaybackState(),
    saveProgress: (positionMs, options) =>
      store.saveProgress(positionMs, options),
    prefetchAround: () => store.prefetchAround(),
    startSmoothProgress: () => store.startSmoothProgress(),
  };
}
