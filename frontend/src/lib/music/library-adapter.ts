// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { SubsonicClient } from "$lib/subsonic/client";
import type {
  SubsonicAlbum,
  SubsonicArtist,
  SubsonicArtistInfo,
  SubsonicGenre,
  SubsonicSearchResult,
  SubsonicSong,
  InternetRadioStation,
  ServerPlaylist,
  StarredContent,
} from "$lib/subsonic/types";
import type { LyricsSearchHit, ParsedLyrics } from "$lib/music/lyrics";
import * as subsonicApi from "$lib/subsonic/api";
import * as localApi from "$lib/local-music/api";
import { dedupeAlbums } from "$lib/music/album-dedup";

const LOCAL_ID_PREFIXES = ["trk_", "alb_", "art_"] as const;

export function isLocalMusicId(id: string): boolean {
  return LOCAL_ID_PREFIXES.some((prefix) => id.startsWith(prefix));
}

function mergeUniqueById<T extends { id: string }>(a: T[], b: T[]): T[] {
  const seen = new Set<string>();
  const out: T[] = [];
  for (const item of [...a, ...b]) {
    if (!item.id || seen.has(item.id)) continue;
    seen.add(item.id);
    out.push(item);
  }
  return out;
}

function mergeUniqueGenres(
  a: SubsonicGenre[],
  b: SubsonicGenre[],
): SubsonicGenre[] {
  const seen = new Set<string>();
  const out: SubsonicGenre[] = [];
  for (const item of [...a, ...b]) {
    const key = item.name.toLowerCase();
    if (!item.name || seen.has(key)) continue;
    seen.add(key);
    out.push(item);
  }
  return out;
}

function mergeLyricsHits(
  a: LyricsSearchHit[],
  b: LyricsSearchHit[],
): LyricsSearchHit[] {
  const seen = new Set<string>();
  const out: LyricsSearchHit[] = [];
  for (const item of [...a, ...b]) {
    if (!item.songId || seen.has(item.songId)) continue;
    seen.add(item.songId);
    out.push(item);
  }
  return out;
}

function mergeSearchResults(
  a: SubsonicSearchResult,
  b: SubsonicSearchResult,
): SubsonicSearchResult {
  return {
    artists: mergeUniqueById(a.artists, b.artists),
    albums: dedupeAlbums(mergeUniqueById(a.albums, b.albums)),
    songs: mergeUniqueById(a.songs, b.songs),
  };
}

async function settle<T>(promise: Promise<T>, fallback: T): Promise<T> {
  try {
    return await promise;
  } catch {
    return fallback;
  }
}

const emptySearchResult: SubsonicSearchResult = {
  artists: [],
  albums: [],
  songs: [],
};

async function searchBothLibraries(
  subsonic: MusicLibraryAdapter,
  local: MusicLibraryAdapter,
  query: string,
  limit?: number,
): Promise<SubsonicSearchResult> {
  const [subResult, localResult] = await Promise.all([
    settle(subsonic.search3(query, limit), emptySearchResult),
    settle(local.search3(query, limit), emptySearchResult),
  ]);
  return mergeSearchResults(subResult, localResult);
}

export interface MusicLibraryAdapter {
  getArtists(): Promise<SubsonicArtist[]>;
  getArtist(
    id: string,
  ): Promise<{ artist: SubsonicArtist; albums: SubsonicAlbum[] }>;
  getAlbum(
    id: string,
  ): Promise<{ album: SubsonicAlbum; songs: SubsonicSong[] }>;
  getAlbumList2(
    type:
      | "newest"
      | "recent"
      | "frequent"
      | "random"
      | "alphabeticalByName"
      | "byGenre",
    size?: number,
    offset?: number,
  ): Promise<SubsonicAlbum[]>;
  search3(query: string, limit?: number): Promise<SubsonicSearchResult>;
  getSong(id: string): Promise<SubsonicSong | null>;
  getRandomSongs(size?: number): Promise<SubsonicSong[]>;
  getGenres(): Promise<SubsonicGenre[]>;
  getSongsByGenre(
    genre: string,
    count?: number,
    offset?: number,
  ): Promise<SubsonicSong[]>;
  getServerPlaylists(): Promise<ServerPlaylist[]>;
  getServerPlaylist(
    id: string,
  ): Promise<{ playlist: ServerPlaylist; songs: SubsonicSong[] } | null>;
  getInternetRadioStations(): Promise<InternetRadioStation[]>;
  getStarred2(): Promise<StarredContent>;
  star(id: string): Promise<void>;
  unstar(id: string): Promise<void>;
  scrobble(id: string, submission: boolean): Promise<void>;
  getSimilarSongs(trackId: string, count?: number): Promise<SubsonicSong[]>;
  searchArtistAlbums(artist: string, limit?: number): Promise<SubsonicAlbum[]>;
  getArtistInfo(id: string): Promise<SubsonicArtistInfo>;
  getLyricsForSong(song: {
    id: string;
    artist?: string;
    title: string;
  }): Promise<ParsedLyrics | null>;
  searchLyricsByText(
    query: string,
    options?: {
      maxResults?: number;
      scanLimit?: number;
      extraSongs?: SubsonicSong[];
    },
  ): Promise<LyricsSearchHit[]>;
}

