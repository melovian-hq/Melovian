// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { getActiveInstanceId } from "$lib/features/instances/context";
import { resolveMediaUrl } from "$lib/config/runtime";
import { ApiPaths } from "$lib/core/http/api-paths";
import { isLocalMusicId } from "$lib/music/library-adapter";
import {
  isLocalMusicSource,
  isLocalTrackId,
  isUnifiedMusicSource,
} from "$lib/music/source.svelte";
import { localCoverArtUrl, localStreamUrl } from "$lib/local-music/api";
import type { SubsonicConfig } from "./types";

export function coverArtUrl(
  config: SubsonicConfig,
  id?: string,
  size = 300,
  trackId?: string,
): string | null {
  if (!id) return null;
  if (/^https?:\/\//i.test(id) || id.startsWith(ApiPaths.partyPrefix)) {
    return id.startsWith("/") ? resolveMediaUrl(id) : id;
  }
  if (isUnifiedMusicSource() && trackId && isLocalTrackId(trackId)) {
    return localCoverArtUrl(id, size);
  }
  if (
    isLocalMusicSource() ||
    (isUnifiedMusicSource() && id && isLocalMusicId(id))
  ) {
    return localCoverArtUrl(id, size);
  }
  const url = coverArtImageUrl(config, id, size);
  if (!url) return null;
  return resolveMediaUrl(url);
}

export function coverArtImageUrl(
  config: SubsonicConfig,
  id?: string,
  size = 300,
): string | null {
  if (!id) return null;
  const base = config.serverUrl.replace(/\/+$/, "");
  const params = subsonicMediaParams(config, { id, size: String(size) });
  return `${base}/rest/getCoverArt.view?${params.toString()}`;
}

export interface StreamOptions {
  maxBitRate?: number;
  format?: string;
}

export function streamUrl(
  config: SubsonicConfig,
  trackId: string,
  options?: StreamOptions,
): string {
  if (isUnifiedMusicSource() && isLocalTrackId(trackId)) {
    return localStreamUrl(trackId);
  }
  if (isLocalMusicSource()) {
    return localStreamUrl(trackId);
  }
  const base = resolveMediaUrl(config.serverUrl.replace(/\/+$/, ""));
  const params = subsonicMediaParams(config, { id: trackId });
  if (options?.maxBitRate !== undefined) {
    params.set("maxBitRate", String(options.maxBitRate));
  }
  if (options?.format) {
    params.set("format", options.format);
  }
  return `${base}/rest/stream.view?${params.toString()}`;
}

/**
 * downloadStreamUrl points at a track that has been cached on disk by the
 * backend, served from the embedded HTTP server with range support.
 */
export function downloadStreamUrl(trackId: string): string {
  const params = new URLSearchParams();
  const instanceId = getActiveInstanceId();
  if (instanceId) params.set("_instance", instanceId);
  const query = params.toString();
  const path = `${ApiPaths.downloadStream(trackId)}${query ? `?${query}` : ""}`;
  return resolveMediaUrl(path);
}

function subsonicMediaParams(
  config: SubsonicConfig,
  query: Record<string, string>,
): URLSearchParams {
  const params = new URLSearchParams({
    ...query,
    v: config.version,
    c: config.clientName,
  });
  const instanceId = getActiveInstanceId();
  if (instanceId) {
    params.set("_instance", instanceId);
  }
  return params;
}

export function formatDuration(seconds?: number): string {
  if (!seconds || seconds <= 0) return "--:--";
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins}:${secs.toString().padStart(2, "0")}`;
}

export function formatDurationMs(ms?: number): string {
  if (!ms || ms <= 0) return "--:--";
  return formatDuration(Math.floor(ms / 1000));
}
