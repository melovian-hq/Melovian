// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
import { requireOk } from "$lib/core/http/errors";
import { ApiPaths } from "$lib/core/http/api-paths";
import { localStreamUrl } from "$lib/local-music/api";
import type {
  LocalVideo,
  TrackVideoLink,
  VideoResolveResult,
  VideoSearchHit,
  VideoSearchProvider,
  VideoSettings,
  VideoSource,
} from "./ids";

async function videoGet<T>(path: string): Promise<T> {
  const response = await fetchWithRetry(path, { headers: apiHeaders() });
  await requireOk(response, `Video request failed: ${response.status}`);
  return (await response.json()) as T;
}

async function videoSend<T>(
  path: string,
  method: string,
  body?: unknown,
): Promise<T | null> {
  const response = await fetchWithRetry(path, {
    method,
    headers: {
      ...apiHeaders(),
      ...(body !== undefined ? { "Content-Type": "application/json" } : {}),
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  if (response.status === 204) return null;
  await requireOk(response, `Video request failed: ${response.status}`);
  if (
    response.status === 204 ||
    response.headers.get("content-length") === "0"
  ) {
    return null;
  }
  const text = await response.text();
  if (!text) return null;
  return JSON.parse(text) as T;
}

export async function listLocalVideos(): Promise<LocalVideo[]> {
  const payload = await videoGet<{ videos: LocalVideo[] }>(
    ApiPaths.localMusicVideos,
  );
  return payload.videos ?? [];
}

export async function getLocalVideo(id: string): Promise<LocalVideo> {
  return videoGet<LocalVideo>(ApiPaths.localMusicVideo(id));
}

export function localVideoStreamUrl(id: string): string {
  return localStreamUrl(id);
}

export async function getVideoSettings(): Promise<VideoSettings> {
  return videoGet<VideoSettings>(ApiPaths.videoSettings);
}

export async function saveVideoSettings(
  settings: VideoSettings,
): Promise<VideoSettings> {
  const result = await videoSend<VideoSettings>(
    ApiPaths.videoSettings,
    "PUT",
    settings,
  );
  return result ?? settings;
}

export async function searchVideos(
  query: string,
  provider?: VideoSearchProvider,
): Promise<{ results: VideoSearchHit[]; provider: VideoSearchProvider }> {
  const params = new URLSearchParams({ q: query });
  if (provider) params.set("provider", provider);
  const payload = await videoGet<{
    results: VideoSearchHit[];
    provider?: VideoSearchProvider;
  }>(`${ApiPaths.videoSearch}?${params.toString()}`);
  return {
    results: payload.results ?? [],
    provider: payload.provider === "youtube" ? "youtube" : "invidious",
  };
}

export async function resolveVideo(input: {
  source: VideoSource;
  id: string;
  title?: string;
}): Promise<VideoResolveResult> {
  const params = new URLSearchParams({
    source: input.source,
    id: input.id,
  });
  if (input.title) params.set("title", input.title);
  return videoGet<VideoResolveResult>(
    `${ApiPaths.videoResolve}?${params.toString()}`,
  );
}

export async function getTrackVideoLink(
  trackId: string,
): Promise<TrackVideoLink | null> {
  // Single attempt: missing links are the common case (404), not transient errors.
  const response = await fetchWithRetry(
    ApiPaths.videoLink(trackId),
    { headers: apiHeaders() },
    1,
  );
  if (response.status === 404) return null;
  await requireOk(response, `Video link request failed: ${response.status}`);
  return (await response.json()) as TrackVideoLink;
}

export async function saveTrackVideoLink(
  trackId: string,
  input: { source: VideoSource; videoId: string; title?: string },
): Promise<TrackVideoLink> {
  const result = await videoSend<TrackVideoLink>(
    ApiPaths.videoLink(trackId),
    "PUT",
    input,
  );
  if (!result) {
    throw new Error("Empty response saving video link");
  }
  return result;
}

export async function deleteTrackVideoLink(trackId: string): Promise<void> {
  await videoSend(ApiPaths.videoLink(trackId), "DELETE");
}
