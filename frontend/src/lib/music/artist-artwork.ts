// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/**
 * Resolve Subsonic/Navidrome artist portrait URLs for cards and hero art.
 */

import { getActiveInstanceId } from "$lib/features/instances/context";
import { coverArtUrl } from "$lib/subsonic/urls";
import type { SubsonicArtist, SubsonicConfig } from "$lib/subsonic/types";

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
  const trimmed = serverUrl.trim().replace(/\/+$/, "") || "/api/subsonic";
  if (trimmed.startsWith("/")) return trimmed;
  try {
    const absolute = new URL(trimmed);
    if (isLoopbackHostname(absolute.hostname)) return "/api/subsonic";
    if (absolute.pathname.startsWith("/api/subsonic")) {
      return absolute.pathname.replace(/\/+$/, "") || "/api/subsonic";
    }
  } catch {
    /* ignore */
  }
  return "/api/subsonic";
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
