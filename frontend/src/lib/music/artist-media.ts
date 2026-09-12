// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/**
 * Artist media helpers. Resolves Subsonic/Navidrome artist portrait URLs
 * for cards and hero art, and caches artist info lookups in memory and
 * localStorage.
 */

import { StorageKeys } from "$lib/brand";
import { createDetailCache, DETAIL_CACHE_TTL_MS } from "$lib/core/detail-cache";
import { ApiPaths } from "$lib/core/http/api-paths";
import { getActiveInstanceId } from "$lib/features/instances/context";
import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import { isLocalMusicId } from "$lib/music/library-adapter";
import { coverArtUrl } from "$lib/subsonic/urls";
import type {
  SubsonicArtist,
  SubsonicArtistInfo,
  SubsonicConfig,
} from "$lib/subsonic/types";

function isLoopbackHostname(hostname: string): boolean {
  const host = hostname.toLowerCase();
  return (
    host === "localhost" ||
    host === "127.0.0.1" ||
    host === "::1" ||
    host === "[::1]"
  );
}

/**
 * browserFacingProxyBase returns a path the browser can load via melovian.
 * Never return a loopback absolute ND URL for browser img src.
 */
function browserFacingProxyBase(serverUrl: string): string {
  const trimmed =
    serverUrl.trim().replace(/\/+$/, "") || ApiPaths.subsonicPrefix;
  if (trimmed.startsWith("/")) return trimmed;
  try {
    const absolute = new URL(trimmed);
    if (isLoopbackHostname(absolute.hostname)) return ApiPaths.subsonicPrefix;
    if (absolute.pathname.startsWith(ApiPaths.subsonicPrefix)) {
      return absolute.pathname.replace(/\/+$/, "") || ApiPaths.subsonicPrefix;
    }
  } catch {
    /* ignore */
  }
  return ApiPaths.subsonicPrefix;
}

function withInstanceQuery(pathWithQuery: string): string {
  const instanceId = getActiveInstanceId();
  if (!instanceId) return pathWithQuery;
  const joiner = pathWithQuery.includes("?") ? "&" : "?";
  return `${pathWithQuery}${joiner}_instance=${encodeURIComponent(instanceId)}`;
}

