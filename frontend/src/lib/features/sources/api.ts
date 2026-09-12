// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { ApiPaths } from "$lib/core/http/api-paths";
import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
import { readAPIError } from "$lib/core/http/errors";
import { parseJson } from "$lib/core/http/parse";
import { sourceStatusSchema } from "./schemas";

export type SourceViewMode = "subsonic" | "local" | "unified";

export interface SourceStatus {
  mode: SourceViewMode;
  activeInstanceId: string;
  activeLocalId: string;
  multiLocalLibrary: boolean;
  unifiedAvailable: boolean;
}

export async function fetchSourceStatus(): Promise<SourceStatus> {
  const response = await fetchWithRetry(ApiPaths.sourcesStatus, {
    headers: apiHeaders(),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  return parseJson(sourceStatusSchema, response, "source status");
}

export async function setMultiLocalLibrary(
  enabled: boolean,
): Promise<SourceStatus> {
  const response = await fetchWithRetry(ApiPaths.sourcesMultiLocalLibrary, {
    method: "PUT",
    headers: apiHeaders("application/json"),
    body: JSON.stringify({ enabled }),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  return parseJson(sourceStatusSchema, response, "source status");
}

export async function setSourceViewMode(
  mode: SourceViewMode,
): Promise<SourceStatus> {
  const response = await fetchWithRetry(ApiPaths.sourcesViewMode, {
    method: "PUT",
    headers: apiHeaders("application/json"),
    body: JSON.stringify({ mode }),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  return parseJson(sourceStatusSchema, response, "source status");
}
