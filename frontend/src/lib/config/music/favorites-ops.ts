// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import * as musicApi from "$lib/music/api";
import type {
  SubsonicAlbum,
  SubsonicArtist,
  SubsonicSong,
} from "$lib/subsonic";
import type { FavoriteTrack } from "$lib/subsonic/types";
import { toast } from "$lib/ui/toast.svelte";
import { favoriteToSong } from "./helpers";

export interface MusicFavoritesContext {
  connected: boolean;
  library: MusicLibraryAdapter;
  favoriteTracks: FavoriteTrack[];
  favoriteIds: Set<string>;
  favoriteAlbums: SubsonicAlbum[];
  favoriteAlbumIds: Set<string>;
  favoriteArtists: SubsonicArtist[];
  favoriteArtistIds: Set<string>;
  shuffle: boolean;
  isFavorite(trackId: string): boolean;
  isFavoriteAlbum(albumId: string): boolean;
  isFavoriteArtist(artistId: string): boolean;
  playArtistAlbums(albums: SubsonicAlbum[], shuffle?: boolean): Promise<void>;
  playTracks(tracks: SubsonicSong[], startIndex?: number): void;
}

export async function refreshFavorites(ctx: MusicFavoritesContext) {
  const local = await musicApi
    .listFavorites()
    .catch(() => [] as FavoriteTrack[]);
  const starred = ctx.connected
    ? await ctx.library.getStarred2().catch(() => ({
        songs: [] as SubsonicSong[],
        albums: [],
        artists: [],
      }))
    : { songs: [] as SubsonicSong[], albums: [], artists: [] };

  const localIds = new Set(local.map((item) => item.trackId));
  const serverOnly = starred.songs
    .filter((song) => !localIds.has(song.id))
    .map((song): FavoriteTrack => ({
      trackId: song.id,
      trackTitle: song.title,
      artistName: song.artist ?? "",
      albumId: song.albumId ?? "",
      albumTitle: song.album ?? "",
      durationMs: (song.duration ?? 0) * 1000,
      coverArtId: song.coverArt ?? song.albumId ?? song.id,
      favoritedAt: new Date().toISOString(),
    }));

  ctx.favoriteTracks = [...local, ...serverOnly];
  ctx.favoriteIds = new Set([
    ...local.map((item) => item.trackId),
    ...starred.songs.map((song) => song.id),
  ]);
  ctx.favoriteAlbums = starred.albums;
  ctx.favoriteAlbumIds = new Set(starred.albums.map((album) => album.id));
  ctx.favoriteArtists = starred.artists;
  ctx.favoriteArtistIds = new Set(starred.artists.map((artist) => artist.id));
}

export async function toggleFavoriteAlbum(
  ctx: MusicFavoritesContext,
  album: SubsonicAlbum,
) {
  if (!ctx.connected) return;
  if (ctx.isFavoriteAlbum(album.id)) {
    await ctx.library.unstar(album.id).catch(() => {});
    ctx.favoriteAlbums = ctx.favoriteAlbums.filter(
      (item) => item.id !== album.id,
    );
    const next = new Set(ctx.favoriteAlbumIds);
    next.delete(album.id);
    ctx.favoriteAlbumIds = next;
    toast.success(`Removed ${album.name} from favorites`);
    return;
  }

  await ctx.library.star(album.id).catch(() => {});
  ctx.favoriteAlbums = [album, ...ctx.favoriteAlbums];
  ctx.favoriteAlbumIds = new Set([album.id, ...ctx.favoriteAlbumIds]);
  toast.success(`Added ${album.name} to favorites`);
}

export async function toggleFavoriteArtist(
  ctx: MusicFavoritesContext,
  artist: SubsonicArtist,
) {
  if (!ctx.connected) return;
  if (ctx.isFavoriteArtist(artist.id)) {
    await ctx.library.unstar(artist.id).catch(() => {});
    ctx.favoriteArtists = ctx.favoriteArtists.filter(
      (item) => item.id !== artist.id,
    );
    const next = new Set(ctx.favoriteArtistIds);
    next.delete(artist.id);
    ctx.favoriteArtistIds = next;
    toast.success(`Removed ${artist.name} from favorites`);
    return;
  }

  await ctx.library.star(artist.id).catch(() => {});
  ctx.favoriteArtists = [artist, ...ctx.favoriteArtists];
  ctx.favoriteArtistIds = new Set([artist.id, ...ctx.favoriteArtistIds]);
  toast.success(`Added ${artist.name} to favorites`);
}

