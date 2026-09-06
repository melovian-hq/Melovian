// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { SubsonicClient } from "./client";
import { dedupeBy } from "$lib/core/collection";
import {
  lyricsMatchText,
  lyricsSnippet,
  mapWithConcurrency,
  parseLyricsText,
  parseStructuredLyricsEntry,
  pickPreferredStructuredEntry,
  type LyricsSearchHit,
  type ParsedLyrics,
} from "$lib/music/lyrics";
import type {
  SubsonicAlbum,
  SubsonicArtist,
  SubsonicArtistInfo,
  SimilarArtist,
  SubsonicGenre,
  SubsonicSearchResult,
  SubsonicSong,
  SubsonicTrackArtist,
  ServerPlaylist,
  StarredContent,
  InternetRadioStation,
} from "./types";
import { buildPlaylistReorderParams } from "$lib/music/server-playlist-editing";

function mapArtist(raw: Record<string, unknown>): SubsonicArtist {
  return {
    id: String(raw.id ?? ""),
    name: String(raw.name ?? ""),
    albumCount: raw.albumCount as number | undefined,
    coverArt: raw.coverArt as string | undefined,
    artistImageUrl: raw.artistImageUrl as string | undefined,
  };
}

function mapAlbum(raw: Record<string, unknown>): SubsonicAlbum {
  return {
    id: String(raw.id ?? ""),
    name: String(raw.name ?? ""),
    artist: raw.artist as string | undefined,
    artistId: raw.artistId as string | undefined,
    year: raw.year as number | undefined,
    songCount: raw.songCount as number | undefined,
    duration: raw.duration as number | undefined,
    coverArt: raw.coverArt as string | undefined,
    genre: raw.genre as string | undefined,
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
  const channels =
    readPositiveInt(raw.channels) ?? readPositiveInt(raw.channelCount);
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
    year: raw.year as number | undefined,
    genre: raw.genre as string | undefined,
    bitRate: raw.bitRate as number | undefined,
    contentType: raw.contentType as string | undefined,
    suffix: raw.suffix as string | undefined,
    transcoded: raw.transcoded === true || raw.isTranscoded === true,
    channels,
    channelCount: channels,
    samplingRate:
      readPositiveInt(raw.samplingRate) ?? readPositiveInt(raw.sampleRate),
    bitDepth: readPositiveInt(raw.bitDepth),
    path: typeof raw.path === "string" ? raw.path : undefined,
  };
}

function readPositiveInt(value: unknown): number | undefined {
  if (typeof value === "number" && Number.isFinite(value) && value > 0) {
    return Math.round(value);
  }
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number.parseInt(value, 10);
    if (Number.isFinite(parsed) && parsed > 0) return parsed;
  }
  return undefined;
}

function unwrapList<T>(
  value: unknown,
  key: string,
  mapper: (raw: Record<string, unknown>) => T,
): T[] {
  if (!value || typeof value !== "object") return [];
  const container = value as Record<string, unknown>;
  const entry = container[key];
  if (!entry) return [];
  if (Array.isArray(entry))
    return entry.map((item) => mapper(item as Record<string, unknown>));
  return [mapper(entry as Record<string, unknown>)];
}

export async function getAlbumList2(
  client: SubsonicClient,
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
  const resp = await client.request<{ albumList2?: { album?: unknown } }>(
    "getAlbumList2.view",
    {
      type,
      size,
      offset,
    },
  );
  return unwrapList(resp.albumList2, "album", mapAlbum);
}

export async function getAlbum(
  client: SubsonicClient,
  id: string,
): Promise<{ album: SubsonicAlbum; songs: SubsonicSong[] }> {
  const resp = await client.request<{
    album?: Record<string, unknown> & { song?: unknown };
  }>("getAlbum.view", { id });
  const raw = resp.album ?? {};
  const album = mapAlbum(raw);
  const songs = unwrapList(raw, "song", mapSong);
  return { album, songs };
}

