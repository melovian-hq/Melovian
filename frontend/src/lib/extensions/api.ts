// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
import { readAPIError } from "$lib/core/http/errors";
import type { ExtensionManifest } from "./types";

export type ExtensionListItem = {
  id: string;
  name: string;
  version: string;
  description?: string;
  author?: string;
  enabled: boolean;
  installed: boolean;
  bundled: boolean;
  hasScript: boolean;
  scriptSafe: boolean;
  hasWasm: boolean;
  iconUrl?: string;
  imageUrl?: string;
  /** Named app chrome theme when enabled. Example: "neon". */
  appTheme?: string;
  /** Folder mtime of the installed extension as an ISO/RFC3339 string. */
  installedAt?: string;
};

export type ExtensionsPayload = {
  items: ExtensionListItem[];
  manifests: ExtensionManifest[];
  dir: string;
};

export async function fetchExtensions(): Promise<ExtensionsPayload> {
  const response = await fetchWithRetry("/api/extensions", {
    headers: apiHeaders(),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  return response.json() as Promise<ExtensionsPayload>;
}

export async function setExtensionEnabled(
  id: string,
  enabled: boolean,
): Promise<ExtensionsPayload> {
  const response = await fetchWithRetry(`/api/extensions/${id}/enabled`, {
    method: "PUT",
    headers: apiHeaders("application/json"),
    body: JSON.stringify({ enabled }),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  return response.json() as Promise<ExtensionsPayload>;
}

export async function installExtension(file: File): Promise<ExtensionsPayload> {
  const body = new FormData();
  body.append("package", file, file.name);
  const response = await fetchWithRetry("/api/extensions/install", {
    method: "POST",
    headers: apiHeaders(),
    body,
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  return response.json() as Promise<ExtensionsPayload>;
}

export async function uninstallExtension(
  id: string,
): Promise<ExtensionsPayload> {
  const response = await fetchWithRetry(`/api/extensions/${id}`, {
    method: "DELETE",
    headers: apiHeaders(),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  return response.json() as Promise<ExtensionsPayload>;
}

export async function reinstallExtension(
  id: string,
): Promise<ExtensionsPayload> {
  const response = await fetchWithRetry(`/api/extensions/${id}/reinstall`, {
    method: "POST",
    headers: apiHeaders(),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  return response.json() as Promise<ExtensionsPayload>;
}