export async function playFavoriteAlbums(
  ctx: MusicFavoritesContext,
  albums: SubsonicAlbum[],
) {
  if (albums.length === 0) return;
  await ctx.playArtistAlbums(albums, false);
}

export async function playFavoriteArtists(
  ctx: MusicFavoritesContext,
  artists: SubsonicArtist[],
) {
  const albums: SubsonicAlbum[] = [];
  for (const artist of artists) {
    const detail = await ctx.library.getArtist(artist.id).catch(() => null);
    if (detail) albums.push(...detail.albums);
  }
  if (albums.length === 0) return;
  await ctx.playArtistAlbums(albums, false);
}

export async function toggleFavorite(
  ctx: MusicFavoritesContext,
  track: SubsonicSong,
) {
  if (ctx.isFavorite(track.id)) {
    await Promise.all([
      musicApi.removeFavorite(track.id).catch(() => {}),
      ctx.connected
        ? ctx.library.unstar(track.id).catch(() => {})
        : Promise.resolve(),
    ]);
    ctx.favoriteTracks = ctx.favoriteTracks.filter(
      (item) => item.trackId !== track.id,
    );
    const next = new Set(ctx.favoriteIds);
    next.delete(track.id);
    ctx.favoriteIds = next;
    toast.success(`Removed ${track.title} from favorites`);
    return;
  }

  await Promise.all([
    musicApi.addFavorite(track.id, {
      trackTitle: track.title,
      artistName: track.artist ?? "",
      albumId: track.albumId ?? "",
      albumTitle: track.album ?? "",
      durationMs: (track.duration ?? 0) * 1000,
      coverArtId: track.coverArt ?? track.albumId ?? track.id,
    }),
    ctx.connected
      ? ctx.library.star(track.id).catch(() => {})
      : Promise.resolve(),
  ]);
  const entry: FavoriteTrack = {
    trackId: track.id,
    trackTitle: track.title,
    artistName: track.artist ?? "",
    albumId: track.albumId ?? "",
    albumTitle: track.album ?? "",
    durationMs: (track.duration ?? 0) * 1000,
    coverArtId: track.coverArt ?? track.albumId ?? track.id,
    favoritedAt: new Date().toISOString(),
  };
  ctx.favoriteTracks = [entry, ...ctx.favoriteTracks];
  ctx.favoriteIds = new Set([track.id, ...ctx.favoriteIds]);
  toast.success(`Added ${track.title} to favorites`);
}

export function playFavorites(ctx: MusicFavoritesContext, startIndex = 0) {
  const tracks = ctx.favoriteTracks.map((entry) => favoriteToSong(entry));
  if (tracks.length === 0) return;
  ctx.playTracks(tracks, startIndex);
}

export function playAllFavorites(ctx: MusicFavoritesContext, shuffle = false) {
  if (ctx.favoriteTracks.length === 0) return;
  if (shuffle) ctx.shuffle = true;
  playFavorites(ctx, 0);
}

export function createFavoritesOps(ctx: MusicFavoritesContext) {
  return {
    refreshFavorites: () => refreshFavorites(ctx),
    toggleFavoriteAlbum: (album: SubsonicAlbum) =>
      toggleFavoriteAlbum(ctx, album),
    toggleFavoriteArtist: (artist: SubsonicArtist) =>
      toggleFavoriteArtist(ctx, artist),
    toggleFavorite: (track: SubsonicSong) => toggleFavorite(ctx, track),
    playFavoriteAlbums: (albums: SubsonicAlbum[]) =>
      playFavoriteAlbums(ctx, albums),
    playFavoriteArtists: (artists: SubsonicArtist[]) =>
      playFavoriteArtists(ctx, artists),
    playFavorites: (startIndex = 0) => playFavorites(ctx, startIndex),
    playAllFavorites: (shuffle = false) => playAllFavorites(ctx, shuffle),
  };
}