export async function getArtist(
  client: SubsonicClient,
  id: string,
): Promise<{ artist: SubsonicArtist; albums: SubsonicAlbum[] }> {
  const resp = await client.request<{
    artist?: Record<string, unknown> & { album?: unknown };
  }>("getArtist.view", { id });
  const raw = resp.artist ?? {};
  const artist = mapArtist(raw);
  const albums = unwrapList(raw, "album", mapAlbum);
  return { artist, albums };
}

function mapSimilarArtist(raw: Record<string, unknown>): SimilarArtist {
  return {
    id: raw.id as string | undefined,
    name: String(raw.name ?? ""),
    coverArt: raw.coverArt as string | undefined,
  };
}

function dedupeSimilarArtists(artists: SimilarArtist[]): SimilarArtist[] {
  return dedupeBy(artists, (artist) => artist.id ?? artist.name.toLowerCase());
}

export async function getArtistInfo(
  client: SubsonicClient,
  id: string,
): Promise<SubsonicArtistInfo> {
  const resp = await client.request<{
    artistInfo?: Record<string, unknown> & { similarArtist?: unknown };
  }>("getArtistInfo.view", { id });
  const raw = resp.artistInfo ?? {};
  return {
    biography: raw.biography as string | undefined,
    musicBrainzId: raw.musicBrainzId as string | undefined,
    lastFmUrl: raw.lastFmUrl as string | undefined,
    smallImageUrl: raw.smallImageUrl as string | undefined,
    mediumImageUrl: raw.mediumImageUrl as string | undefined,
    largeImageUrl: raw.largeImageUrl as string | undefined,
    similarArtists: dedupeSimilarArtists(
      unwrapList(raw, "similarArtist", mapSimilarArtist).filter(
        (artist) => artist.name !== "",
      ),
    ),
  };
}

function mapLyricsSearchHit(
  song: {
    id: string;
    artist?: string;
    title: string;
    album?: string;
    duration?: number;
    coverArt?: string;
  },
  lyrics: ParsedLyrics,
  query: string,
): LyricsSearchHit {
  return {
    songId: song.id,
    title: song.title,
    artist: song.artist,
    album: song.album,
    duration: song.duration,
    coverArt: song.coverArt,
    lyrics,
    snippet: lyricsSnippet(lyrics.rawValue, query),
  };
}

function parseLyricsBySongIdResponse(
  body: Record<string, unknown>,
  song: { artist?: string; title: string },
): ParsedLyrics | null {
  const lyricsList = body.lyricsList as Record<string, unknown> | undefined;
  if (lyricsList) {
    const structured = unwrapList(lyricsList, "structuredLyrics", (raw) => raw);
    const preferred = pickPreferredStructuredEntry(structured);
    if (preferred) {
      return parseStructuredLyricsEntry(preferred, {
        artist: song.artist,
        title: song.title,
      });
    }
  }

  const legacy = body.lyrics as
    { value?: string; artist?: string; title?: string } | undefined;
  if (legacy?.value) {
    return parseLyricsText(legacy.value, {
      artist: legacy.artist ?? song.artist,
      title: legacy.title ?? song.title,
    });
  }

  return null;
}

export async function getLyricsForSong(
  client: SubsonicClient,
  song: { id: string; artist?: string; title: string },
): Promise<ParsedLyrics | null> {
  try {
    const byId = await client.request<Record<string, unknown>>(
      "getLyricsBySongId.view",
      { id: song.id },
    );
    const parsed = parseLyricsBySongIdResponse(byId, song);
    if (parsed) return parsed;
  } catch {
    /* OpenSubsonic extension is optional */
  }

  if (!song.artist) return null;

  const resp = await client.request<{
    lyrics?: { value?: string; artist?: string; title?: string };
  }>("getLyrics.view", {
    artist: song.artist,
    title: song.title,
  });

  if (!resp.lyrics?.value) return null;
  return parseLyricsText(resp.lyrics.value, {
    artist: resp.lyrics.artist ?? song.artist,
    title: resp.lyrics.title ?? song.title,
  });
}

