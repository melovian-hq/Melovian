// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

/**
 * Subsonic view responses are loose envelopes: a top-level key holds either a
 * record or a container with a scalar-or-list payload. Entry contents stay
 * unknown because the mapSong/mapAlbum helpers already normalize them.
 */
const container = (entryKey: string) =>
  v.looseObject({ [entryKey]: v.optional(v.unknown()) });

// Empty containers can arrive as null on some servers.
const optionalContainer = (entryKey: string) =>
  v.optional(v.nullable(container(entryKey)));

const optionalRecord = v.optional(v.nullable(v.looseObject({})));

export const subsonicAlbumListResponseSchema = v.looseObject({
  albumList2: optionalContainer("album"),
});

export const subsonicAlbumResponseSchema = v.looseObject({
  album: optionalContainer("song"),
});

export const subsonicArtistResponseSchema = v.looseObject({
  artist: optionalContainer("album"),
});

export const subsonicArtistInfoResponseSchema = v.looseObject({
  artistInfo: optionalContainer("similarArtist"),
});

export const subsonicLyricsBySongIdResponseSchema = v.looseObject({});

export const subsonicLyricsResponseSchema = v.looseObject({
  lyrics: v.optional(
    v.nullable(
      v.looseObject({
        value: v.optional(v.string()),
        artist: v.optional(v.string()),
        title: v.optional(v.string()),
      }),
    ),
  ),
});

export const subsonicArtistsResponseSchema = v.looseObject({
  artists: optionalContainer("index"),
});

export const subsonicRandomSongsResponseSchema = v.looseObject({
  randomSongs: optionalContainer("song"),
});

export const subsonicSearchResponseSchema = v.looseObject({
  searchResult3: optionalRecord,
});

export const subsonicSongResponseSchema = v.looseObject({
  song: optionalRecord,
});

export const subsonicGenresResponseSchema = v.looseObject({
  genres: optionalContainer("genre"),
});

export const subsonicSongsByGenreResponseSchema = v.looseObject({
  songsByGenre: optionalContainer("song"),
});

export const subsonicSimilarSongsResponseSchema = v.looseObject({
  similarSongs2: optionalContainer("song"),
});

export const subsonicPlaylistsResponseSchema = v.looseObject({
  playlists: optionalContainer("playlist"),
});

export const subsonicPlaylistResponseSchema = v.looseObject({
  playlist: v.optional(
    v.nullable(v.looseObject({ entry: v.unknown(), child: v.unknown() })),
  ),
});

export const subsonicCreatePlaylistResponseSchema = v.looseObject({
  playlist: optionalRecord,
});

export const subsonicStarredResponseSchema = v.looseObject({
  starred2: optionalRecord,
});

export const subsonicInternetRadioStationsResponseSchema = v.looseObject({
  internetRadioStations: optionalContainer("internetRadioStation"),
});

// The JSON envelope every /rest endpoint wraps its payload in. The inner
// payload is validated per endpoint in api.ts, so it stays loose here.
export const subsonicEnvelopeSchema = v.looseObject({
  "subsonic-response": v.optional(
    v.looseObject({
      status: v.optional(v.string()),
      error: v.optional(
        v.looseObject({
          code: v.optional(v.number()),
          message: v.optional(v.string()),
        }),
      ),
    }),
  ),
});

// Entity schemas for the Melovian music API. They mirror the interfaces in
// types.ts; keep them in sync when those change.

export const listenEntrySchema = v.looseObject({
  trackId: v.string(),
  trackTitle: v.string(),
  artistName: v.string(),
  albumId: v.string(),
  albumTitle: v.string(),
  positionMs: v.number(),
  durationMs: v.number(),
  played: v.boolean(),
  playCount: v.number(),
  listenedMs: v.number(),
  lastPlayedAt: v.string(),
  coverArtId: v.string(),
});

export const listenEventSchema = v.looseObject({
  id: v.number(),
  trackId: v.string(),
  trackTitle: v.string(),
  artistName: v.string(),
  albumId: v.string(),
  albumTitle: v.string(),
  durationMs: v.number(),
  coverArtId: v.string(),
  playedAt: v.string(),
});

const statEntrySchema = v.looseObject({
  key: v.string(),
  label: v.string(),
  count: v.number(),
});

export const listenStatsSchema = v.looseObject({
  totalPlays: v.number(),
  uniqueTracks: v.number(),
  totalListeningMs: v.number(),
  topArtists: v.array(statEntrySchema),
  topTracks: v.array(statEntrySchema),
  topAlbums: v.array(statEntrySchema),
});

export const musicStatusSchema = v.looseObject({
  enabled: v.boolean(),
  connected: v.boolean(),
  serverName: v.optional(v.string()),
  version: v.optional(v.string()),
  error: v.optional(v.string()),
  // The server also reports "unified", which the MusicStatus type predates.
  source: v.optional(v.string()),
});

export const libraryStatsSchema = v.looseObject({
  songCount: v.number(),
  albumCount: v.number(),
  artistCount: v.number(),
  folderCount: v.number(),
  scanning: v.boolean(),
  lastScan: v.optional(v.string()),
});

export const playlistTrackSchema = v.looseObject({
  trackId: v.string(),
  trackTitle: v.string(),
  artistName: v.string(),
  albumId: v.string(),
  albumTitle: v.string(),
  durationMs: v.number(),
  coverArtId: v.string(),
  position: v.optional(v.number()),
});

// The list endpoint serializes empty lists as null, so nullable fields
// normalize to undefined.
const nullToUndefined = <T>(value: T | null | undefined): T | undefined =>
  value ?? undefined;

export const musicPlaylistSchema = v.looseObject({
  id: v.string(),
  name: v.string(),
  kind: v.optional(v.string()),
  rulesJson: v.optional(v.string()),
  createdAt: v.string(),
  updatedAt: v.string(),
  trackCount: v.number(),
  durationMs: v.optional(v.number()),
  coverArtIds: v.optional(
    v.pipe(v.nullable(v.array(v.string())), v.transform(nullToUndefined)),
  ),
  tracks: v.optional(
    v.pipe(
      v.nullable(v.array(playlistTrackSchema)),
      v.transform(nullToUndefined),
    ),
  ),
});

export const favoriteTrackSchema = v.looseObject({
  trackId: v.string(),
  trackTitle: v.string(),
  artistName: v.string(),
  albumId: v.string(),
  albumTitle: v.string(),
  durationMs: v.number(),
  coverArtId: v.string(),
  favoritedAt: v.string(),
});
