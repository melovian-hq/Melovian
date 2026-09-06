// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/** Normalize a Subsonic base URL for storage and ping. */
export function normalizeServerUrl(raw: string): string {
  let url = raw.trim();
  if (!url) return url;
  if (!/^[a-z][a-z0-9+.-]*:\/\//i.test(url)) {
    url = `https://${url}`;
  }
  return url.replace(/\/+$/, "");
}

/** Derive a short display name from a server URL hostname. */
export function displayNameFromServerUrl(raw: string): string {
  const normalized = normalizeServerUrl(raw);
  if (!normalized) return "";
  try {
    const host = new URL(normalized).hostname;
    if (!host) return "";
    return host.replace(/^www\./i, "");
  } catch {
    return "";
  }
}
