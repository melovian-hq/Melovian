// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
import { ApiPaths } from "$lib/core/http/api-paths";
import { requireOk } from "$lib/core/http/errors";
import { parseJson } from "$lib/core/http/parse";
import * as v from "valibot";
import {
  metadataAlbumAutofixResultSchema,
  metadataBatchResponseSchema,
  metadataLookupResponseSchema,
  metadataSearchResultSchema,
  metadataSuggestionsResponseSchema,
  metadataSummarySchema,
  metadataTrackSchema,
} from "./schemas";
import type {
  MetadataBatchResult,
  MetadataFilenameSuggestion,
  MetadataIssueFilter,
  MetadataLookupMatch,
  MetadataSearchResult,
  MetadataSummary,
  MetadataTrack,
  MetadataTrackUpdate,
} from "./types";

async function metadataGet<TSchema extends v.GenericSchema>(
  schema: TSchema,
  path: string,
): Promise<v.InferOutput<TSchema>> {
  const response = await fetchWithRetry(path, { headers: apiHeaders() });
  await requireOk(response, `Metadata request failed: ${response.status}`);
  return parseJson(schema, response, path);
}

async function metadataPatch<TSchema extends v.GenericSchema>(
  schema: TSchema,
  path: string,
  body: unknown,
): Promise<v.InferOutput<TSchema>> {
  const response = await fetchWithRetry(path, {
    method: "PATCH",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Metadata update failed: ${response.status}`);
  }
  return parseJson(schema, response, path);
}

async function metadataPost<TSchema extends v.GenericSchema>(
  schema: TSchema,
  path: string,
  body: unknown,
): Promise<v.InferOutput<TSchema>> {
  const response = await fetchWithRetry(path, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Metadata request failed: ${response.status}`);
  }
  return parseJson(schema, response, path);
}

export async function fetchMetadataSummary(): Promise<MetadataSummary> {
  return metadataGet(metadataSummarySchema, ApiPaths.localMusicMetadataSummary);
}

export async function searchMetadataTracks(options: {
  q?: string;
  issue?: MetadataIssueFilter;
  limit?: number;
  offset?: number;
}): Promise<MetadataSearchResult> {
  const params = new URLSearchParams();
  if (options.q?.trim()) params.set("q", options.q.trim());
  if (options.issue) params.set("issue", options.issue);
  if (options.limit) params.set("limit", String(options.limit));
  if (options.offset) params.set("offset", String(options.offset));
  const query = params.toString();
  return metadataGet(
    metadataSearchResultSchema,
    `${ApiPaths.localMusicMetadataTracks}${query ? `?${query}` : ""}`,
  );
}

export async function fetchMetadataTrack(id: string): Promise<MetadataTrack> {
  return metadataGet(metadataTrackSchema, ApiPaths.localMusicMetadataTrack(id));
}

export async function updateMetadataTrack(
  id: string,
  update: MetadataTrackUpdate,
): Promise<MetadataTrack> {
  return metadataPatch(
    metadataTrackSchema,
    ApiPaths.localMusicMetadataTrack(id),
    update,
  );
}

export type MetadataLookupSource =
  "itunes" | "musicbrainz" | "deezer" | "theaudiodb" | "all";

export const metadataLookupSources: Array<{
  id: MetadataLookupSource;
  label: string;
}> = [
  { id: "itunes", label: "iTunes" },
  { id: "musicbrainz", label: "MusicBrainz" },
  { id: "deezer", label: "Deezer" },
  { id: "theaudiodb", label: "TheAudioDB" },
  { id: "all", label: "All providers" },
];

export function metadataLookupSourceLabel(source: string): string {
  return (
    metadataLookupSources.find((entry) => entry.id === source)?.label ?? source
  );
}

export async function lookupMetadataMatches(options: {
  q?: string;
  trackId?: string;
  limit?: number;
  source?: MetadataLookupSource;
}): Promise<MetadataLookupMatch[]> {
  const params = new URLSearchParams();
  if (options.q?.trim()) params.set("q", options.q.trim());
  if (options.trackId) params.set("trackId", options.trackId);
  if (options.limit) params.set("limit", String(options.limit));
  if (options.source) params.set("source", options.source);
  const payload = await metadataGet(
    metadataLookupResponseSchema,
    `${ApiPaths.localMusicMetadataLookup}?${params.toString()}`,
  );
  return payload.matches ?? [];
}

export async function applyMetadataAutofix(
  trackId: string,
  match: MetadataLookupMatch,
): Promise<MetadataTrack> {
  return metadataPost(
    metadataTrackSchema,
    ApiPaths.localMusicMetadataTrackAutofix(trackId),
    { match },
  );
}

export async function applyMetadataMatchToAlbum(
  trackId: string,
  match: MetadataLookupMatch,
): Promise<{ updated: number; failed: number; tracks: MetadataTrack[] }> {
  return metadataPost(
    metadataAlbumAutofixResultSchema,
    ApiPaths.localMusicMetadataTrackAutofixAlbum(trackId),
    { match },
  );
}

export async function fetchMetadataSuggestions(
  trackId: string,
): Promise<MetadataFilenameSuggestion> {
  const payload = await metadataGet(
    metadataSuggestionsResponseSchema,
    ApiPaths.localMusicMetadataTrackSuggestions(trackId),
  );
  return payload.filename;
}

export type MetadataBatchSource = "filename" | MetadataLookupSource;

export async function batchAutofixMetadata(options: {
  trackIds: string[];
  source?: MetadataBatchSource;
}): Promise<MetadataBatchResult[]> {
  const payload = await metadataPost(
    metadataBatchResponseSchema,
    ApiPaths.localMusicMetadataAutofixBatch,
    {
      trackIds: options.trackIds,
      source: options.source ?? "filename",
    },
  );
  return payload.results ?? [];
}
