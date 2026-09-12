// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicLibraryBrowseContext } from "../library-browse-ops";
import type { MusicStoreHost } from "../types";

export function createLibraryBrowseContext(
  store: MusicStoreHost,
): MusicLibraryBrowseContext {
  return {
    get library() {
      return store.library;
    },
    get recentAlbums() {
      return store.recentAlbums;
    },
    set recentAlbums(v) {
      store.recentAlbums = v;
    },
    get frequentAlbums() {
      return store.frequentAlbums;
    },
    set frequentAlbums(v) {
      store.frequentAlbums = v;
    },
    get randomTracks() {
      return store.randomTracks;
    },
    set randomTracks(v) {
      store.randomTracks = v;
    },
    get resumeTracks() {
      return store.resumeTracks;
    },
    set resumeTracks(v) {
      store.resumeTracks = v;
    },
    get homeCoreFetchedAt() {
      return store.homeCoreFetchedAt;
    },
    set homeCoreFetchedAt(v) {
      store.homeCoreFetchedAt = v;
    },
    get listenHistory() {
      return store.listenHistory;
    },
    set listenHistory(v) {
      store.listenHistory = v;
    },
    get stats() {
      return store.stats;
    },
    set stats(v) {
      store.stats = v;
    },
    get libraryStats() {
      return store.libraryStats;
    },
    set libraryStats(v) {
      store.libraryStats = v;
    },
    get genres() {
      return store.genres;
    },
    set genres(v) {
      store.genres = v;
    },
    get allArtists() {
      return store.allArtists;
    },
    set allArtists(v) {
      store.allArtists = v;
    },
    get artistsLoaded() {
      return store.artistsLoaded;
    },
    set artistsLoaded(v) {
      store.artistsLoaded = v;
    },
    get artistsLoadedAt() {
      return store.artistsLoadedAt;
    },
    set artistsLoadedAt(v) {
      store.artistsLoadedAt = v;
    },
    get shuffle() {
      return store.shuffle;
    },
    set shuffle(v) {
      store.shuffle = v;
    },
    playTracks: (tracks, startIndex) => store.playTracks(tracks, startIndex),
    addTracksToQueue: (tracks) => store.addTracksToQueue(tracks),
    playTracksNext: (tracks) => store.playTracksNext(tracks),
  };
}
