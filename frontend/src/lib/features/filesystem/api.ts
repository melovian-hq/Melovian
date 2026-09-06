// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
import { readAPIError } from "$lib/core/http/errors";

export type DirectoryEntry = {
  name: string;
  path: string;
};

export type DirectoryListing = {
  path: string;
  parent: string;
  entries: DirectoryEntry[];
};

export async function listDirectories(path = ""): Promise<DirectoryListing> {
  const query = path.trim() ? `?path=${encodeURIComponent(path.trim())}` : "";
  const response = await fetchWithRetry(`/api/filesystem/directories${query}`, {
    headers: apiHeaders(),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  const payload = (await response.json()) as DirectoryListing;
  return {
    path: payload.path ?? "",
    parent: payload.parent ?? "",
    entries: Array.isArray(payload.entries) ? payload.entries : [],
  };
}