function isBrowserUnreachableAbsoluteUrl(url: string): boolean {
  try {
    if (!/^https?:\/\//i.test(url)) return false;
    return isLoopbackHostname(new URL(url).hostname);
  } catch {
    return false;
  }
}

/**
 * normalizeExternalMediaUrl sends Navidrome share/rest media through the
 * melovian Subsonic proxy when the URL is relative or points at loopback.
 * Remote browsers cannot load http://localhost:4533 directly.
 */
export function normalizeExternalMediaUrl(
  config: Pick<SubsonicConfig, "serverUrl">,
  raw: string,
): string {
  const trimmed = raw.trim();
  if (!trimmed) return trimmed;

  let pathAndQuery: string;
  try {
    if (/^https?:\/\//i.test(trimmed)) {
      const absolute = new URL(trimmed);
      if (!isLoopbackHostname(absolute.hostname)) return trimmed;
      pathAndQuery = `${absolute.pathname}${absolute.search}`;
    } else if (trimmed.startsWith("/")) {
      pathAndQuery = trimmed;
    } else {
      return trimmed;
    }
  } catch {
    return trimmed;
  }

  const proxyBase = browserFacingProxyBase(config.serverUrl);
  return withInstanceQuery(`${proxyBase}${pathAndQuery}`);
}

export function resolveServerArtistArtUrl(
  config: SubsonicConfig,
  artist: Pick<SubsonicArtist, "coverArt" | "artistImageUrl">,
  size: number,
  resolveMedia: (url: string) => string,
): string | null {
  const imageUrl = artist.artistImageUrl?.trim();
  if (imageUrl) {
    const normalized = normalizeExternalMediaUrl(config, imageUrl);
    if (!isBrowserUnreachableAbsoluteUrl(normalized)) {
      return resolveMedia(normalized);
    }
  }
  if (artist.coverArt) return coverArtUrl(config, artist.coverArt, size);
  return null;
}

export function hasServerArtistArt(
  artist: Pick<SubsonicArtist, "coverArt" | "artistImageUrl">,
): boolean {
  return Boolean(artist.artistImageUrl?.trim() || artist.coverArt?.trim());
}

const ARTIST_INFO_CACHE_TTL_MS = 30 * 60 * 1000;
const ARTIST_INFO_STORAGE_PREFIX = StorageKeys.artistInfoPrefix;
const STORAGE_TTL_MS = 24 * 60 * 60 * 1000;

const infoCache = createDetailCache<SubsonicArtistInfo>(
  ARTIST_INFO_CACHE_TTL_MS,
);
const inflight = new Map<string, Promise<SubsonicArtistInfo>>();

interface StoredArtistInfo {
  at: number;
  value: SubsonicArtistInfo;
}

interface PersistedArtistInfo {
  value: SubsonicArtistInfo;
  stale: boolean;
}

function readPersistentArtistInfo(id: string): PersistedArtistInfo | null {
  if (typeof localStorage === "undefined") return null;
  try {
    const raw = localStorage.getItem(`${ARTIST_INFO_STORAGE_PREFIX}${id}`);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as StoredArtistInfo;
    if (Date.now() - parsed.at > STORAGE_TTL_MS) {
      localStorage.removeItem(`${ARTIST_INFO_STORAGE_PREFIX}${id}`);
      return null;
    }
    return {
      value: parsed.value,
      stale: Date.now() - parsed.at > ARTIST_INFO_CACHE_TTL_MS,
    };
  } catch {
    return null;
  }
}

function writePersistentArtistInfo(
  id: string,
  value: SubsonicArtistInfo,
): void {
  if (typeof localStorage === "undefined") return;
  try {
    const payload: StoredArtistInfo = { at: Date.now(), value };
    localStorage.setItem(
      `${ARTIST_INFO_STORAGE_PREFIX}${id}`,
      JSON.stringify(payload),
    );
  } catch {
    /* storage full or unavailable */
  }
}

function emptyArtistInfo(): SubsonicArtistInfo {
  return { similarArtists: [] };
}

export async function fetchArtistInfoWithCache(
  library: MusicLibraryAdapter,
  id: string,
  onStale?: (data: SubsonicArtistInfo) => void,
): Promise<SubsonicArtistInfo> {
  if (isLocalMusicId(id)) {
    return emptyArtistInfo();
  }

  const cached = infoCache.peek(id);
  if (cached && !cached.stale) return cached.value;
  if (cached?.stale) {
    onStale?.(cached.value);
    void revalidateArtistInfo(library, id);
    return cached.value;
  }

  const persisted = readPersistentArtistInfo(id);
  if (persisted) {
    infoCache.set(id, persisted.value);
    onStale?.(persisted.value);
    if (persisted.stale) {
      void revalidateArtistInfo(library, id);
    }
    return persisted.value;
  }

  const pending = inflight.get(id);
  if (pending) return pending;

  const promise = (async () => {
    const fresh = await library
      .getArtistInfo(id)
      .catch(() => emptyArtistInfo());
    infoCache.set(id, fresh);
    writePersistentArtistInfo(id, fresh);
    return fresh;
  })();

  inflight.set(id, promise);
  try {
    return await promise;
  } finally {
    inflight.delete(id);
  }
}

async function revalidateArtistInfo(
  library: MusicLibraryAdapter,
  id: string,
): Promise<void> {
  try {
    const fresh = await library.getArtistInfo(id);
    infoCache.set(id, fresh);
    writePersistentArtistInfo(id, fresh);
  } catch {
    /* keep stale cache */
  }
}

export function prefetchArtistInfo(
  library: MusicLibraryAdapter,
  id: string,
): void {
  if (isLocalMusicId(id)) return;
  const cached = infoCache.peek(id);
  if (cached && !cached.stale) return;
  void fetchArtistInfoWithCache(library, id);
}

export function clearArtistInfoMemoryCache(id?: string) {
  if (id) {
    infoCache.delete(id);
    inflight.delete(id);
    return;
  }
  infoCache.clear();
  inflight.clear();
}

export function invalidateArtistInfoCache(id?: string) {
  if (id) {
    infoCache.delete(id);
    inflight.delete(id);
    if (typeof localStorage !== "undefined") {
      localStorage.removeItem(`${ARTIST_INFO_STORAGE_PREFIX}${id}`);
    }
    return;
  }
  infoCache.clear();
  inflight.clear();
  if (typeof localStorage === "undefined") return;
  const keys: string[] = [];
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i);
    if (key?.startsWith(ARTIST_INFO_STORAGE_PREFIX)) keys.push(key);
  }
  for (const key of keys) localStorage.removeItem(key);
}

export function resetArtistInfoCaches() {
  invalidateArtistInfoCache();
}

export { ARTIST_INFO_CACHE_TTL_MS, DETAIL_CACHE_TTL_MS };