export function createSubsonicLibraryAdapter(
  client: SubsonicClient,
): MusicLibraryAdapter {
  return {
    getArtists: () => subsonicApi.getArtists(client),
    getArtist: async (id) => {
      const result = await subsonicApi.getArtist(client, id);
      return { ...result, albums: dedupeAlbums(result.albums) };
    },
    getAlbum: (id) => subsonicApi.getAlbum(client, id),
    getAlbumList2: async (type, size, offset) =>
      dedupeAlbums(await subsonicApi.getAlbumList2(client, type, size, offset)),
    search3: (query, limit) => subsonicApi.search3(client, query, limit),
    getSong: (id) => subsonicApi.getSong(client, id),
    getRandomSongs: (size) => subsonicApi.getRandomSongs(client, size),
    getGenres: () => subsonicApi.getGenres(client),
    getSongsByGenre: (genre, count, offset) =>
      subsonicApi.getSongsByGenre(client, genre, count, offset),
    getServerPlaylists: () => subsonicApi.getServerPlaylists(client),
    getServerPlaylist: (id) => subsonicApi.getServerPlaylist(client, id),
    getInternetRadioStations: () =>
      subsonicApi.getInternetRadioStations(client),
    getStarred2: () => subsonicApi.getStarred2(client),
    star: (id) => subsonicApi.star(client, id),
    unstar: (id) => subsonicApi.unstar(client, id),
    scrobble: (id, submission) => subsonicApi.scrobble(client, id, submission),
    getSimilarSongs: (trackId, count) =>
      subsonicApi.getSimilarSongs(client, trackId, count),
    searchArtistAlbums: (artist, limit) =>
      subsonicApi.searchArtistAlbums(client, artist, limit),
    getArtistInfo: (id) => subsonicApi.getArtistInfo(client, id),
    getLyricsForSong: (song) => subsonicApi.getLyricsForSong(client, song),
    searchLyricsByText: (query, options) =>
      subsonicApi.searchLyricsByText(client, query, options),
  };
}

export function createLocalLibraryAdapter(): MusicLibraryAdapter {
  return {
    getArtists: () => localApi.getArtists(),
    getArtist: async (id) => {
      const result = await localApi.getArtist(id);
      return { ...result, albums: dedupeAlbums(result.albums) };
    },
    getAlbum: (id) => localApi.getAlbum(id),
    getAlbumList2: async (type, size, offset) =>
      dedupeAlbums(await localApi.getAlbumList2(type, size, offset)),
    search3: (query, limit) => localApi.search3(query, limit),
    getSong: (id) => localApi.getSong(id),
    getRandomSongs: (size) => localApi.getRandomSongs(size),
    getGenres: () => localApi.getGenres(),
    getSongsByGenre: (genre, count, offset) =>
      localApi.getSongsByGenre(genre, count, offset),
    getServerPlaylists: () => localApi.getServerPlaylists(),
    getServerPlaylist: (id) => localApi.getServerPlaylist(id),
    getInternetRadioStations: () => localApi.getInternetRadioStations(),
    getStarred2: () => localApi.getStarred2(),
    star: (id) => localApi.star(id),
    unstar: (id) => localApi.unstar(id),
    scrobble: (id, submission) => localApi.scrobble(id, submission),
    getSimilarSongs: (trackId, count) =>
      localApi.getSimilarSongs(trackId, count),
    searchArtistAlbums: (artist, limit) =>
      localApi.searchArtistAlbums(artist, limit),
    getArtistInfo: (id) => localApi.getArtistInfo(id),
    getLyricsForSong: (song) => localApi.getLyricsForSong(song),
    searchLyricsByText: (query, options) =>
      localApi.searchLyricsByText(query).then((hits) => {
        if (!options?.maxResults) return hits;
        return hits.slice(0, options.maxResults);
      }),
  };
}