export async function searchLyricsByText(
  client: SubsonicClient,
  query: string,
  options: {
    maxResults?: number;
    scanLimit?: number;
    extraSongs?: SubsonicSong[];
  } = {},
): Promise<LyricsSearchHit[]> {
  const needle = query.trim();
  if (!needle) return [];

  const maxResults = options.maxResults ?? 20;
  const scanLimit = options.scanLimit ?? 160;
  const candidates = new Map<string, SubsonicSong>();

  const addSong = (song: SubsonicSong | null | undefined) => {
    if (!song?.id) return;
    if (!candidates.has(song.id)) candidates.set(song.id, song);
  };

  for (const song of options.extraSongs ?? []) addSong(song);

  const searchResult = await search3(client, needle, 40).catch(() => ({
    artists: [],
    albums: [],
    songs: [],
  }));
  for (const song of searchResult.songs) addSong(song);

  if (candidates.size < scanLimit) {
    const [randomSongs, recentAlbums] = await Promise.all([
      getRandomSongs(client, 40).catch(() => [] as SubsonicSong[]),
      getAlbumList2(client, "recent", 8).catch(() => [] as SubsonicAlbum[]),
    ]);
    for (const song of randomSongs) addSong(song);

    const albumSongs = await Promise.all(
      recentAlbums.slice(0, 6).map((album) =>
        getAlbum(client, album.id)
          .then(({ songs }) => songs)
          .catch(() => [] as SubsonicSong[]),
      ),
    );
    for (const songs of albumSongs) {
      for (const song of songs) addSong(song);
    }
  }

  const pool = [...candidates.values()].slice(0, scanLimit);
  const hits = await mapWithConcurrency(pool, 5, async (song) => {
    const lyrics = await getLyricsForSong(client, song).catch(() => null);
    if (!lyrics || !lyricsMatchText(lyrics.rawValue, needle)) return null;
    return mapLyricsSearchHit(song, lyrics, needle);
  });

  hits.sort((a, b) => a.title.localeCompare(b.title));
  return hits.slice(0, maxResults);
}

export async function getArtists(
  client: SubsonicClient,
): Promise<SubsonicArtist[]> {
  const resp = await client.request<{ artists?: { index?: unknown } }>(
    "getArtists.view",
  );
  const indexes = unwrapList(resp.artists, "index", (raw) => raw);
  const artists: SubsonicArtist[] = [];
  for (const index of indexes) {
    artists.push(...unwrapList(index, "artist", mapArtist));
  }
  return artists;
}

export async function getRandomSongs(
  client: SubsonicClient,
  size = 12,
): Promise<SubsonicSong[]> {
  const resp = await client.request<{ randomSongs?: { song?: unknown } }>(
    "getRandomSongs.view",
    { size },
  );
  return unwrapList(resp.randomSongs, "song", mapSong);
}

export async function search3(
  client: SubsonicClient,
  query: string,
  limit = 20,
): Promise<SubsonicSearchResult> {
  const resp = await client.request<{
    searchResult3?: Record<string, unknown>;
  }>("search3.view", {
    query,
    artistCount: limit,
    albumCount: limit,
    songCount: limit,
  });
  const result = resp.searchResult3 ?? {};
  return {
    artists: unwrapList(result, "artist", mapArtist),
    albums: unwrapList(result, "album", mapAlbum),
    songs: unwrapList(result, "song", mapSong),
  };
}

export async function getSong(
  client: SubsonicClient,
  id: string,
): Promise<SubsonicSong | null> {
  const resp = await client.request<{ song?: Record<string, unknown> }>(
    "getSong.view",
    { id },
  );
  if (!resp.song) return null;
  return mapSong(resp.song);
}

