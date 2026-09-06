// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
import { requireOk } from "$lib/core/http/errors";
import { resolveMediaUrl } from "$lib/config/runtime";
import { ApiPaths } from "$lib/core/http/api-paths";
import {
  lyricsMatchText,
  lyricsSnippet,
  mapWithConcurrency,
  type LyricsSearchHit,
  type ParsedLyrics,
} from "$lib/music/lyrics";
import * as musicApi from "$lib/music/api";
import type {
  SubsonicAlbum,
  SubsonicArtist,
  SubsonicArtistInfo,
  SubsonicGenre,
  SubsonicSearchResult,
  SubsonicSong,
  SubsonicTrackArtist,
  InternetRadioStation,
  ServerPlaylist,
  StarredContent,
} from "$lib/subsonic/types";
import type { FavoriteTrack } from "$lib/subsonic/types";

function mapArtist(raw: Record<string, unknown>): SubsonicArtist {
  return {
    id: String(raw.id ?? ""),
    name: String(raw.name ?? ""),
    albumCount: raw.albumCount as number | undefined,
    coverArt: raw.coverArt as string | undefined,
  };
}

function mapAlbum(raw: Record<string, unknown>): SubsonicAlbum {
  return {
    id: String(raw.id ?? ""),
    name: String(raw.name ?? ""),
    artist: raw.artist as string | undefined,
    artistId: raw.artistId as string | undefined,
    songCount: raw.songCount as number | undefined,
    duration: raw.duration as number | undefined,
    coverArt: raw.coverArt as string | undefined,
  };
}

function mapTrackArtist(raw: Record<string, unknown>): SubsonicTrackArtist {
  return {
    id: String(raw.id ?? ""),
    name: String(raw.name ?? ""),
  };
}

function unwrapTrackArtists(value: unknown): SubsonicTrackArtist[] {
  if (!value) return [];
  if (Array.isArray(value)) {
    return value
      .map((item) => mapTrackArtist(item as Record<string, unknown>))
      .filter((artist) => artist.name.length > 0);
  }
  if (typeof value === "object") {
    const container = value as Record<string, unknown>;
    if (container.artist !== undefined) {
      return unwrapTrackArtists(container.artist);
    }
    const artist = mapTrackArtist(container);
    return artist.name ? [artist] : [];
  }
  return [];
}

function mapSong(raw: Record<string, unknown>): SubsonicSong {
  const artists = unwrapTrackArtists(raw.artists);
  return {
    id: String(raw.id ?? ""),
    title: String(raw.title ?? ""),
    album: raw.album as string | undefined,
    albumId: raw.albumId as string | undefined,
    artist: raw.artist as string | undefined,
    artistId: raw.artistId as string | undefined,
    artists: artists.length > 0 ? artists : undefined,
    track: raw.track as number | undefined,
    duration: raw.duration as number | undefined,
    coverArt: raw.coverArt as string | undefined,
    suffix: raw.suffix as string | undefined,
    genre: raw.genre as string | undefined,
    starred: raw.starred as string | undefined,
    playCount: raw.playCount as number | undefined,
  };
}

function favoriteToLocalSong(fav: FavoriteTrack): SubsonicSong {
  return {
    id: fav.trackId,
    title: fav.trackTitle,
    album: fav.albumTitle || undefined,
    albumId: fav.albumId || undefined,
    artist: fav.artistName || undefined,
    duration: fav.durationMs ? fav.durationMs / 1000 : undefined,
    coverArt: fav.coverArtId || undefined,
    starred: fav.favoritedAt,
  };
}

async function localGet<T>(path: string): Promise<T> {
  const response = await fetchWithRetry(path, { headers: apiHeaders() });
  await requireOk(response, `Local music request failed: ${response.status}`);
  return (await response.json()) as T;
}

export async function getArtists(): Promise<SubsonicArtist[]> {
  const payload = await localGet<{ artists: Record<string, unknown>[] }>(
    "/api/local-music/artists",
  );
  return (payload.artists ?? []).map(mapArtist);
}

export async function getArtist(
  id: string,
): Promise<{ artist: SubsonicArtist; albums: SubsonicAlbum[] }> {
  const payload = await localGet<{
    artist: Record<string, unknown>;
    albums: Record<string, unknown>[];
  }>(`/api/local-music/artists/${encodeURIComponent(id)}`);
  return {
    artist: mapArtist(payload.artist ?? {}),
    albums: (payload.albums ?? []).map(mapAlbum),
  };
}

