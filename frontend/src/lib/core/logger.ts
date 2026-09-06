// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { captureClientError, captureClientMessage } from "$lib/core/sentry";
import { APP_NAME } from "$lib/brand";

export type LogLevel = "debug" | "info" | "warn" | "error";

const levelRank: Record<LogLevel, number> = {
  debug: 0,
  info: 1,
  warn: 2,
  error: 3,
};

let minLevel: LogLevel = import.meta.env.DEV ? "debug" : "info";

export interface ClientLogPayload {
  level: LogLevel;
  message: string;
  source?: string;
  stack?: string;
  url?: string;
  time?: string;
  requestId?: string;
}

function shouldLog(level: LogLevel): boolean {
  return levelRank[level] >= levelRank[minLevel];
}

function formatContext(context?: Record<string, unknown>): string {
  if (!context || Object.keys(context).length === 0) return "";
  try {
    return " " + JSON.stringify(context);
  } catch {
    return "";
  }
}

export function newRequestId(): string {
  if (
    typeof crypto !== "undefined" &&
    typeof crypto.randomUUID === "function"
  ) {
    return crypto.randomUUID();
  }
  return `req-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

function shipToBackend(payload: ClientLogPayload): void {
  if (typeof fetch === "undefined") return;
  if (payload.level !== "warn" && payload.level !== "error") return;

  const body = JSON.stringify({
    ...payload,
    url: payload.url ?? (typeof location !== "undefined" ? location.href : ""),
    time: payload.time ?? new Date().toISOString(),
    requestId: payload.requestId,
  });

  void import("$lib/config/remote-server")
    .then(({ isRemoteClient, resolveApiUrl }) => {
      fetch(resolveApiUrl("/api/client-log"), {
        method: "POST",
        credentials: isRemoteClient() ? "include" : "same-origin",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
          ...(payload.requestId ? { "X-Request-Id": payload.requestId } : {}),
        },
        body,
        keepalive: true,
      }).catch(() => {});
    })
    .catch(() => {});
}

function log(
  level: LogLevel,
  message: string,
  context?: Record<string, unknown>,
  source?: string,
  stack?: string,
  requestId?: string,
): void {
  if (!shouldLog(level)) return;

  const line = `[${APP_NAME}] ${message}${formatContext(context)}`;
  switch (level) {
    case "debug":
      console.debug(line, context ?? "");
      break;
    case "info":
      console.info(line, context ?? "");
      break;
    case "warn":
      console.warn(line, context ?? "");
      break;
    case "error":
      console.error(line, context ?? "");
      break;
  }

  if (level === "warn" || level === "error") {
    if (level === "error") {
      const err = new Error(message);
      if (stack) {
        err.stack = stack;
      }
      captureClientError(err, source);
    } else {
      captureClientMessage(message, "warning", source);
    }
    shipToBackend({
      level,
      message,
      source,
      stack,
      url: typeof location !== "undefined" ? location.href : "",
      time: new Date().toISOString(),
      requestId,
    });
  }
}

export function setLogLevel(level: LogLevel): void {
  minLevel = level;
}

export const logger = {
  debug(
    message: string,
    context?: Record<string, unknown>,
    source?: string,
    requestId?: string,
  ) {
    log("debug", message, context, source, undefined, requestId);
  },
  info(
    message: string,
    context?: Record<string, unknown>,
    source?: string,
    requestId?: string,
  ) {
    log("info", message, context, source, undefined, requestId);
  },
  warn(
    message: string,
    context?: Record<string, unknown>,
    source?: string,
    requestId?: string,
  ) {
    log("warn", message, context, source, undefined, requestId);
  },
  error(
    message: string,
    error?: unknown,
    context?: Record<string, unknown>,
    source?: string,
    requestId?: string,
  ) {
    const stack =
      error instanceof Error
        ? error.stack
        : error !== undefined
          ? String(error)
          : undefined;
    const merged =
      error instanceof Error ? { ...context, err: error.message } : context;
    log("error", message, merged, source, stack, requestId);
  },
};

export function logClientError(
  error: unknown,
  source?: string,
  requestId?: string,
): ClientLogPayload {
  const time = new Date().toISOString();
  const url = typeof location !== "undefined" ? location.href : "";

  if (error instanceof Error) {
    const payload: ClientLogPayload = {
      level: "error",
      message: error.message || "Unknown error",
      stack: error.stack,
      source,
      url,
      time,
      requestId,
    };
    logger.error(payload.message, error, undefined, source, requestId);
    return payload;
  }

  if (typeof error === "string") {
    const payload: ClientLogPayload = {
      level: "error",
      message: error,
      source,
      url,
      time,
      requestId,
    };
    logger.error(error, undefined, undefined, source, requestId);
    return payload;
  }

  const payload: ClientLogPayload = {
    level: "error",
    message: "An unknown error occurred",
    source,
    url,
    time,
    requestId,
  };
  logger.error(payload.message, error, undefined, source, requestId);
  return payload;
}