function mapGenre(raw: Record<string, unknown>): SubsonicGenre {
  const name = String(raw.value ?? raw.name ?? "");
  return {
    name,
    songCount: raw.songCount as number | undefined,
    albumCount: raw.albumCount as number | undefined,
  };
}

export async function getGenres(
  client: SubsonicClient,
): Promise<SubsonicGenre[]> {
  const resp = await client.request<{ genres?: { genre?: unknown } }>(
    "getGenres.view",
  );
  return unwrapList(resp.genres, "genre", mapGenre)
    .filter((genre) => genre.name !== "")
    .sort((a, b) => (b.songCount ?? 0) - (a.songCount ?? 0));
}

export async function getSongsByGenre(
  client: SubsonicClient,
  genre: string,
  count = 100,
  offset = 0,
): Promise<SubsonicSong[]> {
  const resp = await client.request<{ songsByGenre?: { song?: unknown } }>(
    "getSongsByGenre.view",
    { genre, count, offset },
  );
  return unwrapList(resp.songsByGenre, "song", mapSong);
}

/**
 * scrobble reports a play to the Subsonic server. submission=false registers a
 * "now playing" notification. submission=true records a completed play.
 */
export async function scrobble(
  client: SubsonicClient,
  id: string,
  submission: boolean,
): Promise<void> {
  await client.request("scrobble.view", { id, submission });
}

export async function getSimilarSongs(
  client: SubsonicClient,
  id: string,
  count = 25,
): Promise<SubsonicSong[]> {
  const resp = await client.request<{ similarSongs2?: { song?: unknown } }>(
    "getSimilarSongs2.view",
    { id, count },
  );
  return unwrapList(resp.similarSongs2, "song", mapSong);
}

export async function searchArtistAlbums(
  client: SubsonicClient,
  artistName: string,
  limit = 6,
): Promise<SubsonicAlbum[]> {
  const result = await search3(client, artistName, limit);
  return result.albums.filter(
    (a) => a.artist?.toLowerCase() === artistName.toLowerCase() || a.name,
  );
}

function mapServerPlaylist(raw: Record<string, unknown>): ServerPlaylist {
  return {
    id: String(raw.id ?? ""),
    name: String(raw.name ?? ""),
    songCount: raw.songCount as number | undefined,
    duration: raw.duration as number | undefined,
    coverArt: raw.coverArt as string | undefined,
    owner: raw.owner as string | undefined,
    public: raw.public === true,
    created: raw.created as string | undefined,
    changed: raw.changed as string | undefined,
  };
}

export async function getServerPlaylists(
  client: SubsonicClient,
): Promise<ServerPlaylist[]> {
  const resp = await client.request<{ playlists?: { playlist?: unknown } }>(
    "getPlaylists.view",
  );
  return unwrapList(resp.playlists, "playlist", mapServerPlaylist).filter(
    (pl) => pl.id && pl.name,
  );
}

export async function getServerPlaylist(
  client: SubsonicClient,
  id: string,
): Promise<{ playlist: ServerPlaylist; songs: SubsonicSong[] }> {
  const resp = await client.request<{
    playlist?: Record<string, unknown> & { entry?: unknown; child?: unknown };
  }>("getPlaylist.view", { id });
  const raw = resp.playlist ?? {};
  const playlist = mapServerPlaylist(raw);
  const songs = [
    ...unwrapList(raw, "entry", mapSong),
    ...unwrapList(raw, "child", mapSong),
  ];
  const seen = new Set<string>();
  const uniqueSongs = songs.filter((song) => {
    if (!song.id || seen.has(song.id)) return false;
    seen.add(song.id);
    return true;
  });
  if (
    playlist.songCount === undefined ||
    playlist.songCount < uniqueSongs.length
  ) {
    playlist.songCount = uniqueSongs.length;
  }
  return { playlist, songs: uniqueSongs };
}

