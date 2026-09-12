// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

// The bundled /demo/catalog.json fixture is exported from internal/democatalog.
// Fields use Go-style capitalized keys; entries stay loose.
const demoArtistSchema = v.looseObject({
  ID: v.string(),
  Name: v.string(),
  CoverArt: v.string(),
  AlbumIDs: v.array(v.string()),
  Biography: v.optional(v.string()),
  SimilarIDs: v.optional(v.array(v.string())),
});

const demoAlbumSchema = v.looseObject({
  ID: v.string(),
  Name: v.string(),
  ArtistID: v.string(),
  Artist: v.string(),
  Year: v.number(),
  Genre: v.string(),
  CoverArt: v.string(),
  SongIDs: v.array(v.string()),
  CreatedAt: v.string(),
});

const demoSongSchema = v.looseObject({
  ID: v.string(),
  Title: v.string(),
  AlbumID: v.string(),
  Album: v.string(),
  ArtistID: v.string(),
  Artist: v.string(),
  Track: v.number(),
  Duration: v.number(),
  Year: v.number(),
  Genre: v.string(),
  CoverArt: v.string(),
  BitRate: v.number(),
  Starred: v.boolean(),
  PlayCount: v.number(),
  ContentType: v.string(),
  Suffix: v.string(),
});

const demoPlaylistSchema = v.looseObject({
  ID: v.string(),
  Name: v.string(),
  Comment: v.string(),
  SongIDs: v.array(v.string()),
  Created: v.string(),
  Changed: v.string(),
  Public: v.boolean(),
  Owner: v.string(),
});

const demoGenreSchema = v.looseObject({
  Name: v.string(),
  SongCount: v.number(),
  AlbumCount: v.number(),
});

export const demoCatalogSchema = v.looseObject({
  artists: v.array(demoArtistSchema),
  albums: v.array(demoAlbumSchema),
  songs: v.array(demoSongSchema),
  playlists: v.array(demoPlaylistSchema),
  genres: v.array(demoGenreSchema),
});
