// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";
import {
  favoriteTrackSchema,
  listenEntrySchema,
  listenEventSchema,
  musicPlaylistSchema,
} from "$lib/subsonic/schemas";

export const listenItemsResponseSchema = v.looseObject({
  items: v.optional(v.nullable(v.array(listenEntrySchema))),
});

export const listenEventsPageSchema = v.looseObject({
  items: v.optional(v.nullable(v.array(listenEventSchema))),
  hasMore: v.optional(v.boolean()),
});

export const listenEventYearsSchema = v.looseObject({
  years: v.optional(v.nullable(v.array(v.number()))),
});

export const listenProgressBatchSchema = v.record(
  v.string(),
  listenEntrySchema,
);

export const rockskySettingsSchema = v.looseObject({
  enabled: v.boolean(),
  hasToken: v.boolean(),
  token: v.string(),
});

export const listenBrainzSettingsSchema = v.looseObject({
  enabled: v.boolean(),
  hasToken: v.boolean(),
  token: v.string(),
  endpoint: v.string(),
});

export const lastFMSettingsSchema = v.looseObject({
  enabled: v.boolean(),
  hasToken: v.boolean(),
  apiKey: v.string(),
  apiSecret: v.string(),
  sessionKey: v.string(),
  endpoint: v.string(),
});

export const tokenTestResultSchema = v.looseObject({
  ok: v.boolean(),
  userName: v.optional(v.string()),
});

export const playlistsResponseSchema = v.looseObject({
  playlists: v.optional(v.nullable(v.array(musicPlaylistSchema))),
});

export const favoriteItemsResponseSchema = v.looseObject({
  items: v.optional(v.nullable(v.array(favoriteTrackSchema))),
});

export const cacheSettingsResponseSchema = v.looseObject({
  enabled: v.boolean(),
  limitBytes: v.number(),
  strategy: v.picklist(["playback", "most_listened", "playlists", "all"]),
  usedBytes: v.number(),
  trackCount: v.number(),
});

export const downloadDirResponseSchema = v.looseObject({
  path: v.optional(v.string()),
});

export const downloadedTrackSchema = v.looseObject({
  trackId: v.string(),
  size: v.number(),
  contentType: v.string(),
  trackTitle: v.string(),
  artistName: v.string(),
  createdAt: v.number(),
});

export const downloadsResponseSchema = v.looseObject({
  downloads: v.optional(v.nullable(v.array(downloadedTrackSchema))),
});

export const shareTrackSchema = v.looseObject({
  id: v.string(),
  title: v.string(),
  artist: v.string(),
  album: v.string(),
  albumId: v.optional(v.string()),
  durationMs: v.number(),
  coverArtId: v.optional(v.string()),
});

const musicShareEntries = {
  id: v.string(),
  token: v.string(),
  url: v.string(),
  resourceType: v.string(),
  resourceId: v.string(),
  description: v.string(),
  visitCount: v.number(),
  createdAt: v.string(),
  accessMode: v.picklist(["public", "password", "restricted"]),
  instanceId: v.optional(v.string()),
  usernames: v.optional(v.array(v.string())),
  expiresAt: v.optional(v.string()),
  title: v.optional(v.string()),
  tracks: v.optional(v.array(shareTrackSchema)),
  requiresPassword: v.optional(v.boolean()),
  requiresLogin: v.optional(v.boolean()),
};

export const musicShareSchema = v.looseObject(musicShareEntries);

export const shareItemsResponseSchema = v.looseObject({
  items: v.optional(v.nullable(v.array(musicShareSchema))),
});

/** The public share endpoint also returns { error } on denied access. */
export const publicShareResponseSchema = v.looseObject({
  ...musicShareEntries,
  error: v.optional(v.string()),
});

/** Access-denied share bodies only carry flags and an error message. */
export const shareDeniedResponseSchema = v.looseObject({
  requiresPassword: v.optional(v.boolean()),
  requiresLogin: v.optional(v.boolean()),
  error: v.optional(v.string()),
});

