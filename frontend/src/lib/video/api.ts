// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
import { requireOk } from "$lib/core/http/errors";
import { ApiPaths } from "$lib/core/http/api-paths";
import { parseJson, parsePayload } from "$lib/core/http/parse";
import { localStreamUrl } from "$lib/local-music/api";
import * as v from "valibot";
import {
  localVideoSchema,
  localVideosResponseSchema,
  trackVideoLinkSchema,
  videoResolveResultSchema,
  videoSearchResponseSchema,
  videoSettingsSchema,
} from "./schemas";
import type {
  LocalVideo,
  TrackVideoLink,
  VideoResolveResult,
  VideoSearchHit,
  VideoSearchProvider,
  VideoSettings,
  VideoSource,
} from "./ids";

async function videoGet<TSchema extends v.GenericSchema>(
  schema: TSchema,
  path: string,
): Promise<v.InferOutput<TSchema>> {
  const response = await fetchWithRetry(path, { headers: apiHeaders() });
  await requireOk(response, `Video request failed: ${response.status}`);
  return parseJson(schema, response, path);
}

async function videoSend<TSchema extends v.GenericSchema>(
  schema: TSchema,
  path: string,
  method: string,
  body?: unknown,
): Promise<v.InferOutput<TSchema> | null> {
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
  return parsePayload(schema, JSON.parse(text), path);
}

export async function listLocalVideos(): Promise<LocalVideo[]> {
  const payload = await videoGet(
    localVideosResponseSchema,
    ApiPaths.localMusicVideos,
  );
  return payload.videos ?? [];
}

export async function getLocalVideo(id: string): Promise<LocalVideo> {
  return videoGet(localVideoSchema, ApiPaths.localMusicVideo(id));
}

export function localVideoStreamUrl(id: string): string {
  return localStreamUrl(id);
}

export async function getVideoSettings(): Promise<VideoSettings> {
  return videoGet(videoSettingsSchema, ApiPaths.videoSettings);
}

export async function saveVideoSettings(
  settings: VideoSettings,
): Promise<VideoSettings> {
  const result = await videoSend(
    videoSettingsSchema,
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
  const payload = await videoGet(
    videoSearchResponseSchema,
    `${ApiPaths.videoSearch}?${params.toString()}`,
  );
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
  return videoGet(
    videoResolveResultSchema,
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
  return parseJson(trackVideoLinkSchema, response, "video link");
}

export async function saveTrackVideoLink(
  trackId: string,
  input: { source: VideoSource; videoId: string; title?: string },
): Promise<TrackVideoLink> {
  const result = await videoSend(
    trackVideoLinkSchema,
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
  await videoSend(v.unknown(), ApiPaths.videoLink(trackId), "DELETE");
}