export function createUnifiedLibraryAdapter(
  subsonic: MusicLibraryAdapter,
  local: MusicLibraryAdapter,
): MusicLibraryAdapter {
  const pick = <T>(
    id: string,
    localFn: () => Promise<T>,
    subFn: () => Promise<T>,
  ) => (isLocalMusicId(id) ? localFn() : subFn());

  return {
    getArtists: async () =>
      mergeUniqueById(
        await settle(subsonic.getArtists(), []),
        await settle(local.getArtists(), []),
      ),
    getArtist: (id) =>
      pick(
        id,
        () => local.getArtist(id),
        () => subsonic.getArtist(id),
      ),
    getAlbum: (id) =>
      pick(
        id,
        () => local.getAlbum(id),
        () => subsonic.getAlbum(id),
      ),
    getAlbumList2: async (type, size, offset) =>
      dedupeAlbums(
        mergeUniqueById(
          await settle(subsonic.getAlbumList2(type, size, offset), []),
          await settle(local.getAlbumList2(type, size, offset), []),
        ),
      ),
    search3: (query, limit) =>
      searchBothLibraries(subsonic, local, query, limit),
    getSong: async (id) => {
      if (isLocalMusicId(id)) {
        return settle(local.getSong(id), null);
      }
      const song = await settle(subsonic.getSong(id), null);
      if (song) return song;
      return settle(local.getSong(id), null);
    },
    getRandomSongs: async (size) => {
      const half = Math.max(1, Math.ceil((size ?? 50) / 2));
      const songs = [
        ...(await settle(subsonic.getRandomSongs(half), [])),
        ...(await settle(local.getRandomSongs(half), [])),
      ];
      return songs.slice(0, size ?? songs.length);
    },
    getGenres: async () =>
      mergeUniqueGenres(
        await settle(subsonic.getGenres(), []),
        await settle(local.getGenres(), []),
      ),
    getSongsByGenre: async (genre, count, offset) => {
      const songs = [
        ...(await settle(subsonic.getSongsByGenre(genre, count, offset), [])),
        ...(await settle(local.getSongsByGenre(genre, count, offset), [])),
      ];
      return songs.slice(0, count ?? songs.length);
    },
    getServerPlaylists: () => subsonic.getServerPlaylists(),
    getServerPlaylist: (id) => subsonic.getServerPlaylist(id),
    getInternetRadioStations: () => subsonic.getInternetRadioStations(),
    getStarred2: () => subsonic.getStarred2(),
    star: (id) =>
      pick(
        id,
        () => local.star(id),
        () => subsonic.star(id),
      ),
    unstar: (id) =>
      pick(
        id,
        () => local.unstar(id),
        () => subsonic.unstar(id),
      ),
    scrobble: (id, submission) =>
      pick(
        id,
        () => local.scrobble(id, submission),
        () => subsonic.scrobble(id, submission),
      ),
    getSimilarSongs: (trackId, count) =>
      pick(
        trackId,
        () => local.getSimilarSongs(trackId, count),
        () => subsonic.getSimilarSongs(trackId, count),
      ),
    searchArtistAlbums: async (artist, limit) =>
      dedupeAlbums(
        mergeUniqueById(
          await settle(subsonic.searchArtistAlbums(artist, limit), []),
          await settle(local.searchArtistAlbums(artist, limit), []),
        ),
      ),
    getArtistInfo: (id) =>
      pick(
        id,
        () => local.getArtistInfo(id),
        () => subsonic.getArtistInfo(id),
      ),
    getLyricsForSong: (song) =>
      pick(
        song.id,
        () => local.getLyricsForSong(song),
        () => subsonic.getLyricsForSong(song),
      ),
    searchLyricsByText: async (query, options) => {
      const hits = mergeLyricsHits(
        await settle(subsonic.searchLyricsByText(query, options), []),
        await settle(local.searchLyricsByText(query, options), []),
      );
      if (!options?.maxResults) return hits;
      return hits.slice(0, options.maxResults);
    },
  };
}
