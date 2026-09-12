// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { ApiPaths } from "$lib/core/http/api-paths";
import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
import { requireOk } from "$lib/core/http/errors";
import { parseJson, parsePayload } from "$lib/core/http/parse";
import {
  cacheSettingsResponseSchema,
  connectionSettingsPatchSchema,
  createdSmartPlaylistBodySchema,
  downloadDirResponseSchema,
  downloadsResponseSchema,
  eqSettingsSchema,
  favoriteItemsResponseSchema,
  lastFMSettingsSchema,
  listenBrainzSettingsSchema,
  listenEventsPageSchema,
  listenEventYearsSchema,
  listenItemsResponseSchema,
  listenProgressBatchSchema,
  lyricsSettingsResponseSchema,
  musicShareSchema,
  playlistsResponseSchema,
  publicShareResponseSchema,
  rockskySettingsSchema,
  shareDeniedResponseSchema,
  shareItemsResponseSchema,
  smartPlaylistSupportSchema,
  tokenTestResultSchema,
} from "$lib/music/schemas";
import {
  libraryStatsSchema,
  listenEntrySchema,
  listenStatsSchema,
  musicPlaylistSchema,
  musicStatusSchema,
} from "$lib/subsonic/schemas";
import { resolveMediaUrl } from "$lib/config/runtime";
import { isRemoteClient, resolveApiUrl } from "$lib/config/remote-server";
import { getActiveInstanceId } from "$lib/features/instances/context";
import { normalizeEqSettings, type EqSettings } from "$lib/music/eq";
import {
  defaultConnectionSettings,
  type ConnectionSettings,
} from "$lib/music/connection-settings";
import type { ParsedLyrics } from "$lib/music/lyrics";
import { normalizeParsedLyrics } from "$lib/music/lyrics";
import type { LyricsSettingsResponse } from "$lib/music/lyrics-settings";
import type {
  ListenEntry,
  ListenEvent,
  ListenStats,
  MusicPlaylist,
  MusicStatus,
  FavoriteTrack,
  LibraryStats,
  PlaylistTrack,
} from "$lib/subsonic/types";

export async function getMusicStatus(): Promise<MusicStatus> {
  const response = await fetchWithRetry(ApiPaths.musicStatus, {
    headers: apiHeaders(),
  });
  if (!response.ok) {
    return {
      enabled: false,
      connected: false,
      error: "Music service unavailable",
    };
  }
  const status = await parseJson(musicStatusSchema, response, "music status");
  // The server also reports "unified", which the MusicStatus union predates.
  return { ...status, source: status.source as MusicStatus["source"] };
}

export async function getLibraryStats(options?: {
  bypassCache?: boolean;
}): Promise<LibraryStats | null> {
  const query = options?.bypassCache ? `?_refresh=${Date.now()}` : "";
  try {
    const response = await fetchWithRetry(
      `${ApiPaths.musicLibraryStats}${query}`,
      {
        headers: apiHeaders(),
      },
    );
    if (!response.ok) return null;
    return await parseJson(libraryStatsSchema, response, "library stats");
  } catch {
    // Background library watch treats stats as best-effort.
    return null;
  }
}

export async function bustLibraryCache(): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.musicLibraryRefresh, {
    method: "POST",
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to refresh library cache");
}

export async function getListenHistory(limit = 50): Promise<ListenEntry[]> {
  const response = await fetchWithRetry(
    `${ApiPaths.musicHistory}?limit=${limit}`,
    {
      headers: apiHeaders(),
    },
  );
  await requireOk(response, "Failed to load listen history");
  const payload = await parseJson(
    listenItemsResponseSchema,
    response,
    "listen history",
  );
  return payload.items ?? [];
}

export interface ListenEventsPage {
  items: ListenEvent[];
  hasMore: boolean;
}

export interface ListenEventsQuery {
  limit?: number;
  offset?: number;
  period?: string;
  q?: string;
}

export async function getListenEvents(
  query: ListenEventsQuery = {},
): Promise<ListenEventsPage> {
  const params = new URLSearchParams();
  params.set("limit", String(query.limit ?? 100));
  params.set("offset", String(query.offset ?? 0));
  if (query.period && query.period !== "all") {
    params.set("period", query.period);
  }
  if (query.q?.trim()) {
    params.set("q", query.q.trim());
  }
  const response = await fetchWithRetry(
    `${ApiPaths.musicListenEvents}?${params.toString()}`,
    { headers: apiHeaders() },
  );
  await requireOk(response, "Failed to load listen history");
  const payload = await parseJson(
    listenEventsPageSchema,
    response,
    "listen events",
  );
  return {
    items: payload.items ?? [],
    hasMore: payload.hasMore ?? false,
  };
}

