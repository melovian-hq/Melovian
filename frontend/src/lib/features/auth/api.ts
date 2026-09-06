// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { ApiPaths } from "$lib/core/http/api-paths";
import { fetchWithRetry } from "$lib/core/http/client";
import { readAPIError } from "$lib/core/http/errors";

export interface AuthUser {
  id: string;
  username: string;
}

export interface AuthStatus {
  enabled: boolean;
  authenticated: boolean;
  setupRequired: boolean;
  demoMode?: boolean;
  fakeCatalog?: boolean;
  oidcEnabled?: boolean;
  oidcLoginUrl?: string;
  localLoginEnabled?: boolean;
  user?: AuthUser;
}

export async function getAuthStatus(): Promise<AuthStatus> {
  const response = await fetchWithRetry(ApiPaths.authStatus);
  if (!response.ok) {
    throw new Error("Failed to load auth status");
  }
  return response.json() as Promise<AuthStatus>;
}

export async function setupAccount(
  username: string,
  password: string,
): Promise<AuthUser> {
  const response = await fetchWithRetry(ApiPaths.authSetup, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  const payload = (await response.json()) as { user: AuthUser };
  return payload.user;
}

export async function loginAccount(
  username: string,
  password: string,
): Promise<AuthUser> {
  const response = await fetchWithRetry(ApiPaths.authLogin, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  if (!response.ok) {
    throw new Error(await readAPIError(response));
  }
  const payload = (await response.json()) as { user: AuthUser };
  return payload.user;
}

export async function logoutAccount(): Promise<void> {
  await fetchWithRetry(ApiPaths.authLogout, { method: "POST" });
}
