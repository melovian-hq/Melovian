// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

// Local music responses carry raw entity objects that the map* helpers in
// api.ts normalize field by field. Schemas only pin the envelope shape.

const rawRecord = v.record(v.string(), v.unknown());
const rawList = v.array(rawRecord);

export const localArtistsResponseSchema = v.looseObject({
  artists: v.optional(v.nullable(rawList)),
});

export const localArtistResponseSchema = v.looseObject({
  artist: v.optional(rawRecord),
  albums: v.optional(v.nullable(rawList)),
});

export const localAlbumResponseSchema = v.looseObject({
  album: v.optional(rawRecord),
  songs: v.optional(v.nullable(rawList)),
});

export const localAlbumsResponseSchema = v.looseObject({
  albums: v.optional(v.nullable(rawList)),
});

export const localSearchResponseSchema = v.looseObject({
  artists: v.optional(v.nullable(rawList)),
  albums: v.optional(v.nullable(rawList)),
  songs: v.optional(v.nullable(rawList)),
});

export const localSongResponseSchema = v.looseObject({});

export const localSongsResponseSchema = v.looseObject({
  songs: v.optional(v.nullable(rawList)),
});

export const localGenresResponseSchema = v.looseObject({
  genres: v.optional(v.nullable(rawList)),
});

export const localStarredResponseSchema = v.looseObject({
  songs: v.optional(v.nullable(rawList)),
  albums: v.optional(v.nullable(rawList)),
  artists: v.optional(v.nullable(rawList)),
});
