// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import { prewarmGenreArt } from "$lib/music/genre-art";
import * as musicApi from "$lib/music/api";
import { suggestion as fuzzySuggestion } from "$lib/music/search";
import type {
  ListenEntry,
  ListenStats,
  LibraryStats,
} from "$lib/subsonic/types";
import type {
  SubsonicAlbum,
  SubsonicArtist,
  SubsonicGenre,
  SubsonicSearchResult,
  SubsonicSong,
} from "$lib/subsonic";
import { shuffleTracks } from "./helpers";

export const ARTIST_INDEX_TTL_MS = 5 * 60 * 1000;

export interface MusicLibraryBrowseContext {
  library: MusicLibraryAdapter;
  recentAlbums: SubsonicAlbum[];
  frequentAlbums: SubsonicAlbum[];
  randomTracks: SubsonicSong[];
  resumeTracks: ListenEntry[];
  homeCoreFetchedAt: number;
  listenHistory: ListenEntry[];
  stats: ListenStats | null;
  libraryStats: LibraryStats | null;
  genres: SubsonicGenre[];
  allArtists: SubsonicArtist[];
  artistsLoaded: boolean;
  artistsLoadedAt: number;
  shuffle: boolean;
  playTracks(tracks: SubsonicSong[], startIndex?: number): void;
  addTracksToQueue(tracks: SubsonicSong[]): void;
  playTracksNext(tracks: SubsonicSong[]): void;
}

export function invalidateArtistIndex(ctx: MusicLibraryBrowseContext) {
  ctx.artistsLoaded = false;
  ctx.artistsLoadedAt = 0;
}

export async function ensureArtists(ctx: MusicLibraryBrowseContext) {
  const fresh =
    ctx.artistsLoaded && Date.now() - ctx.artistsLoadedAt < ARTIST_INDEX_TTL_MS;
  if (fresh) return;
  try {
    ctx.allArtists = await ctx.library.getArtists();
    ctx.artistsLoadedAt = Date.now();
  } catch {
    /* suggestions degrade gracefully without the artist index */
  }
  ctx.artistsLoaded = true;
}

export async function refreshHomeCore(ctx: MusicLibraryBrowseContext) {
  const [recent, frequent, random, resume] = await Promise.all([
    ctx.library.getAlbumList2("newest", 12),
    ctx.library.getAlbumList2("frequent", 12),
    ctx.library.getRandomSongs(8),
    musicApi.getResumeTracks(12).catch(() => [] as ListenEntry[]),
  ]);
  ctx.recentAlbums = recent;
  ctx.frequentAlbums = frequent;
  ctx.randomTracks = random;
  ctx.resumeTracks = resume;
  ctx.homeCoreFetchedAt = Date.now();
}

export async function refreshHistory(
  ctx: MusicLibraryBrowseContext,
  limit = 50,
) {
  ctx.listenHistory = await musicApi.getListenHistory(limit);
}

export async function clearListenHistory(ctx: MusicLibraryBrowseContext) {
  await musicApi.clearListenEvents();
  ctx.listenHistory = [];
  ctx.resumeTracks = [];
  await refreshStats(ctx).catch(() => {});
}

export async function refreshStats(ctx: MusicLibraryBrowseContext, limit = 8) {
  ctx.stats = await musicApi.getListenStats(limit);
}

export async function refreshLibraryStats(
  ctx: MusicLibraryBrowseContext,
  options?: { bypassCache?: boolean },
) {
  const stats = await musicApi.getLibraryStats(options);
  if (stats) ctx.libraryStats = stats;
}

export async function search(ctx: MusicLibraryBrowseContext, term: string) {
  if (!term.trim()) return { artists: [], albums: [], songs: [] };
  return ctx.library.search3(term.trim(), 12);
}

export async function searchAll(
  ctx: MusicLibraryBrowseContext,
  term: string,
): Promise<{
  query: string;
  result: SubsonicSearchResult;
  suggestion: string | null;
  similar: SubsonicSong[];
}> {
  const query = term.trim();
  const empty: SubsonicSearchResult = { artists: [], albums: [], songs: [] };
  if (!query) {
    return { query, result: empty, suggestion: null, similar: [] };
  }

  const result = await ctx.library.search3(query, 20).catch(() => empty);
  const hits =
    result.artists.length + result.albums.length + result.songs.length;

  let didYouMean: string | null = null;
  if (hits < 4) {
    const labels = [
      ...result.artists.map((a) => a.name),
      ...result.albums.map((a) => a.name),
      ...result.songs.map((s) => s.title),
      ...result.songs.map((s) => s.artist ?? ""),
    ].filter(Boolean);
    didYouMean = fuzzySuggestion(query, labels);
  }

  return { query, result, suggestion: didYouMean, similar: [] };
}

