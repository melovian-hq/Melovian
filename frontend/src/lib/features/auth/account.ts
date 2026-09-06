// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { fetchWithRetry } from "$lib/core/http/client";
import { readAPIError } from "$lib/core/http/errors";
import type { AuthUser } from "./api";

export interface AuthSession {
  id: string;
  expiresAt: string;
  current: boolean;
}

export async function listSessions(): Promise<AuthSession[]> {
  const response = await fetchWithRetry("/api/auth/sessions");
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  const payload = (await response.json()) as { sessions: AuthSession[] };
  return payload.sessions ?? [];
}

export async function revokeSession(id: string): Promise<void> {
  const response = await fetchWithRetry(
    `/api/auth/sessions/${encodeURIComponent(id)}`,
    {
      method: "DELETE",
    },
  );
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
}

export async function changePassword(
  currentPassword: string,
  newPassword: string,
): Promise<void> {
  const response = await fetchWithRetry("/api/auth/change-password", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ currentPassword, newPassword }),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
}

export async function changeUsername(username: string): Promise<AuthUser> {
  const response = await fetchWithRetry("/api/auth/username", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username }),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  return (await response.json()) as AuthUser;
}

export async function deleteAccount(password: string): Promise<void> {
  const response = await fetchWithRetry("/api/auth/delete-account", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password }),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
}

export type { AuthUser };
