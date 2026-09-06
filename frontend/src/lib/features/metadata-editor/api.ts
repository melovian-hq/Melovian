// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
import { requireOk } from "$lib/core/http/errors";
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

async function metadataGet<T>(path: string): Promise<T> {
  const response = await fetchWithRetry(path, { headers: apiHeaders() });
  await requireOk(response, `Metadata request failed: ${response.status}`);
  return (await response.json()) as T;
}

async function metadataPatch<T>(path: string, body: unknown): Promise<T> {
  const response = await fetchWithRetry(path, {
    method: "PATCH",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Metadata update failed: ${response.status}`);
  }
  return (await response.json()) as T;
}

async function metadataPost<T>(path: string, body: unknown): Promise<T> {
  const response = await fetchWithRetry(path, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Metadata request failed: ${response.status}`);
  }
  return (await response.json()) as T;
}

export async function fetchMetadataSummary(): Promise<MetadataSummary> {
  return metadataGet<MetadataSummary>("/api/local-music/metadata/summary");
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
  return metadataGet<MetadataSearchResult>(
    `/api/local-music/metadata/tracks${query ? `?${query}` : ""}`,
  );
}

export async function fetchMetadataTrack(id: string): Promise<MetadataTrack> {
  return metadataGet<MetadataTrack>(
    `/api/local-music/metadata/tracks/${encodeURIComponent(id)}`,
  );
}

export async function updateMetadataTrack(
  id: string,
  update: MetadataTrackUpdate,
): Promise<MetadataTrack> {
  return metadataPatch<MetadataTrack>(
    `/api/local-music/metadata/tracks/${encodeURIComponent(id)}`,
    update,
  );
}

export type MetadataLookupSource =
  | "itunes"
  | "musicbrainz"
  | "deezer"
  | "theaudiodb"
  | "all";

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
  const payload = await metadataGet<{ matches: MetadataLookupMatch[] }>(
    `/api/local-music/metadata/lookup?${params.toString()}`,
  );
  return payload.matches ?? [];
}

export async function applyMetadataAutofix(
  trackId: string,
  match: MetadataLookupMatch,
): Promise<MetadataTrack> {
  return metadataPost<MetadataTrack>(
    `/api/local-music/metadata/tracks/${encodeURIComponent(trackId)}/autofix`,
    { match },
  );
}

export async function applyMetadataMatchToAlbum(
  trackId: string,
  match: MetadataLookupMatch,
): Promise<{ updated: number; failed: number; tracks: MetadataTrack[] }> {
  return metadataPost(
    `/api/local-music/metadata/tracks/${encodeURIComponent(trackId)}/autofix-album`,
    { match },
  );
}

export async function fetchMetadataSuggestions(
  trackId: string,
): Promise<MetadataFilenameSuggestion> {
  const payload = await metadataGet<{ filename: MetadataFilenameSuggestion }>(
    `/api/local-music/metadata/tracks/${encodeURIComponent(trackId)}/suggestions`,
  );
  return payload.filename;
}

export type MetadataBatchSource = "filename" | MetadataLookupSource;

export async function batchAutofixMetadata(options: {
  trackIds: string[];
  source?: MetadataBatchSource;
}): Promise<MetadataBatchResult[]> {
  const payload = await metadataPost<{ results: MetadataBatchResult[] }>(
    "/api/local-music/metadata/autofix-batch",
    {
      trackIds: options.trackIds,
      source: options.source ?? "filename",
    },
  );
  return payload.results ?? [];
}
