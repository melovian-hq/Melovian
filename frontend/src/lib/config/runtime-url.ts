// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/** Rewrite wildcard bind hosts so copy-paste URLs work in browsers. */
export function normalizePublicBaseUrl(url: string): string {
  const trimmed = url.trim().replace(/\/+$/, "");
  if (!trimmed) return "";
  try {
    const parsed = new URL(trimmed);
    if (
      parsed.hostname === "0.0.0.0" ||
      parsed.hostname === "::" ||
      parsed.hostname === "[::]"
    ) {
      parsed.hostname = "127.0.0.1";
    }
    return parsed.origin;
  } catch {
    return trimmed
      .replace(/:\/\/0\.0\.0\.0(?=[:/]|$)/, "://127.0.0.1")
      .replace(/:\/\/\[::\](?=[:/]|$)/, "://127.0.0.1");
  }
}
