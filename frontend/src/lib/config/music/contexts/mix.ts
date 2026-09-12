// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicMixContext } from "../mix-ops";
import type { MusicStoreHost } from "../types";

export function createMixContext(store: MusicStoreHost): MusicMixContext {
  return {
    get config() {
      return store.config;
    },
    get personalMixes() {
      return store.personalMixes;
    },
    set personalMixes(v) {
      store.personalMixes = v;
    },
    get mixSettings() {
      return store.mixSettings;
    },
    set mixSettings(v) {
      store.mixSettings = v;
    },
    get mixesRegenerating() {
      return store.mixesRegenerating;
    },
    set mixesRegenerating(v) {
      store.mixesRegenerating = v;
    },
    get mixesRefreshInFlight() {
      return store.mixesRefreshInFlight;
    },
    set mixesRefreshInFlight(v) {
      store.mixesRefreshInFlight = v;
    },
    get personalizationFetchedAt() {
      return store.personalizationFetchedAt;
    },
    set personalizationFetchedAt(v) {
      store.personalizationFetchedAt = v;
    },
    get connected() {
      return store.connected;
    },
    get stats() {
      return store.stats;
    },
    set stats(v) {
      store.stats = v;
    },
    get listenHistory() {
      return store.listenHistory;
    },
    set listenHistory(v) {
      store.listenHistory = v;
    },
    get frequentAlbums() {
      return store.frequentAlbums;
    },
    get recommendations() {
      return store.recommendations;
    },
    set recommendations(v) {
      store.recommendations = v;
    },
    get library() {
      return store.library;
    },
    get shuffle() {
      return store.shuffle;
    },
    set shuffle(v) {
      store.shuffle = v;
    },
    entryToSong: (entry) => store.entryToSong(entry),
    resolveTracksForRestore: (ids) => store.resolveTracksForRestore(ids),
    playTracks: (tracks, startIndex) => store.playTracks(tracks, startIndex),
    refreshMixes: () => store.mixOps.refreshMixes(),
    doRefreshMixes: (force) => store.mixOps.doRefreshMixes(force),
    refreshRecommendations: () => store.mixOps.refreshRecommendations(),
    hydrateMixTracks: (mix) => store.mixOps.hydrateMixTracks(mix),
    replaceMixWithHydrated: (hydrated) =>
      store.mixOps.replaceMixWithHydrated(hydrated),
  };
}
