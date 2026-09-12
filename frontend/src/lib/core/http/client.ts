// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { getActiveInstanceId } from "$lib/features/instances/context";
import { logger, newRequestId } from "$lib/core/logger";
import { getOrCreateDeviceId } from "$lib/music/device-id";
import { clientCompatHeaders } from "$lib/compat";
import { isRemoteClient, resolveApiUrl } from "$lib/config/remote-server";

const REQUEST_ID_HEADER = "X-Request-Id";

export const DEFAULT_RETRY_ATTEMPTS = 3;
export const RETRY_BASE_DELAY_MS = 200;

/** Browser/network failures that are expected and should not hit BugSink. */
export function isTransientNetworkError(err: unknown): boolean {
  if (!(err instanceof Error)) return false;
  if (err.name === "AbortError") return true;
  const msg = err.message.toLowerCase();
  return (
    err.name === "TypeError" ||
    msg.includes("failed to fetch") ||
    msg.includes("networkerror") ||
    msg.includes("network request failed") ||
    msg.includes("load failed") ||
    msg.includes("fetch failed")
  );
}

export async function fetchWithRetry(
  input: RequestInfo | URL,
  init?: RequestInit,
  attempts = DEFAULT_RETRY_ATTEMPTS,
): Promise<Response> {
  let lastError: unknown;
  let lastRequestId = "";
  const resolvedInput =
    typeof input === "string" ? resolveApiUrl(input) : input;
  const crossOrigin = isRemoteClient();
  for (let attempt = 0; attempt < attempts; attempt++) {
    const requestId = newRequestId();
    lastRequestId = requestId;
    try {
      const response = await fetch(resolvedInput, {
        credentials:
          init?.credentials ?? (crossOrigin ? "include" : "same-origin"),
        ...init,
        headers: {
          ...apiHeaders(undefined, requestId),
          ...(init?.headers ?? {}),
        },
      });
      const responseRequestId =
        response.headers?.get?.(REQUEST_ID_HEADER) ?? requestId;
      if (response.status === 426) {
        return response;
      }
      if (response.status >= 500 && attempt < attempts - 1) {
        await sleep(RETRY_BASE_DELAY_MS * (attempt + 1) * (attempt + 1));
        continue;
      }
      if (
        (response.status === 404 ||
          response.status === 502 ||
          response.status === 504) &&
        attempt < attempts - 1
      ) {
        await sleep(RETRY_BASE_DELAY_MS * (attempt + 1) * (attempt + 1));
        continue;
      }
      if (!response.ok && response.status >= 400) {
        // 4xx (auth challenges, not found) are expected in normal flows.
        // Only ship 5xx to the backend log stream.
        const level = response.status >= 500 ? "warn" : "debug";
        logger[level](
          "HTTP request returned error status",
          {
            url: String(resolvedInput),
            status: response.status,
            attempt: attempt + 1,
          },
          "http.client",
          responseRequestId,
        );
      }
      return response;
    } catch (err) {
      lastError = err;
      if (init?.signal?.aborted) {
        throw err instanceof Error ? err : new Error("Request aborted");
      }
      if (attempt < attempts - 1) {
        await sleep(RETRY_BASE_DELAY_MS * (attempt + 1) * (attempt + 1));
        continue;
      }
    }
  }
  const err =
    lastError instanceof Error ? lastError : new Error("Request failed");
  const context = {
    url: String(resolvedInput),
    attempts,
    err: err.message,
  };
  // Transient network blips (offline, brief server hiccup) are expected for
  // background polls. Keep them out of BugSink. logger.warn ships to Sentry.
  if (isTransientNetworkError(err)) {
    logger.info("HTTP request failed", context, "http.client", lastRequestId);
  } else {
    logger.warn("HTTP request failed", context, "http.client", lastRequestId);
  }
  throw err;
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export function apiHeaders(
  contentType?: string,
  requestId?: string,
): HeadersInit {
  const out: HeadersInit = {
    Accept: "application/json",
    ...clientCompatHeaders(),
  };
  if (contentType) {
    out["Content-Type"] = contentType;
  }
  const instanceId = getActiveInstanceId();
  if (instanceId) {
    out["X-Instance-Id"] = instanceId;
  }
  const deviceId = getOrCreateDeviceId();
  if (deviceId) {
    out["X-Device-Id"] = deviceId;
  }
  if (requestId) {
    out[REQUEST_ID_HEADER] = requestId;
  }
  return out;
}
