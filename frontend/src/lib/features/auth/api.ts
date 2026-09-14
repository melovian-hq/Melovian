// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { ApiPaths } from "$lib/core/http/api-paths";
import { fetchWithRetry } from "$lib/core/http/client";
import { readAPIError } from "$lib/core/http/errors";
import { parseJson } from "$lib/core/http/parse";
import { authStatusSchema, authUserResponseSchema } from "./schemas";

export interface AuthUser {
  id: string;
  username: string;
  hasPassword?: boolean;
}

export interface AuthStatus {
  enabled: boolean;
  authenticated: boolean;
  setupRequired: boolean;
  demoMode?: boolean;
  fakeCatalog?: boolean;
  oidcEnabled?: boolean;
  oidcLoginUrl?: string;
  oidcProviderName?: string;
  localLoginEnabled?: boolean;
  user?: AuthUser;
}

/** Label for the SSO button, "SSO" when the provider has no configured name. */
export function ssoProviderLabel(providerName?: string | null): string {
  const name = providerName?.trim();
  return name ? name : "SSO";
}

export async function getAuthStatus(): Promise<AuthStatus> {
  const response = await fetchWithRetry(ApiPaths.authStatus);
  if (!response.ok) {
    throw new Error("Failed to load auth status");
  }
  return parseJson(authStatusSchema, response, "auth status");
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
  const payload = await parseJson(
    authUserResponseSchema,
    response,
    "auth setup",
  );
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
  const payload = await parseJson(
    authUserResponseSchema,
    response,
    "auth login",
  );
  return payload.user;
}

export async function logoutAccount(): Promise<void> {
  await fetchWithRetry(ApiPaths.authLogout, { method: "POST" });
}
