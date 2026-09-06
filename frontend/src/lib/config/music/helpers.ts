// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import type { MixFetchers } from "$lib/music/mix-generator";
import type { SubsonicSong } from "$lib/subsonic";
import type { FavoriteTrack, ListenEntry } from "$lib/subsonic/types";

export function entryToSong(entry: ListenEntry): SubsonicSong {
  return {
    id: entry.trackId,
    title: entry.trackTitle,
    artist: entry.artistName,
    album: entry.albumTitle,
    albumId: entry.albumId,
    coverArt: entry.coverArtId,
    duration: Math.floor(entry.durationMs / 1000),
  };
}

export function favoriteToSong(entry: FavoriteTrack): SubsonicSong {
  return {
    id: entry.trackId,
    title: entry.trackTitle,
    artist: entry.artistName,
    album: entry.albumTitle,
    albumId: entry.albumId,
    coverArt: entry.coverArtId,
    duration: Math.floor(entry.durationMs / 1000),
  };
}

export function shuffleTracks<T>(items: T[]): T[] {
  const copy = [...items];
  for (let i = copy.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [copy[i], copy[j]] = [copy[j], copy[i]];
  }
  return copy;
}

export function createMixFetchers(library: MusicLibraryAdapter): MixFetchers {
  return {
    searchArtistSongs: async (artist, limit) => {
      const result = await library.search3(artist, limit).catch(() => ({
        artists: [],
        albums: [],
        songs: [],
      }));
      return result.songs.filter(
        (song) => song.artist?.toLowerCase() === artist.toLowerCase(),
      );
    },
    searchArtistAlbums: (artist, limit) =>
      library.searchArtistAlbums(artist, limit).catch(() => []),
    getAlbumSongs: async (albumId) => {
      const album = await library.getAlbum(albumId).catch(() => null);
      return album?.songs ?? [];
    },
    getSimilarSongs: (trackId, count) =>
      library.getSimilarSongs(trackId, count).catch(() => []),
    getRandomSongs: (count) => library.getRandomSongs(count).catch(() => []),
    getRandomAlbums: (count) =>
      library.getAlbumList2("random", count).catch(() => []),
    getNewestAlbums: (count) =>
      library.getAlbumList2("newest", count).catch(() => []),
    getGenreSongs: (genre, count) =>
      library.getSongsByGenre(genre, count).catch(() => []),
    getGenres: () => library.getGenres().catch(() => []),
    getStarredSongs: async () => {
      const starred = await library.getStarred2().catch(() => ({
        songs: [] as SubsonicSong[],
        albums: [],
        artists: [],
      }));
      return starred.songs ?? [];
    },
  };
}