export interface UpdateServerPlaylistParams {
  playlistId: string;
  name?: string;
  comment?: string;
  public?: boolean;
  songIdToAdd?: string[];
  songIndexToRemove?: number[];
}

export async function createServerPlaylist(
  client: SubsonicClient,
  name: string,
  songIds: string[] = [],
): Promise<ServerPlaylist> {
  const resp = await client.requestPost<{
    playlist?: Record<string, unknown>;
  }>("createPlaylist.view", {
    name,
    songId: songIds,
  });
  const raw = resp.playlist;
  if (!raw || typeof raw !== "object") {
    throw new Error("Server did not return the created playlist");
  }
  return mapServerPlaylist(raw);
}

export async function updateServerPlaylist(
  client: SubsonicClient,
  params: UpdateServerPlaylistParams,
): Promise<void> {
  const {
    playlistId,
    name,
    comment,
    public: isPublic,
    songIdToAdd,
    songIndexToRemove,
  } = params;
  await client.requestPost("updatePlaylist.view", {
    playlistId,
    name,
    comment,
    public: isPublic,
    songIdToAdd,
    songIndexToRemove,
  });
}

export async function deleteServerPlaylist(
  client: SubsonicClient,
  id: string,
): Promise<void> {
  await client.requestPost("deletePlaylist.view", { id });
}

export async function addSongsToServerPlaylist(
  client: SubsonicClient,
  playlistId: string,
  songIds: string[],
): Promise<void> {
  const unique = [...new Set(songIds.filter(Boolean))];
  if (unique.length === 0) return;
  await updateServerPlaylist(client, {
    playlistId,
    songIdToAdd: unique,
  });
}

export async function removeSongFromServerPlaylist(
  client: SubsonicClient,
  playlistId: string,
  songIndex: number,
): Promise<void> {
  if (songIndex < 0) {
    throw new Error("Invalid song index");
  }
  await updateServerPlaylist(client, {
    playlistId,
    songIndexToRemove: [songIndex],
  });
}

export async function setServerPlaylistSongOrder(
  client: SubsonicClient,
  playlistId: string,
  currentSongIds: readonly string[],
  nextSongIds: readonly string[],
): Promise<void> {
  const params = buildPlaylistReorderParams(currentSongIds, nextSongIds);
  if (!params) return;
  await updateServerPlaylist(client, {
    playlistId,
    songIdToAdd: params.songIdToAdd,
    songIndexToRemove: params.songIndexToRemove,
  });
}

export async function getStarred2(
  client: SubsonicClient,
): Promise<StarredContent> {
  const resp = await client.request<{ starred2?: Record<string, unknown> }>(
    "getStarred2.view",
  );
  const starred = resp.starred2 ?? {};
  return {
    songs: unwrapList(starred, "song", mapSong),
    albums: unwrapList(starred, "album", mapAlbum),
    artists: unwrapList(starred, "artist", mapArtist),
  };
}

export async function star(client: SubsonicClient, id: string): Promise<void> {
  await client.request("star.view", { id });
}

export async function unstar(
  client: SubsonicClient,
  id: string,
): Promise<void> {
  await client.request("unstar.view", { id });
}

function mapInternetRadioStation(
  raw: Record<string, unknown>,
): InternetRadioStation {
  return {
    id: String(raw.id ?? ""),
    name: String(raw.name ?? ""),
    streamUrl: String(raw.streamUrl ?? ""),
    homePageUrl: raw.homePageUrl as string | undefined,
    coverArt: raw.coverArt as string | undefined,
  };
}

export async function getInternetRadioStations(
  client: SubsonicClient,
): Promise<InternetRadioStation[]> {
  const resp = await client.request<{
    internetRadioStations?: { internetRadioStation?: unknown };
  }>("getInternetRadioStations.view");
  return unwrapList(
    resp.internetRadioStations,
    "internetRadioStation",
    mapInternetRadioStation,
  ).filter((station) => station.id && station.name && station.streamUrl);
}
