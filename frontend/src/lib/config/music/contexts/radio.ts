// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicRadioContext } from "../radio-ops";
import type { MusicStoreHost } from "../types";

export function createRadioContext(store: MusicStoreHost): MusicRadioContext {
  return {
    get continuousBusy() {
      return store.continuousBusy;
    },
    set continuousBusy(v) {
      store.continuousBusy = v;
    },
    get continuousMode() {
      return store.continuousMode;
    },
    set continuousMode(v) {
      store.continuousMode = v;
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
    get library() {
      return store.library;
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
    get listenHistory() {
      return store.listenHistory;
    },
    get stats() {
      return store.stats;
    },
    get mixSettings() {
      return store.mixSettings;
    },
    get internetRadios() {
      return store.internetRadios;
    },
    playTracks: (tracks, startIndex, resume, continuousMode, options) =>
      store.playTracks(tracks, startIndex, resume, continuousMode, options),
    startRandomRadio: (count) => store.radioOps.startRandomRadio(count),
    playInternetRadio: (station) => store.radioOps.playInternetRadio(station),
    refreshInternetRadios: () => store.refreshInternetRadios(),
    persistPlaybackState: () => store.persistPlaybackState(),
    entryToSong: (entry) => store.entryToSong(entry),
    personalRadioOptions: (coldStart) => store.personalRadioOptions(coldStart),
  };
}
