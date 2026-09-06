// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { ApiPaths } from "$lib/core/http/api-paths";
import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
import { readAPIError, requireOk } from "$lib/core/http/errors";
import type { InstanceInput, SubsonicInstance } from "./types";

export async function listInstances(): Promise<SubsonicInstance[]> {
  const response = await fetchWithRetry(ApiPaths.instances, {
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to load instances");
  const payload = (await response.json()) as { instances: SubsonicInstance[] };
  return payload.instances ?? [];
}

export async function getActiveInstance(): Promise<SubsonicInstance | null> {
  const response = await fetchWithRetry(ApiPaths.instancesActive, {
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to load active instance");
  const payload = (await response.json()) as Partial<SubsonicInstance>;
  return payload.id ? (payload as SubsonicInstance) : null;
}

export async function createInstance(
  input: InstanceInput,
): Promise<SubsonicInstance> {
  const response = await fetchWithRetry(ApiPaths.instances, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(input),
  });
  await requireOk(response, "Failed to create instance");
  return (await response.json()) as SubsonicInstance;
}

export async function testInstance(input: InstanceInput): Promise<string> {
  const response = await fetchWithRetry(ApiPaths.instancesTest, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(input),
  });
  await requireOk(response, "Connection test failed");
  const payload = (await response.json()) as { serverName?: string };
  return payload.serverName ?? "Connected";
}

export async function updateInstance(
  id: string,
  input: InstanceInput,
): Promise<SubsonicInstance> {
  const response = await fetchWithRetry(ApiPaths.instanceById(id), {
    method: "PUT",
    headers: apiHeaders("application/json"),
    body: JSON.stringify(input),
  });
  await requireOk(response, "Failed to update instance");
  return (await response.json()) as SubsonicInstance;
}

export interface InstancePing {
  online: boolean;
  latencyMs: number;
  serverName?: string;
  version?: string;
  songCount?: number;
  error?: string;
}

export async function pingInstance(id: string): Promise<InstancePing> {
  const response = await fetchWithRetry(ApiPaths.instancePing(id), {
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to ping instance");
  return (await response.json()) as InstancePing;
}

export async function activateInstance(id: string): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.instanceActivate(id), {
    method: "POST",
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to activate instance");
}

export async function deleteInstance(id: string): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.instanceById(id), {
    method: "DELETE",
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to delete instance");
}

export async function readErrorMessage(
  response: Response,
  fallback: string,
): Promise<string> {
  return readAPIError(response, fallback);
}
