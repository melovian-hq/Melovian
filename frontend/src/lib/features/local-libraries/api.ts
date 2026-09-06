// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { ApiPaths } from "$lib/core/http/api-paths";
import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
import type {
  LocalLibrary,
  LocalLibraryConfig,
  LocalLibraryInput,
} from "./types";

export async function listLocalLibraries(): Promise<LocalLibrary[]> {
  const response = await fetchWithRetry(ApiPaths.localLibraries, {
    headers: apiHeaders(),
  });
  if (!response.ok) throw new Error("Failed to load local libraries");
  const payload = (await response.json()) as { libraries: LocalLibrary[] };
  return payload.libraries ?? [];
}

export async function getActiveLocalLibrary(): Promise<LocalLibrary | null> {
  const response = await fetchWithRetry(ApiPaths.localLibrariesActive, {
    headers: apiHeaders(),
  });
  if (!response.ok) throw new Error("Failed to load active local library");
  const payload = (await response.json()) as Partial<LocalLibrary>;
  return payload.id ? (payload as LocalLibrary) : null;
}

export async function createLocalLibrary(
  input: LocalLibraryInput,
): Promise<LocalLibrary> {
  const response = await fetchWithRetry(ApiPaths.localLibraries, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(input),
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || "Failed to create local library");
  }
  return (await response.json()) as LocalLibrary;
}

export async function activateLocalLibrary(id: string): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.localLibraryActivate(id), {
    method: "POST",
    headers: apiHeaders(),
  });
  if (!response.ok) throw new Error("Failed to activate local library");
}

export async function scanLocalLibrary(id: string): Promise<LocalLibrary> {
  const response = await fetchWithRetry(ApiPaths.localLibraryScan(id), {
    method: "POST",
    headers: apiHeaders(),
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || "Failed to scan local library");
  }
  return (await response.json()) as LocalLibrary;
}

export async function deleteLocalLibrary(id: string): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.localLibraryById(id), {
    method: "DELETE",
    headers: apiHeaders(),
  });
  if (!response.ok) throw new Error("Failed to delete local library");
}

export async function fetchLocalLibraryConfig(): Promise<LocalLibraryConfig> {
  const response = await fetchWithRetry(ApiPaths.config, {
    headers: apiHeaders(),
  });
  if (!response.ok) {
    return { enabled: false, defaultPath: "", allowCustomPath: false };
  }
  const payload = (await response.json()) as {
    localLibrary?: Partial<LocalLibraryConfig>;
  };
  return {
    enabled: payload.localLibrary?.enabled ?? false,
    defaultPath: payload.localLibrary?.defaultPath ?? "",
    allowCustomPath: payload.localLibrary?.allowCustomPath ?? false,
  };
}