export async function getListenEventYears(): Promise<number[]> {
  const response = await fetchWithRetry(ApiPaths.musicListenEventYears, {
    headers: apiHeaders(),
  });
  if (!response.ok) return [];
  const payload = await parseJson(
    listenEventYearsSchema,
    response,
    "listen event years",
  );
  return payload.years ?? [];
}

export async function clearListenEvents(): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.musicListenEvents, {
    method: "DELETE",
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to clear listen history");
}

export async function getResumeTracks(limit = 18): Promise<ListenEntry[]> {
  const response = await fetchWithRetry(
    `${ApiPaths.musicResume}?limit=${limit}`,
    {
      headers: apiHeaders(),
    },
  );
  await requireOk(response, "Failed to load resume tracks");
  const payload = await parseJson(
    listenItemsResponseSchema,
    response,
    "resume tracks",
  );
  return payload.items ?? [];
}

export async function getListenStats(limit = 8): Promise<ListenStats> {
  const response = await fetchWithRetry(
    `${ApiPaths.musicStats}?limit=${limit}`,
    {
      headers: apiHeaders(),
    },
  );
  await requireOk(response, "Failed to load listen stats");
  return parseJson(listenStatsSchema, response, "listen stats");
}

export async function getListenProgressBatch(
  trackIds: string[],
): Promise<Map<string, ListenEntry>> {
  const ids = [...new Set(trackIds.map((id) => id.trim()).filter(Boolean))];
  if (ids.length === 0) return new Map();

  const response = await fetchWithRetry(
    `${ApiPaths.musicBatch}?ids=${encodeURIComponent(ids.join(","))}`,
    { headers: apiHeaders() },
  );
  await requireOk(response, "Failed to load listen progress batch");
  const payload = await parseJson(
    listenProgressBatchSchema,
    response,
    "listen progress batch",
  );
  return new Map(Object.entries(payload));
}

export interface ListenUpsertBody {
  positionMs: number;
  played?: boolean;
  incrementPlay?: boolean;
  deltaMs?: number;
  trackTitle?: string;
  artistName?: string;
  albumId?: string;
  albumTitle?: string;
  durationMs?: number;
  coverArtId?: string;
}

export async function saveListenProgress(
  trackId: string,
  body: ListenUpsertBody,
): Promise<ListenEntry | null> {
  const response = await fetchWithRetry(ApiPaths.musicItem(trackId), {
    method: "PUT",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(body),
  });
  await requireOk(response, "Failed to save listen progress");
  return parseJson(listenEntrySchema, response, "listen progress");
}

export async function markTrackPlayed(trackId: string): Promise<void> {
  await fetchWithRetry(ApiPaths.musicItemPlayed(trackId), {
    method: "POST",
    headers: apiHeaders(),
  });
}

export interface RockskySettings {
  enabled: boolean;
  hasToken: boolean;
  token: string;
}

export interface RockskyTrack {
  id: string;
  title: string;
  artist: string;
  album: string;
  duration: number;
}

export async function getRockskySettings(): Promise<RockskySettings | null> {
  const response = await fetchWithRetry(ApiPaths.musicSettingsRocksky, {
    headers: apiHeaders(),
  });
  if (!response.ok) return null;
  return parseJson(rockskySettingsSchema, response, "rocksky settings");
}

export async function saveRockskySettings(
  settings: Pick<RockskySettings, "token">,
): Promise<RockskySettings | null> {
  const response = await fetchWithRetry(ApiPaths.musicSettingsRocksky, {
    method: "PUT",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(settings),
  });
  if (!response.ok) return null;
  return parseJson(rockskySettingsSchema, response, "rocksky settings");
}

export async function testRockskyToken(
  token: string,
): Promise<{ ok: boolean; userName?: string } | null> {
  const response = await fetchWithRetry(ApiPaths.musicRockskyTest, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify({ token }),
  });
  if (!response.ok) return null;
  return parseJson(tokenTestResultSchema, response, "rocksky token test");
}