export async function getAlbum(
  id: string,
): Promise<{ album: SubsonicAlbum; songs: SubsonicSong[] }> {
  const payload = await localGet<{
    album: Record<string, unknown>;
    songs: Record<string, unknown>[];
  }>(`/api/local-music/albums/${encodeURIComponent(id)}`);
  return {
    album: mapAlbum(payload.album ?? {}),
    songs: (payload.songs ?? []).map(mapSong),
  };
}

export async function getAlbumList2(
  type:
    | "newest"
    | "recent"
    | "frequent"
    | "random"
    | "alphabeticalByName"
    | "byGenre",
  size = 18,
  offset = 0,
): Promise<SubsonicAlbum[]> {
  const params = new URLSearchParams({
    type,
    size: String(size),
    offset: String(offset),
  });
  const payload = await localGet<{ albums: Record<string, unknown>[] }>(
    `/api/local-music/albums?${params.toString()}`,
  );
  return (payload.albums ?? []).map(mapAlbum);
}

export async function search3(
  query: string,
  limit = 20,
): Promise<SubsonicSearchResult> {
  const params = new URLSearchParams({ q: query, limit: String(limit) });
  const payload = await localGet<{
    artists: Record<string, unknown>[];
    albums: Record<string, unknown>[];
    songs: Record<string, unknown>[];
  }>(`/api/local-music/search?${params.toString()}`);
  return {
    artists: (payload.artists ?? []).map(mapArtist),
    albums: (payload.albums ?? []).map(mapAlbum),
    songs: (payload.songs ?? []).map(mapSong),
  };
}

export async function getSong(id: string): Promise<SubsonicSong | null> {
  try {
    const payload = await localGet<Record<string, unknown>>(
      `/api/local-music/songs/${encodeURIComponent(id)}`,
    );
    return mapSong(payload);
  } catch {
    return null;
  }
}

export async function getRandomSongs(size = 12): Promise<SubsonicSong[]> {
  const params = new URLSearchParams({ size: String(size) });
  const payload = await localGet<{ songs: Record<string, unknown>[] }>(
    `/api/local-music/randomSongs?${params.toString()}`,
  );
  return (payload.songs ?? []).map(mapSong);
}

export async function getGenres(): Promise<SubsonicGenre[]> {
  const payload = await localGet<{ genres: Record<string, unknown>[] }>(
    "/api/local-music/genres",
  );
  return (payload.genres ?? []).map((raw) => ({
    name: String(raw.name ?? ""),
    songCount: raw.songCount as number | undefined,
    albumCount: raw.albumCount as number | undefined,
  }));
}

export async function getSongsByGenre(
  genre: string,
  count = 50,
  offset = 0,
): Promise<SubsonicSong[]> {
  const params = new URLSearchParams({
    genre,
    count: String(count),
    offset: String(offset),
  });
  const payload = await localGet<{ songs: Record<string, unknown>[] }>(
    `/api/local-music/songsByGenre?${params.toString()}`,
  );
  return (payload.songs ?? []).map(mapSong);
}

export async function getServerPlaylists(): Promise<ServerPlaylist[]> {
  return [];
}

export async function getServerPlaylist(
  _id: string,
): Promise<{ playlist: ServerPlaylist; songs: SubsonicSong[] } | null> {
  return null;
}

export async function getInternetRadioStations(): Promise<
  InternetRadioStation[]
> {
  return [];
}

export async function getStarred2(): Promise<StarredContent> {
  const payload = await localGet<{
    songs: Record<string, unknown>[];
    albums: Record<string, unknown>[];
    artists: Record<string, unknown>[];
  }>("/api/local-music/starred");
  const songs = (payload.songs ?? []).map(mapSong);
  // The endpoint only returns tracks still present in the catalog. Favorited
  // tracks whose files went missing are merged in from the favorites list so
  // the favorites page stays complete.
  const presentIds = new Set(songs.map((song) => song.id));
  const favorites = await musicApi.listFavorites(2000).catch(() => []);
  for (const fav of favorites) {
    if (!fav.trackId.startsWith("trk_") || presentIds.has(fav.trackId)) {
      continue;
    }
    songs.push(favoriteToLocalSong(fav));
  }
  return { songs, albums: [], artists: [] };
}

