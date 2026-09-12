// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

const videoSourceSchema = v.picklist(["local", "invidious", "youtube"]);
const videoSearchProviderSchema = v.picklist(["invidious", "youtube"]);

export const localVideoSchema = v.looseObject({
  id: v.string(),
  title: v.string(),
  artist: v.optional(v.string()),
  album: v.optional(v.string()),
  duration: v.optional(v.number()),
  format: v.string(),
  relPath: v.string(),
  mediaKind: v.string(),
});

export const localVideosResponseSchema = v.looseObject({
  videos: v.optional(v.nullable(v.array(localVideoSchema))),
});

export const videoSettingsSchema = v.looseObject({
  enabled: v.boolean(),
  searchProvider: videoSearchProviderSchema,
  invidiousBaseUrl: v.string(),
  youtubeApiKey: v.string(),
});

export const videoSearchHitSchema = v.looseObject({
  id: v.string(),
  title: v.string(),
  author: v.optional(v.string()),
  lengthSeconds: v.optional(v.number()),
  thumbnail: v.optional(v.string()),
});

export const videoSearchResponseSchema = v.looseObject({
  results: v.optional(v.nullable(v.array(videoSearchHitSchema))),
  provider: v.optional(v.nullable(v.string())),
});

export const videoResolveResultSchema = v.looseObject({
  source: videoSourceSchema,
  videoId: v.string(),
  title: v.optional(v.string()),
  author: v.optional(v.string()),
  embedUrl: v.optional(v.string()),
  playId: v.string(),
});

export const trackVideoLinkSchema = v.looseObject({
  trackId: v.string(),
  source: videoSourceSchema,
  videoId: v.string(),
  title: v.optional(v.string()),
  playId: v.string(),
});