export async function rockskyNowPlaying(track: RockskyTrack): Promise<void> {
  await fetchWithRetry(ApiPaths.musicRockskyNowPlaying, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(track),
  });
}

export async function rockskyScrobble(track: RockskyTrack): Promise<void> {
  await fetchWithRetry(ApiPaths.musicRockskyScrobble, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(track),
  });
}

export interface ListenBrainzSettings {
  enabled: boolean;
  hasToken: boolean;
  token: string;
  endpoint: string;
}

export async function getListenBrainzSettings(): Promise<ListenBrainzSettings | null> {
  const response = await fetchWithRetry(ApiPaths.musicListenBrainzSettings, {
    headers: apiHeaders(),
  });
  if (!response.ok) return null;
  return parseJson(
    listenBrainzSettingsSchema,
    response,
    "listenbrainz settings",
  );
}

export async function saveListenBrainzSettings(
  settings: Pick<ListenBrainzSettings, "token" | "endpoint">,
): Promise<ListenBrainzSettings | null> {
  const response = await fetchWithRetry(ApiPaths.musicListenBrainzSettings, {
    method: "PUT",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(settings),
  });
  if (!response.ok) return null;
  return parseJson(
    listenBrainzSettingsSchema,
    response,
    "listenbrainz settings",
  );
}

export async function testListenBrainzToken(
  token: string,
  endpoint: string,
): Promise<{ ok: boolean; userName?: string } | null> {
  const response = await fetchWithRetry(ApiPaths.musicListenBrainzTest, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify({ token, endpoint }),
  });
  if (!response.ok) return null;
  return parseJson(tokenTestResultSchema, response, "listenbrainz token test");
}

export async function listenbrainzNowPlaying(
  track: RockskyTrack,
): Promise<void> {
  await fetchWithRetry(ApiPaths.musicListenBrainzNowPlaying, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(track),
  });
}

export async function listenbrainzScrobble(track: RockskyTrack): Promise<void> {
  await fetchWithRetry(ApiPaths.musicListenBrainzScrobble, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(track),
  });
}

export interface LastFMSettings {
  enabled: boolean;
  hasToken: boolean;
  apiKey: string;
  apiSecret: string;
  sessionKey: string;
  endpoint: string;
}

export async function getLastFMSettings(): Promise<LastFMSettings | null> {
  const response = await fetchWithRetry(ApiPaths.musicLastFMSettings, {
    headers: apiHeaders(),
  });
  if (!response.ok) return null;
  return parseJson(lastFMSettingsSchema, response, "lastfm settings");
}

export async function saveLastFMSettings(
  settings: Pick<
    LastFMSettings,
    "apiKey" | "apiSecret" | "sessionKey" | "endpoint"
  >,
): Promise<LastFMSettings | null> {
  const response = await fetchWithRetry(ApiPaths.musicLastFMSettings, {
    method: "PUT",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(settings),
  });
  if (!response.ok) return null;
  return parseJson(lastFMSettingsSchema, response, "lastfm settings");
}

export async function testLastFMToken(
  apiKey: string,
  apiSecret: string,
  sessionKey: string,
  endpoint: string,
): Promise<{ ok: boolean; userName?: string } | null> {
  const response = await fetchWithRetry(ApiPaths.musicLastFMTest, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify({ apiKey, apiSecret, sessionKey, endpoint }),
  });
  if (!response.ok) return null;
  return parseJson(tokenTestResultSchema, response, "lastfm token test");
}

export async function lastfmNowPlaying(track: RockskyTrack): Promise<void> {
  await fetchWithRetry(ApiPaths.musicLastFMNowPlaying, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(track),
  });
}

export async function lastfmScrobble(track: RockskyTrack): Promise<void> {
  await fetchWithRetry(ApiPaths.musicLastFMScrobble, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(track),
  });
}

