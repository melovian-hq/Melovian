// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicPlayLaunchContext } from "../play-launch-ops";
import type { MusicStoreHost } from "../types";

export function createPlayLaunchContext(
  store: MusicStoreHost,
): MusicPlayLaunchContext {
  return {
    get continuousMode() {
      return store.continuousMode;
    },
    set continuousMode(v) {
      store.continuousMode = v;
    },
    get libraryPool() {
      return store.libraryPool;
    },
    set libraryPool(v) {
      store.libraryPool = v;
    },
    get personalRadio() {
      return store.personalRadio;
    },
    set personalRadio(v) {
      store.personalRadio = v;
    },
    get queueSettings() {
      return store.queueSettings;
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
    get shuffleHistory() {
      return store.shuffleHistory;
    },
    set shuffleHistory(v) {
      store.shuffleHistory = v;
    },
    get shuffleUpcoming() {
      return store.shuffleUpcoming;
    },
    set shuffleUpcoming(v) {
      store.shuffleUpcoming = v;
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
    get repeat() {
      return store.repeat;
    },
    set repeat(v) {
      store.repeat = v;
    },
    get playerLayout() {
      return store.playerLayout;
    },
    set playerLayout(v) {
      store.playerLayout = v;
    },
    get resumeOnPlay() {
      return store.resumeOnPlay;
    },
    set resumeOnPlay(v) {
      store.resumeOnPlay = v;
    },
    get connected() {
      return store.connected;
    },
    get library() {
      return store.library;
    },
    seedShuffleUpcoming: () => store.seedShuffleUpcoming(),
    persistPlaybackState: () => store.persistPlaybackState(),
    requestPlayCurrent: () => store.requestPlayCurrent(),
    resolveTracksForRestore: (trackIds) =>
      store.resolveTracksForRestore(trackIds),
    playTracks: (tracks, startIndex, resume, continuousMode, options) =>
      store.playLaunchOps.playTracks(
        tracks,
        startIndex,
        resume,
        continuousMode,
        options,
      ),
    bootstrapOfflinePlayback: () => store.bootstrapOfflinePlayback(),
    isDownloaded: (trackId) => store.isDownloaded(trackId),
    resolveOfflineTrack: (trackId) => store.resolveOfflineTrack(trackId),
    connect: (options) => store.connect(options),
    playTrackById: (trackId) => store.playLaunchOps.playTrackById(trackId),
  };
}