export const eqSettingsSchema = v.looseObject({
  enabled: v.optional(v.boolean()),
  presetId: v.optional(v.string()),
  bands: v.optional(v.unknown()),
  gains: v.optional(v.unknown()),
});

export const connectionSettingsPatchSchema = v.looseObject({
  autoReconnect: v.optional(v.boolean()),
  minDelayMs: v.optional(v.number()),
  maxDelayMs: v.optional(v.number()),
  backoffMultiplier: v.optional(v.number()),
  healthCheckIntervalMs: v.optional(v.number()),
  offlinePollMs: v.optional(v.number()),
  rememberHistory: v.optional(v.boolean()),
  maxHistoryEntries: v.optional(v.number()),
  selfHealIntervalMs: v.optional(v.number()),
  retryOnOnline: v.optional(v.boolean()),
});

const lyricsProviderSettingSchema = v.looseObject({
  id: v.string(),
  name: v.optional(v.string()),
  enabled: v.boolean(),
  custom: v.optional(v.boolean()),
  url: v.optional(v.string()),
});

const builtinLyricsProviderSchema = v.looseObject({
  id: v.string(),
  name: v.string(),
  builtin: v.boolean(),
  custom: v.optional(v.boolean()),
});

export const lyricsSettingsResponseSchema = v.looseObject({
  storageDir: v.string(),
  autoFetch: v.boolean(),
  providers: v.array(lyricsProviderSettingSchema),
  whisperUrl: v.optional(v.string()),
  resolvedStorageDir: v.string(),
  defaultStorageDir: v.string(),
  builtinProviders: v.array(builtinLyricsProviderSchema),
  whisperEnabled: v.optional(v.boolean()),
  trackCount: v.number(),
  usedBytes: v.number(),
});

export const smartPlaylistSupportSchema = v.looseObject({
  supported: v.boolean(),
  mode: v.optional(v.picklist(["navidrome", "client"])),
  reason: v.optional(v.string()),
});

export const createdSmartPlaylistBodySchema = v.looseObject({
  error: v.optional(v.string()),
  id: v.optional(v.string()),
  name: v.optional(v.string()),
});

export const partyInviteLinkSchema = v.looseObject({
  sessionId: v.string(),
  token: v.string(),
  url: v.string(),
});

// The user-invite endpoint returns sessionId/token plus the invited user.
export const partyInviteResultSchema = v.looseObject({
  sessionId: v.string(),
  token: v.string(),
  username: v.string(),
  userId: v.string(),
});

export const syncPlaybackSnapshotSchema = v.looseObject({
  trackId: v.optional(v.string()),
  trackTitle: v.optional(v.string()),
  artistName: v.optional(v.string()),
  coverArt: v.optional(v.string()),
  positionMs: v.number(),
  durationMs: v.optional(v.number()),
  paused: v.boolean(),
  queueIds: v.optional(v.array(v.string())),
  queueIndex: v.optional(v.number()),
  serverTime: v.optional(v.number()),
});

export const partyJoinResultSchema = v.looseObject({
  sessionId: v.string(),
  state: v.optional(v.nullable(syncPlaybackSnapshotSchema)),
});

export const partyMemberSchema = v.looseObject({
  deviceId: v.string(),
  userId: v.optional(v.string()),
  username: v.optional(v.string()),
  name: v.string(),
  isHost: v.boolean(),
});

export const partyStatusSchema = v.looseObject({
  sessionId: v.string(),
  hostId: v.string(),
  hostUserId: v.optional(v.string()),
  crossUser: v.boolean(),
  members: v.array(partyMemberSchema),
});

// iTunes Search API payload; only the fields the artwork lookup reads.
export const itunesSearchResponseSchema = v.looseObject({
  results: v.optional(
    v.nullable(
      v.array(
        v.looseObject({
          artistName: v.optional(v.string()),
          collectionName: v.optional(v.string()),
          trackName: v.optional(v.string()),
          artworkUrl100: v.optional(v.string()),
        }),
      ),
    ),
  ),
});