export async function listPlaylists(): Promise<MusicPlaylist[]> {
  const response = await fetchWithRetry(ApiPaths.musicPlaylists, {
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to load playlists");
  const payload = await parseJson(
    playlistsResponseSchema,
    response,
    "playlists",
  );
  return payload.playlists ?? [];
}

export async function getPlaylist(id: string): Promise<MusicPlaylist> {
  const response = await fetchWithRetry(ApiPaths.musicPlaylist(id), {
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to load playlist");
  return parseJson(musicPlaylistSchema, response, "playlist");
}

export async function createPlaylist(
  name: string,
  opts?: { kind?: "static" | "smart"; rulesJson?: string },
): Promise<MusicPlaylist> {
  const response = await fetchWithRetry(ApiPaths.musicPlaylists, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify({
      name,
      kind: opts?.kind,
      rulesJson: opts?.rulesJson,
    }),
  });
  await requireOk(response, "Failed to create playlist");
  return parseJson(musicPlaylistSchema, response, "playlist");
}

export async function setPlaylistTracks(
  playlistId: string,
  tracks: PlaylistTrack[],
): Promise<MusicPlaylist> {
  const response = await fetchWithRetry(
    ApiPaths.musicPlaylistTracks(playlistId),
    {
      method: "PUT",
      headers: apiHeaders("application/json"),
      body: JSON.stringify({ tracks }),
    },
  );
  await requireOk(response, "Failed to update playlist tracks");
  return parseJson(musicPlaylistSchema, response, "playlist");
}

export async function renamePlaylist(
  id: string,
  name: string,
): Promise<MusicPlaylist> {
  const response = await fetchWithRetry(ApiPaths.musicPlaylist(id), {
    method: "PATCH",
    headers: apiHeaders("application/json"),
    body: JSON.stringify({ name }),
  });
  await requireOk(response, "Failed to rename playlist");
  return parseJson(musicPlaylistSchema, response, "playlist");
}

export async function deletePlaylist(id: string): Promise<void> {
  await fetchWithRetry(ApiPaths.musicPlaylist(id), {
    method: "DELETE",
    headers: apiHeaders(),
  });
}

export async function addTrackToPlaylist(
  playlistId: string,
  track: PlaylistTrack,
): Promise<MusicPlaylist> {
  const response = await fetchWithRetry(
    ApiPaths.musicPlaylistTracks(playlistId),
    {
      method: "POST",
      headers: apiHeaders("application/json"),
      body: JSON.stringify(track),
    },
  );
  await requireOk(response, "Failed to add track");
  return parseJson(musicPlaylistSchema, response, "playlist");
}

export async function removeTrackFromPlaylist(
  playlistId: string,
  trackId: string,
): Promise<MusicPlaylist> {
  const response = await fetchWithRetry(
    ApiPaths.musicPlaylistTrack(playlistId, trackId),
    {
      method: "DELETE",
      headers: apiHeaders(),
    },
  );
  await requireOk(response, "Failed to remove track");
  return parseJson(musicPlaylistSchema, response, "playlist");
}

export async function listFavorites(limit = 200): Promise<FavoriteTrack[]> {
  const response = await fetchWithRetry(
    `${ApiPaths.musicFavorites}?limit=${limit}`,
    {
      headers: apiHeaders(),
    },
  );
  await requireOk(response, "Failed to load favorites");
  const payload = await parseJson(
    favoriteItemsResponseSchema,
    response,
    "favorites",
  );
  return payload.items ?? [];
}

export async function addFavorite(
  trackId: string,
  body: Omit<FavoriteTrack, "trackId" | "favoritedAt">,
): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.musicFavorite(trackId), {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(body),
  });
  await requireOk(response, "Failed to favorite track");
}

export async function removeFavorite(trackId: string): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.musicFavorite(trackId), {
    method: "DELETE",
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to remove favorite");
}

export interface DownloadedTrackInfo {
  trackId: string;
  size: number;
  contentType: string;
  trackTitle: string;
  artistName: string;
  createdAt: number;
}

import type { CacheStrategy } from "$lib/music/cache-settings";

export interface CacheSettingsResponse {
  enabled: boolean;
  limitBytes: number;
  strategy: CacheStrategy;
  usedBytes: number;
  trackCount: number;
}

export async function getCacheSettings(): Promise<CacheSettingsResponse | null> {
  const response = await fetchWithRetry(ApiPaths.musicSettingsCache, {
    headers: apiHeaders(),
  });
  if (!response.ok) return null;
  return parseJson(cacheSettingsResponseSchema, response, "cache settings");
}

export async function saveCacheSettingsRemote(
  settings: Pick<CacheSettingsResponse, "enabled" | "limitBytes" | "strategy">,
): Promise<CacheSettingsResponse> {
  const response = await fetchWithRetry(ApiPaths.musicSettingsCache, {
    method: "PUT",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(settings),
  });
  await requireOk(response, "Failed to save cache settings");
  return parseJson(cacheSettingsResponseSchema, response, "cache settings");
}

export async function clearDownloadCache(): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.downloads, {
    method: "DELETE",
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to clear download cache");
}

export async function getDownloadDir(): Promise<string | null> {
  const response = await fetchWithRetry(ApiPaths.downloadsDir, {
    headers: apiHeaders(),
  });
  if (!response.ok) return null;
  const payload = await parseJson(
    downloadDirResponseSchema,
    response,
    "download dir",
  );
  return payload.path?.trim() || null;
}

export async function revealDownloadDir(): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.downloadsReveal, {
    method: "POST",
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to open downloads folder");
}

export async function listDownloads(): Promise<DownloadedTrackInfo[]> {
  const response = await fetchWithRetry(ApiPaths.downloads, {
    headers: apiHeaders(),
  });
  if (!response.ok) return [];
  const payload = await parseJson(
    downloadsResponseSchema,
    response,
    "downloads",
  );
  return payload.downloads ?? [];
}

export async function downloadTrack(
  trackId: string,
  meta: { title?: string; artist?: string } = {},
  signal?: AbortSignal,
): Promise<void> {
  const params = new URLSearchParams();
  if (meta.title) params.set("title", meta.title);
  if (meta.artist) params.set("artist", meta.artist);
  const query = params.toString();
  const response = await fetchWithRetry(
    `${ApiPaths.download(trackId)}${query ? `?${query}` : ""}`,
    {
      method: "POST",
      headers: apiHeaders(),
      signal,
    },
  );
  await requireOk(response, "Failed to download track");
}

export async function deleteDownload(trackId: string): Promise<void> {
  await fetchWithRetry(ApiPaths.download(trackId), {
    method: "DELETE",
    headers: apiHeaders(),
  });
}

function downloadApiPath(path: string): string {
  const params = new URLSearchParams();
  const instanceId = getActiveInstanceId();
  if (instanceId) params.set("_instance", instanceId);
  const query = params.toString();
  return resolveMediaUrl(`${path}${query ? `?${query}` : ""}`);
}

export function downloadExportUrl(trackId: string): string {
  return downloadApiPath(ApiPaths.downloadExport(trackId));
}

export function downloadExportAllUrl(): string {
  return downloadApiPath(ApiPaths.downloadsExportZip);
}

export type ShareAccessMode = "public" | "password" | "restricted";

export interface MusicShare {
  id: string;
  token: string;
  url: string;
  resourceType: string;
  resourceId: string;
  description: string;
  visitCount: number;
  createdAt: string;
  accessMode: ShareAccessMode;
  instanceId?: string;
  usernames?: string[];
  expiresAt?: string;
  title?: string;
  tracks?: ShareTrack[];
  requiresPassword?: boolean;
  requiresLogin?: boolean;
}

export interface ShareTrack {
  id: string;
  title: string;
  artist: string;
  album: string;
  albumId?: string;
  durationMs: number;
  coverArtId?: string;
}

export interface CreateShareInput {
  resourceType: string;
  resourceId: string;
  description?: string;
  expiresInSec?: number;
  accessMode?: ShareAccessMode;
  password?: string;
  usernames?: string[];
  instanceId?: string;
}

export async function listShares(): Promise<MusicShare[]> {
  const response = await fetchWithRetry(ApiPaths.musicShares, {
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to list shares");
  const payload = await parseJson(shareItemsResponseSchema, response, "shares");
  return payload.items ?? [];
}

export async function listShareInbox(): Promise<MusicShare[]> {
  const response = await fetchWithRetry(ApiPaths.musicSharesInbox, {
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to load shared playlists");
  const payload = await parseJson(
    shareItemsResponseSchema,
    response,
    "share inbox",
  );
  return payload.items ?? [];
}

export async function createShare(
  input: CreateShareInput,
): Promise<MusicShare> {
  const response = await fetchWithRetry(ApiPaths.musicShares, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(input),
  });
  await requireOk(response, "Failed to create share");
  return parseJson(musicShareSchema, response, "share");
}

export async function deleteShare(id: string): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.musicShare(id), {
    method: "DELETE",
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to delete share");
}

export async function getPublicShare(token: string): Promise<MusicShare> {
  const response = await fetch(
    resolveApiUrl(`/s/${encodeURIComponent(token)}`),
    {
      credentials: isRemoteClient() ? "include" : "same-origin",
    },
  );
  const raw: unknown = await response.json();
  if (!response.ok) {
    const denied = parsePayload(shareDeniedResponseSchema, raw, "public share");
    if (denied.requiresPassword || denied.requiresLogin) {
      // Gated responses only carry the access flags and an error message.
      return raw as MusicShare;
    }
    throw new Error(denied.error ?? `Share unavailable (${response.status})`);
  }
  return parsePayload(publicShareResponseSchema, raw, "public share");
}

export async function unlockPublicShare(
  token: string,
  password: string,
): Promise<void> {
  const response = await fetch(
    resolveApiUrl(`/s/${encodeURIComponent(token)}/unlock`),
    {
      method: "POST",
      credentials: isRemoteClient() ? "include" : "same-origin",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ password }),
    },
  );
  await requireOk(response, "Invalid password");
}

export function publicShareStreamUrl(token: string, trackId: string): string {
  return resolveMediaUrl(
    `/s/${encodeURIComponent(token)}/tracks/${encodeURIComponent(trackId)}/stream`,
  );
}

export function publicShareDownloadUrl(token: string, trackId: string): string {
  return resolveMediaUrl(
    `/s/${encodeURIComponent(token)}/tracks/${encodeURIComponent(trackId)}/download`,
  );
}

export function mediaTrackDownloadUrl(
  trackId: string,
  options?: { title?: string; artist?: string },
): string {
  const params = new URLSearchParams();
  const instanceId = getActiveInstanceId();
  if (instanceId) params.set("_instance", instanceId);
  if (options?.title) params.set("title", options.title);
  if (options?.artist) params.set("artist", options.artist);
  const query = params.toString();
  return resolveMediaUrl(
    `${ApiPaths.mediaTrackDownload(trackId)}${query ? `?${query}` : ""}`,
  );
}

export function mediaAlbumDownloadZipUrl(albumId: string): string {
  return downloadApiPath(ApiPaths.mediaAlbumDownloadZip(albumId));
}

export function mediaPlaylistDownloadZipUrl(playlistId: string): string {
  return downloadApiPath(ApiPaths.mediaPlaylistDownloadZip(playlistId));
}

export function mediaServerPlaylistDownloadZipUrl(playlistId: string): string {
  return downloadApiPath(ApiPaths.mediaServerPlaylistDownloadZip(playlistId));
}

export async function getEqSettings(): Promise<EqSettings | null> {
  const response = await fetchWithRetry(ApiPaths.musicSettingsEq, {
    headers: apiHeaders(),
  });
  if (!response.ok) return null;
  const payload = await parseJson(eqSettingsSchema, response, "eq settings");
  if (!Array.isArray(payload.bands)) return null;
  return normalizeEqSettings(payload);
}

export async function saveEqSettings(settings: EqSettings): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.musicSettingsEq, {
    method: "PUT",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(settings),
  });
  await requireOk(response, "Failed to save EQ settings");
}