export async function star(id: string): Promise<void> {
  const song = await getSong(id);
  await musicApi.addFavorite(id, {
    trackTitle: song?.title ?? "",
    artistName: song?.artist ?? "",
    albumId: song?.albumId ?? "",
    albumTitle: song?.album ?? "",
    durationMs: (song?.duration ?? 0) * 1000,
    coverArtId: song?.coverArt ?? song?.albumId ?? id,
  });
}

export async function unstar(id: string): Promise<void> {
  await musicApi.removeFavorite(id);
}

// External scrobblers (Last.fm, ListenBrainz, Rocksky) are dispatched
// generically by the player, so local scrobble only marks the track played in
// the built-in listen history.
export async function scrobble(
  id: string,
  submission: boolean,
): Promise<void> {
  if (!submission) return;
  await musicApi.markTrackPlayed(id);
}

export async function getSimilarSongs(
  trackId: string,
  count = 20,
): Promise<SubsonicSong[]> {
  const params = new URLSearchParams({ count: String(count) });
  const payload = await localGet<{ songs: Record<string, unknown>[] }>(
    `/api/local-music/songs/${encodeURIComponent(trackId)}/similar?${params.toString()}`,
  );
  return (payload.songs ?? []).map(mapSong);
}

export async function searchArtistAlbums(
  artist: string,
  limit = 20,
): Promise<SubsonicAlbum[]> {
  const result = await search3(artist, limit);
  const needle = artist.toLowerCase();
  return result.albums.filter(
    (album) => album.artist?.toLowerCase() === needle,
  );
}

export async function getArtistInfo(_id: string): Promise<SubsonicArtistInfo> {
  return { similarArtists: [] };
}

export async function getLyricsForSong(song: {
  id: string;
  artist?: string;
  title: string;
  album?: string;
  duration?: number;
}): Promise<ParsedLyrics | null> {
  return musicApi.getTrackLyrics(song.id, {
    artist: song.artist,
    title: song.title,
    album: song.album,
    durationSec: song.duration,
  });
}

export async function searchLyricsByText(
  query: string,
  options: { maxResults?: number; scanLimit?: number } = {},
): Promise<LyricsSearchHit[]> {
  const needle = query.trim();
  if (!needle) return [];

  const maxResults = options.maxResults ?? 20;
  const scanLimit = options.scanLimit ?? 60;
  const candidates = new Map<string, SubsonicSong>();

  const searchResult = await search3(needle, 40).catch(() => ({
    artists: [] as SubsonicArtist[],
    albums: [] as SubsonicAlbum[],
    songs: [] as SubsonicSong[],
  }));
  for (const song of searchResult.songs) {
    if (song.id) candidates.set(song.id, song);
  }
  if (candidates.size < scanLimit) {
    const random = await getRandomSongs(40).catch(() => [] as SubsonicSong[]);
    for (const song of random) {
      if (song.id && !candidates.has(song.id)) candidates.set(song.id, song);
    }
  }

  const pool = [...candidates.values()].slice(0, scanLimit);
  const hits = await mapWithConcurrency(pool, 5, async (song) => {
    const lyrics = await getLyricsForSong(song).catch(() => null);
    if (!lyrics || !lyricsMatchText(lyrics.rawValue, needle)) return null;
    return {
      songId: song.id,
      title: song.title,
      artist: song.artist,
      album: song.album,
      duration: song.duration,
      coverArt: song.coverArt,
      lyrics,
      snippet: lyricsSnippet(lyrics.rawValue, needle),
    } satisfies LyricsSearchHit;
  });

  hits.sort((a, b) => a.title.localeCompare(b.title));
  return hits.slice(0, maxResults);
}

export function localStreamUrl(trackId: string): string {
  return resolveMediaUrl(
    `/api/local-music/tracks/${encodeURIComponent(trackId)}/stream`,
  );
}

export function localCoverArtUrl(id?: string, size = 300): string | null {
  if (!id) return null;
  const params = new URLSearchParams({ size: String(size) });
  return resolveMediaUrl(
    `/api/local-music/cover/${encodeURIComponent(id)}?${params.toString()}`,
  );
}
