// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { fetchWithRetry } from "$lib/core/http/client";
import { ApiPaths } from "$lib/core/http/api-paths";
import { readAPIError } from "$lib/core/http/errors";
import { parseJson } from "$lib/core/http/parse";
import { authSessionsResponseSchema, authUserSchema } from "./schemas";
import type { AuthUser } from "./api";

export interface AuthSession {
  id: string;
  expiresAt: string;
  current: boolean;
}

export async function listSessions(): Promise<AuthSession[]> {
  const response = await fetchWithRetry(ApiPaths.authSessions);
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  const payload = await parseJson(
    authSessionsResponseSchema,
    response,
    "auth sessions",
  );
  return payload.sessions ?? [];
}

export async function revokeSession(id: string): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.authSessionById(id), {
    method: "DELETE",
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
}

export async function changePassword(
  currentPassword: string,
  newPassword: string,
): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.authChangePassword, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ currentPassword, newPassword }),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
}

export async function changeUsername(username: string): Promise<AuthUser> {
  const response = await fetchWithRetry(ApiPaths.authUsername, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username }),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  return parseJson(authUserSchema, response, "auth user");
}

export async function deleteAccount(password: string): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.authDeleteAccount, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password }),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
}

export type { AuthUser };