export async function getConnectionSettings(): Promise<ConnectionSettings | null> {
  const response = await fetchWithRetry(ApiPaths.musicSettingsConnection, {
    headers: apiHeaders(),
  });
  if (!response.ok) return null;
  const payload = await parseJson(
    connectionSettingsPatchSchema,
    response,
    "connection settings",
  );
  if (Object.keys(payload).length === 0) return null;
  return { ...defaultConnectionSettings(), ...payload };
}

export async function saveConnectionSettingsRemote(
  settings: ConnectionSettings,
): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.musicSettingsConnection, {
    method: "PUT",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(settings),
  });
  await requireOk(response, "Failed to save connection settings");
}

function lyricsQuery(meta: {
  artist?: string;
  title?: string;
  album?: string;
  durationSec?: number;
}) {
  const params = new URLSearchParams();
  if (meta.artist) params.set("artist", meta.artist);
  if (meta.title) params.set("title", meta.title);
  if (meta.album) params.set("album", meta.album);
  if (meta.durationSec && meta.durationSec > 0) {
    params.set("durationSec", String(meta.durationSec));
  }
  const query = params.toString();
  return query ? `?${query}` : "";
}

const LYRICS_REQUEST_TIMEOUT_MS = 50_000;

async function lyricsRequest(
  input: string,
  init: RequestInit = {},
): Promise<Response> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), LYRICS_REQUEST_TIMEOUT_MS);
  const crossOrigin = isRemoteClient();
  try {
    // Resolve remote base URL, but do not use fetchWithRetry: that helper
    // retries 404s, which is the normal "no lyrics" response.
    return await fetch(resolveApiUrl(input), {
      credentials:
        init.credentials ?? (crossOrigin ? "include" : "same-origin"),
      ...init,
      signal: controller.signal,
      headers: {
        ...apiHeaders(),
        ...(init.headers ?? {}),
      },
    });
  } finally {
    clearTimeout(timer);
  }
}

