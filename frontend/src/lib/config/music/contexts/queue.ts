// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicQueueContext } from "../queue-ops";
import type { MusicStoreHost } from "../types";

export function createQueueContext(store: MusicStoreHost): MusicQueueContext {
  return {
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
    set queueSettings(v) {
      store.queueSettings = v;
    },
    get engine() {
      return store.engine;
    },
    get playing() {
      return store.playing;
    },
    set playing(v) {
      store.playing = v;
    },
    get continuousMode() {
      return store.continuousMode;
    },
    set continuousMode(v) {
      store.continuousMode = v;
    },
    get queueOpen() {
      return store.queueOpen;
    },
    set queueOpen(v) {
      store.queueOpen = v;
    },
    get currentTrack() {
      return store.currentTrack;
    },
    get playerLayout() {
      return store.playerLayout;
    },
    set playerLayout(v) {
      store.playerLayout = v;
    },
    get pendingStartAt() {
      return store.pendingStartAt;
    },
    set pendingStartAt(v) {
      store.pendingStartAt = v;
    },
    get pendingStartPaused() {
      return store.pendingStartPaused;
    },
    set pendingStartPaused(v) {
      store.pendingStartPaused = v;
    },
    playTracks: (tracks, startIndex) => store.playTracks(tracks, startIndex),
    prefetchAround: () => store.prefetchAround(),
    persistPlaybackState: () => store.persistPlaybackState(),
    requestPlayCurrent: () => store.requestPlayCurrent(),
    seedShuffleUpcoming: () => store.seedShuffleUpcoming(),
    stopSmoothProgress: () => store.stopSmoothProgress(),
  };
}
