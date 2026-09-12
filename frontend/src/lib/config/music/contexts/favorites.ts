// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicFavoritesContext } from "../favorites-ops";
import type { MusicStoreHost } from "../types";

export function createFavoritesContext(
  store: MusicStoreHost,
): MusicFavoritesContext {
  return {
    get connected() {
      return store.connected;
    },
    get library() {
      return store.library;
    },
    get favoriteTracks() {
      return store.favoriteTracks;
    },
    set favoriteTracks(v) {
      store.favoriteTracks = v;
    },
    get favoriteIds() {
      return store.favoriteIds;
    },
    set favoriteIds(v) {
      store.favoriteIds = v;
    },
    get favoriteAlbums() {
      return store.favoriteAlbums;
    },
    set favoriteAlbums(v) {
      store.favoriteAlbums = v;
    },
    get favoriteAlbumIds() {
      return store.favoriteAlbumIds;
    },
    set favoriteAlbumIds(v) {
      store.favoriteAlbumIds = v;
    },
    get favoriteArtists() {
      return store.favoriteArtists;
    },
    set favoriteArtists(v) {
      store.favoriteArtists = v;
    },
    get favoriteArtistIds() {
      return store.favoriteArtistIds;
    },
    set favoriteArtistIds(v) {
      store.favoriteArtistIds = v;
    },
    get shuffle() {
      return store.shuffle;
    },
    set shuffle(v) {
      store.shuffle = v;
    },
    isFavorite: (trackId) => store.isFavorite(trackId),
    isFavoriteAlbum: (albumId) => store.isFavoriteAlbum(albumId),
    isFavoriteArtist: (artistId) => store.isFavoriteArtist(artistId),
    playArtistAlbums: (albums, shuffle) =>
      store.libraryBrowseOps.playArtistAlbums(albums, shuffle),
    playTracks: (tracks, startIndex) => store.playTracks(tracks, startIndex),
  };
}
