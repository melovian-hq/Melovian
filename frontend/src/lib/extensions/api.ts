// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
import { ApiPaths } from "$lib/core/http/api-paths";
import { readAPIError } from "$lib/core/http/errors";
import { parseJson } from "$lib/core/http/parse";
import * as v from "valibot";
import { extensionsPayloadSchema, registryPayloadSchema } from "./schemas";
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
  /** Current configured values for declared settings. */
  settings?: Record<string, unknown>;
  /** Directory-linked development install. */
  dev?: boolean;
};

export type ExtensionsPayload = {
  items: ExtensionListItem[];
  manifests: ExtensionManifest[];
  dir: string;
};

async function parseExtensionsPayload(
  response: Response,
): Promise<ExtensionsPayload> {
  const payload = await parseJson(
    extensionsPayloadSchema,
    response,
    "extensions",
  );
  return {
    items: payload.items ?? [],
    manifests: payload.manifests ?? [],
    dir: payload.dir ?? "",
  };
}

export async function fetchExtensions(): Promise<ExtensionsPayload> {
  const response = await fetchWithRetry(ApiPaths.extensions, {
    headers: apiHeaders(),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  return parseExtensionsPayload(response);
}

export async function setExtensionEnabled(
  id: string,
  enabled: boolean,
): Promise<ExtensionsPayload> {
  const response = await fetchWithRetry(ApiPaths.extensionEnabled(id), {
    method: "PUT",
    headers: apiHeaders("application/json"),
    body: JSON.stringify({ enabled }),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  return parseExtensionsPayload(response);
}

export async function installExtension(file: File): Promise<ExtensionsPayload> {
  const body = new FormData();
  body.append("package", file, file.name);
  const response = await fetchWithRetry(ApiPaths.extensionsInstall, {
    method: "POST",
    headers: apiHeaders(),
    body,
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  return parseExtensionsPayload(response);
}

export async function uninstallExtension(
  id: string,
): Promise<ExtensionsPayload> {
  const response = await fetchWithRetry(ApiPaths.extensionById(id), {
    method: "DELETE",
    headers: apiHeaders(),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  return parseExtensionsPayload(response);
}

export async function reinstallExtension(
  id: string,
): Promise<ExtensionsPayload> {
  const response = await fetchWithRetry(ApiPaths.extensionReinstall(id), {
    method: "POST",
    headers: apiHeaders(),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  return parseExtensionsPayload(response);
}

export type RegistryChangelogEntry = {
  version: string;
  date?: string;
  notes?: string;
};

export type RegistryItem = {
  id: string;
  name: string;
  version: string;
  description?: string;
  author?: string;
  homepage?: string;
  license?: string;
  tags?: string[];
  risk?: string;
  externalUrls?: string[];
  permissions?: string[];
  minAppVersion?: string;
  requires?: string[];
  delisted?: { reason?: string; at?: string };
  versions?: {
    version: string;
    url?: string;
    releasedAt?: string;
    notes?: string;
  }[];
  iconUrl?: string;
  imageUrl?: string;
  packageUrl?: string;
  sha256?: string;
  bytes?: number;
  hasScript: boolean;
  hasWasm: boolean;
  styles?: number;
  appTheme?: boolean;
  trackRules?: number;
  playerHooks?: number;
  auditStatus?: string;
  auditWarnings?: string[];
  changelog?: RegistryChangelogEntry[];
  installed: boolean;
  installedVersion?: string;
  enabled: boolean;
  updateAvailable: boolean;
};

export type ExtensionRegistryPayload = {
  url: string;
  generatedAt: string;
  signed: boolean;
  custom: boolean;
  items: RegistryItem[];
};

export async function fetchExtensionRegistry(): Promise<ExtensionRegistryPayload> {
  const response = await fetchWithRetry(ApiPaths.extensionsRegistry, {
    headers: apiHeaders(),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  const payload = await parseJson(
    registryPayloadSchema,
    response,
    "extension registry",
  );
  return {
    url: payload.url ?? "",
    generatedAt: payload.generatedAt ?? "",
    signed: payload.signed ?? false,
    custom: payload.custom ?? false,
    items: payload.items ?? [],
  };
}

export async function installExtensionDir(
  path: string,
): Promise<ExtensionsPayload> {
  const response = await fetchWithRetry(ApiPaths.extensionsInstallDir, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify({ path }),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  return parseExtensionsPayload(response);
}

export async function saveRegistryConfig(
  url: string,
  keys: string[],
): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.extensionsRegistry, {
    method: "PUT",
    headers: apiHeaders("application/json"),
    body: JSON.stringify({ url, keys }),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
}

export async function clearRegistryConfig(): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.extensionsRegistry, {
    method: "DELETE",
    headers: apiHeaders(),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
}

const extensionSettingsPayloadSchema = v.looseObject({
  schema: v.optional(v.array(v.unknown())),
  settings: v.optional(v.record(v.string(), v.unknown())),
});

export async function saveExtensionSettings(
  id: string,
  settings: Record<string, unknown>,
): Promise<Record<string, unknown>> {
  const response = await fetchWithRetry(ApiPaths.extensionSettings(id), {
    method: "PUT",
    headers: apiHeaders("application/json"),
    body: JSON.stringify({ settings }),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  const payload = await parseJson(
    extensionSettingsPayloadSchema,
    response,
    "extension settings",
  );
  return payload.settings ?? {};
}

export async function installRemoteExtension(
  id: string,
  version?: string,
): Promise<ExtensionsPayload> {
  const response = await fetchWithRetry(ApiPaths.extensionsInstallRemote, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(version ? { id, version } : { id }),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  return parseExtensionsPayload(response);
}