async function readLyricsResponse(
  response: Response,
): Promise<ParsedLyrics | null> {
  if (response.status === 404) return null;
  let payload: unknown;
  try {
    payload = await response.json();
  } catch {
    if (!response.ok) {
      throw new Error("Failed to read lyrics response");
    }
    return null;
  }
  if (!response.ok) {
    const message =
      payload &&
      typeof payload === "object" &&
      "message" in payload &&
      typeof (payload as { message?: string }).message === "string"
        ? (payload as { message: string }).message
        : "Lyrics request failed";
    throw new Error(message);
  }
  return normalizeParsedLyrics(payload as Partial<ParsedLyrics>);
}

export async function getTrackLyrics(
  trackId: string,
  meta: {
    artist?: string;
    title?: string;
    album?: string;
    durationSec?: number;
  } = {},
): Promise<ParsedLyrics | null> {
  const response = await lyricsRequest(
    `${ApiPaths.musicLyrics(trackId)}${lyricsQuery(meta)}`,
  );
  return readLyricsResponse(response);
}

export async function fetchTrackLyrics(
  trackId: string,
  meta: {
    artist?: string;
    title?: string;
    album?: string;
    durationSec?: number;
  } = {},
): Promise<ParsedLyrics | null> {
  const response = await lyricsRequest(
    `${ApiPaths.musicLyricsFetch(trackId)}${lyricsQuery(meta)}`,
    { method: "POST" },
  );
  return readLyricsResponse(response);
}

