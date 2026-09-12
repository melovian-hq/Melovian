// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

export const metadataSummarySchema = v.looseObject({
  unknownArtist: v.number(),
  unknownAlbum: v.number(),
  missingTitle: v.number(),
  any: v.number(),
  totalTracks: v.number(),
});

export const metadataTrackSchema = v.looseObject({
  id: v.string(),
  title: v.string(),
  artist: v.string(),
  album: v.string(),
  albumArtist: v.string(),
  trackNum: v.number(),
  discNum: v.number(),
  year: v.number(),
  genre: v.string(),
  durationMs: v.number(),
  format: v.string(),
  relPath: v.string(),
  // The Go view marshals a nil slice as null when a track has no issues.
  issues: v.pipe(
    v.nullable(v.array(v.string())),
    v.transform((value) => value ?? []),
  ),
  coverArt: v.string(),
});

export const metadataSearchResultSchema = v.looseObject({
  tracks: v.array(metadataTrackSchema),
  total: v.number(),
});

export const metadataLookupMatchSchema = v.looseObject({
  id: v.string(),
  source: v.string(),
  title: v.string(),
  artist: v.string(),
  album: v.string(),
  albumArtist: v.string(),
  trackNum: v.number(),
  year: v.number(),
  genre: v.string(),
  artworkUrl: v.optional(v.string()),
});

export const metadataLookupResponseSchema = v.looseObject({
  matches: v.optional(v.nullable(v.array(metadataLookupMatchSchema))),
});

export const metadataFilenameSuggestionSchema = v.looseObject({
  source: v.string(),
  title: v.string(),
  artist: v.string(),
  album: v.string(),
  albumArtist: v.string(),
  trackNum: v.number(),
  discNum: v.number(),
});

export const metadataSuggestionsResponseSchema = v.looseObject({
  filename: metadataFilenameSuggestionSchema,
});

export const metadataAlbumAutofixResultSchema = v.looseObject({
  updated: v.number(),
  failed: v.number(),
  tracks: v.array(metadataTrackSchema),
});

export const metadataBatchResultSchema = v.looseObject({
  trackId: v.string(),
  ok: v.boolean(),
  error: v.optional(v.string()),
  track: v.optional(metadataTrackSchema),
});

export const metadataBatchResponseSchema = v.looseObject({
  results: v.optional(v.nullable(v.array(metadataBatchResultSchema))),
});
