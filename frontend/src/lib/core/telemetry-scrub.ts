// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// Client-side telemetry scrubbing. Mirrors internal/observability/scrub.go.
// Subsonic credentials travel in query params (u, t, s, p) and headers, so
// every URL and header that can reach an event payload gets filtered here.

const REDACTED = "[redacted]";

const SENSITIVE_QUERY_KEYS = new Set([
  "u",
  "p",
  "t",
  "s",
  "token",
  "access_token",
  "refresh_token",
  "id_token",
  "password",
  "passwd",
  "secret",
  "client_secret",
  "key",
  "apikey",
  "api_key",
  "auth",
  "authorization",
  "session",
  "sessionid",
  "sid",
  "jwt",
  "signature",
  "sig",
]);

const SENSITIVE_HEADERS = new Set([
  "authorization",
  "proxy-authorization",
  "cookie",
  "set-cookie",
  "x-api-key",
  "x-auth-token",
  "x-subsonic-auth",
]);

export function scrubTelemetryUrl(raw: unknown): string {
  if (typeof raw !== "string" || raw === "") return "";
  let url: URL;
  const relative = raw.startsWith("/");
  try {
    url = new URL(raw, "https://telemetry.local");
  } catch {
    return "";
  }
  url.username = "";
  url.password = "";
  url.hash = "";
  for (const key of [...url.searchParams.keys()]) {
    if (SENSITIVE_QUERY_KEYS.has(key.toLowerCase())) {
      url.searchParams.set(key, REDACTED);
    }
  }
  if (relative) return `${url.pathname}${url.search}`;
  return url.toString();
}

export function scrubTelemetryQuery(raw: unknown): string {
  if (typeof raw !== "string" || raw === "") return "";
  try {
    const params = new URLSearchParams(raw);
    for (const key of [...params.keys()]) {
      if (SENSITIVE_QUERY_KEYS.has(key.toLowerCase())) {
        params.set(key, REDACTED);
      }
    }
    return params.toString();
  } catch {
    return "";
  }
}

export function scrubTelemetryHeaders(
  headers: unknown,
): Record<string, string> | undefined {
  if (!headers || typeof headers !== "object") return undefined;
  const out: Record<string, string> = {};
  for (const [key, value] of Object.entries(headers)) {
    if (typeof value !== "string") continue;
    out[key] = SENSITIVE_HEADERS.has(key.toLowerCase()) ? REDACTED : value;
  }
  return out;
}
