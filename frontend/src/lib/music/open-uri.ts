// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { resolveMediaUrl } from "$lib/config/runtime";
import type { QueueTrack } from "$lib/subsonic/types";

const LOCAL_STREAM_RE =
  /\/api\/local-music\/tracks\/([^/?#]+)\/stream(?:[/?#]|$)/;
const DOWNLOAD_STREAM_RE = /\/api\/downloads\/([^/?#]+)\/stream(?:[/?#]|$)/;
const SUBSONIC_STREAM_RE = /\/rest\/stream\.view(?:[/?#]|$)/i;

export type ParsedOpenUri =
  | { kind: "trackId"; trackId: string }
  | { kind: "stream"; streamUrl: string; title: string };

export function absoluteMediaUri(uri: string): string {
  const trimmed = uri.trim();
  if (!trimmed) return "";
  if (/^https?:\/\//i.test(trimmed) || trimmed.startsWith("/")) {
    return trimmed.startsWith("/") ? resolveMediaUrl(trimmed) : trimmed;
  }
  return trimmed;
}

export function titleFromStreamUri(uri: string): string {
  try {
    const url = new URL(uri);
    const base = decodeURIComponent(url.pathname.split("/").pop() ?? "");
    return base || url.hostname || "External stream";
  } catch {
    return "External stream";
  }
}

export function parseOpenMediaUri(uri: string): ParsedOpenUri | null {
  const absolute = absoluteMediaUri(uri);
  if (!absolute) return null;

  const localMatch = absolute.match(LOCAL_STREAM_RE);
  if (localMatch?.[1]) {
    return { kind: "trackId", trackId: decodeURIComponent(localMatch[1]) };
  }

  const downloadMatch = absolute.match(DOWNLOAD_STREAM_RE);
  if (downloadMatch?.[1]) {
    return { kind: "trackId", trackId: decodeURIComponent(downloadMatch[1]) };
  }

  if (SUBSONIC_STREAM_RE.test(absolute)) {
    try {
      const url = new URL(absolute);
      const trackId = url.searchParams.get("id");
      if (trackId) return { kind: "trackId", trackId };
    } catch {
      // fall through
    }
  }

  if (/^https?:\/\//i.test(absolute)) {
    return {
      kind: "stream",
      streamUrl: absolute,
      title: titleFromStreamUri(absolute),
    };
  }

  return null;
}

export function trackFromOpenStream(uri: string, title: string): QueueTrack {
  return {
    id: `open:${encodeURIComponent(uri).slice(0, 120)}`,
    title,
    streamUrl: uri,
    isInternetRadio: true,
  };
}