export function searchSimilarTracks(
  ctx: MusicLibraryBrowseContext,
  trackId: string,
  count = 12,
): Promise<SubsonicSong[]> {
  return ctx.library.getSimilarSongs(trackId, count).catch(() => []);
}

export async function loadGenres(ctx: MusicLibraryBrowseContext) {
  ctx.genres = await ctx.library.getGenres().catch(() => []);
  prewarmGenreArt(ctx.genres.map((genre) => genre.name));
}

export async function loadArtists(
  ctx: MusicLibraryBrowseContext,
  options?: { force?: boolean },
) {
  if (options?.force) invalidateArtistIndex(ctx);
  await ensureArtists(ctx);
  return ctx.allArtists;
}

export async function getGenreSongs(
  ctx: MusicLibraryBrowseContext,
  genre: string,
  count = 200,
  offset = 0,
): Promise<SubsonicSong[]> {
  return ctx.library.getSongsByGenre(genre, count, offset).catch(() => []);
}

export async function playGenre(
  ctx: MusicLibraryBrowseContext,
  genre: string,
  shuffle = true,
) {
  const songs = await getGenreSongs(ctx, genre, 200);
  if (songs.length === 0) return;
  if (shuffle) ctx.shuffle = true;
  ctx.playTracks(shuffle ? shuffleTracks(songs) : songs, 0);
}

export async function addGenreToQueue(
  ctx: MusicLibraryBrowseContext,
  genre: string,
) {
  const songs = await getGenreSongs(ctx, genre, 200);
  if (songs.length === 0) return;
  ctx.addTracksToQueue(songs);
}

export async function playGenreNext(
  ctx: MusicLibraryBrowseContext,
  genre: string,
) {
  const songs = await getGenreSongs(ctx, genre, 200);
  if (songs.length === 0) return;
  ctx.playTracksNext(songs);
}

export async function playArtistAlbums(
  ctx: MusicLibraryBrowseContext,
  albums: SubsonicAlbum[],
  shuffle = false,
) {
  const tracks: SubsonicSong[] = [];
  for (const album of albums) {
    const detail = await ctx.library.getAlbum(album.id).catch(() => null);
    if (detail) tracks.push(...detail.songs);
  }
  if (tracks.length === 0) return;
  if (shuffle) ctx.shuffle = true;
  ctx.playTracks(tracks, 0);
}

export function createLibraryBrowseOps(ctx: MusicLibraryBrowseContext) {
  return {
    refreshHomeCore: () => refreshHomeCore(ctx),
    refreshHistory: (limit?: number) => refreshHistory(ctx, limit),
    clearListenHistory: () => clearListenHistory(ctx),
    refreshStats: (limit?: number) => refreshStats(ctx, limit),
    refreshLibraryStats: (options?: { bypassCache?: boolean }) =>
      refreshLibraryStats(ctx, options),
    search: (term: string) => search(ctx, term),
    searchAll: (term: string) => searchAll(ctx, term),
    searchSimilarTracks: (trackId: string, count?: number) =>
      searchSimilarTracks(ctx, trackId, count),
    loadGenres: () => loadGenres(ctx),
    loadArtists: (options?: { force?: boolean }) => loadArtists(ctx, options),
    getGenreSongs: (genre: string, count?: number, offset?: number) =>
      getGenreSongs(ctx, genre, count, offset),
    playGenre: (genre: string, shuffle?: boolean) =>
      playGenre(ctx, genre, shuffle),
    addGenreToQueue: (genre: string) => addGenreToQueue(ctx, genre),
    playGenreNext: (genre: string) => playGenreNext(ctx, genre),
    playArtistAlbums: (albums: SubsonicAlbum[], shuffle?: boolean) =>
      playArtistAlbums(ctx, albums, shuffle),
    invalidateArtistIndex: () => invalidateArtistIndex(ctx),
  };
}