export async function getLyricsSettings(): Promise<LyricsSettingsResponse | null> {
  const response = await fetchWithRetry(ApiPaths.musicSettingsLyrics, {
    headers: apiHeaders(),
  });
  if (!response.ok) return null;
  return parseJson(lyricsSettingsResponseSchema, response, "lyrics settings");
}

export async function saveLyricsSettingsRemote(
  settings: LyricsSettingsResponse,
): Promise<LyricsSettingsResponse> {
  const response = await fetchWithRetry(ApiPaths.musicSettingsLyrics, {
    method: "PUT",
    headers: apiHeaders("application/json"),
    body: JSON.stringify({
      storageDir: settings.storageDir,
      autoFetch: settings.autoFetch,
      providers: settings.providers,
      whisperUrl: settings.whisperUrl,
    }),
  });
  await requireOk(response, "Failed to save lyrics settings");
  return parseJson(lyricsSettingsResponseSchema, response, "lyrics settings");
}

export async function clearLyricsCache(): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.musicLyricsCache, {
    method: "DELETE",
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to clear lyrics cache");
}

export interface SmartPlaylistSupport {
  supported: boolean;
  mode?: "navidrome" | "client";
  reason?: string;
}

export async function getSmartPlaylistSupport(): Promise<SmartPlaylistSupport> {
  const response = await fetchWithRetry(ApiPaths.musicSmartPlaylistsSupport, {
    headers: apiHeaders(),
  });
  if (!response.ok) {
    return {
      supported: false,
      reason: "Could not check smart playlist support",
    };
  }
  return parseJson(
    smartPlaylistSupportSchema,
    response,
    "smart playlist support",
  );
}

export interface CreatedSmartPlaylist {
  id: string;
  name: string;
}

export async function createSmartPlaylist(
  payload: import("$lib/music/smart-playlist/types").CreateSmartPlaylistPayload,
): Promise<CreatedSmartPlaylist> {
  const response = await fetchWithRetry(ApiPaths.musicSmartPlaylists, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(payload),
  });
  const body = parsePayload(
    createdSmartPlaylistBodySchema,
    await response.json().catch(() => ({})),
    "smart playlist",
  );
  if (!response.ok) {
    throw new Error(body.error ?? "Failed to create smart playlist");
  }
  if (!body.id || !body.name) {
    throw new Error("Server returned an invalid smart playlist response");
  }
  return { id: body.id, name: body.name };
}
