// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as Sentry from "@sentry/svelte";
import { StorageKeys } from "$lib/brand";
import {
  scrubTelemetryHeaders,
  scrubTelemetryQuery,
  scrubTelemetryUrl,
} from "./telemetry-scrub";

export type SentryRuntimeConfig = {
  dsn?: string;
  environment?: string;
  release?: string;
  tracesSampleRate?: number;
  clientReporting?: boolean;
};

let activeDSN = "";
let clientReportingEnabled = false;

// Content blockers reject the envelope POST and the browser logs each
// attempt as a failed request. Remembering an unreachable DSN lets later
// page loads skip init entirely, so the flood only happens once a day.
const SENTRY_BLOCKED_TTL_MS = 24 * 60 * 60 * 1000;

interface StoredSentryBlocked {
  at: number;
  dsn: string;
}

function sentryBlockedFor(dsn: string): boolean {
  try {
    const raw = localStorage.getItem(StorageKeys.sentryBlocked);
    if (!raw) return false;
    const parsed = JSON.parse(raw) as Partial<StoredSentryBlocked>;
    if (parsed.dsn !== dsn || typeof parsed.at !== "number") return false;
    return Date.now() - parsed.at < SENTRY_BLOCKED_TTL_MS;
  } catch {
    return false;
  }
}

function markSentryBlocked(dsn: string): void {
  try {
    const payload: StoredSentryBlocked = { at: Date.now(), dsn };
    localStorage.setItem(StorageKeys.sentryBlocked, JSON.stringify(payload));
  } catch {
    /* storage unavailable */
  }
}

export function resetSentryForTests(): void {
  activeDSN = "";
  clientReportingEnabled = false;
}

function scrubEvent<T extends { request?: unknown; user?: unknown }>(
  event: T,
): T {
  const request = event.request;
  if (request && typeof request === "object") {
    const req = request as {
      url?: unknown;
      query_string?: unknown;
      cookies?: unknown;
      headers?: unknown;
      data?: unknown;
    };
    if (typeof req.url === "string") {
      req.url = scrubTelemetryUrl(req.url);
    }
    if (typeof req.query_string === "string") {
      req.query_string = scrubTelemetryQuery(req.query_string);
    }
    const headers = scrubTelemetryHeaders(req.headers);
    if (headers) req.headers = headers;
    // Bodies and cookies can carry session material. Drop them entirely.
    delete req.cookies;
    delete req.data;
  }
  // Never send PII. Keep a stable anonymous id only.
  if (event.user && typeof event.user === "object") {
    const id = (event.user as { id?: unknown }).id;
    event.user = typeof id === "string" || typeof id === "number" ? { id } : {};
  }
  return event;
}

function scrubBreadcrumb(crumb: Sentry.Breadcrumb): Sentry.Breadcrumb | null {
  if (crumb.data && typeof crumb.data === "object") {
    const data = crumb.data as Record<string, unknown>;
    if (typeof data.url === "string") {
      data.url = scrubTelemetryUrl(data.url);
    }
  }
  return crumb;
}

// Content blockers and offline servers reject the envelope POST with a
// network error, which the browser logs as a failed request and the SDK
// would otherwise keep retrying. Once a send fails, report success so the
// SDK drains its queue and stop hitting the network for this session.
type TransportOptions = Parameters<typeof Sentry.makeFetchTransport>[0];
type Transport = ReturnType<typeof Sentry.makeFetchTransport>;
type SendResult = Awaited<ReturnType<Transport["send"]>>;

function blockedTolerantTransport(
  options: TransportOptions,
  dsn: string,
): Transport {
  const inner = Sentry.makeFetchTransport(options);
  let blocked = false;
  const dropped: SendResult = {};
  return {
    send(envelope) {
      if (blocked) return Promise.resolve(dropped);
      return Promise.resolve(inner.send(envelope)).catch(() => {
        blocked = true;
        if (dsn) markSentryBlocked(dsn);
        return dropped;
      });
    },
    flush(timeout) {
      return inner.flush(timeout);
    },
  };
}

function buildOptions(cfg: SentryRuntimeConfig): Sentry.BrowserOptions | null {
  const dsn = cfg.dsn?.trim() ?? "";
  if (!dsn || !cfg.clientReporting) return null;
  if (sentryBlockedFor(dsn)) return null;
  return {
    dsn,
    environment: cfg.environment?.trim() || undefined,
    release: cfg.release?.trim() || undefined,
    tracesSampleRate: cfg.tracesSampleRate ?? 0,
    sendDefaultPii: false,
    transport: (options) => blockedTolerantTransport(options, dsn),
    beforeSend: (event) => scrubEvent(event),
    beforeBreadcrumb: (crumb) => scrubBreadcrumb(crumb),
  };
}

function initOptions(opts: Sentry.BrowserOptions, reporting: boolean): void {
  const dsn = opts.dsn ?? "";
  clientReportingEnabled = reporting && dsn !== "";
  if (!clientReportingEnabled) {
    activeDSN = "";
    return;
  }
  if (activeDSN === dsn) return;
  Sentry.init(opts);
  activeDSN = dsn;
}

export function initSentryFromBuildEnv(): void {
  const dsn = import.meta.env.VITE_SENTRY_DSN?.trim();
  if (!dsn) return;
  // Telemetry is opt-in. A build-embedded DSN still waits for an accepted
  // consent answer, cached locally by telemetry-consent.
  try {
    if (localStorage.getItem(StorageKeys.telemetryConsent) !== "accepted") {
      return;
    }
  } catch {
    return;
  }
  const tracesRaw = import.meta.env.VITE_SENTRY_TRACES_SAMPLE_RATE;
  const tracesSampleRate =
    tracesRaw === undefined || tracesRaw === "" ? 0 : Number(tracesRaw);
  const opts = buildOptions({
    dsn,
    environment: import.meta.env.VITE_SENTRY_ENVIRONMENT,
    release: import.meta.env.VITE_SENTRY_RELEASE,
    tracesSampleRate: Number.isFinite(tracesSampleRate) ? tracesSampleRate : 0,
    clientReporting: true,
  });
  if (!opts) return;
  initOptions(opts, true);
}

export function applyRuntimeSentryConfig(cfg?: SentryRuntimeConfig): void {
  const reporting = cfg?.clientReporting === true;
  const opts = buildOptions({ ...cfg, clientReporting: reporting });
  if (!opts) {
    disableClientSentry();
    return;
  }
  initOptions(opts, reporting);
}

export function disableClientSentry(): void {
  clientReportingEnabled = false;
  activeDSN = "";
  try {
    void Sentry.close(2000);
  } catch {
    // close is best-effort when the SDK was never initialized
  }
}

export function sentryEnabled(): boolean {
  return clientReportingEnabled && activeDSN !== "";
}

export function captureClientError(error: unknown, source?: string): void {
  if (!sentryEnabled()) return;
  Sentry.withScope((scope) => {
    if (source) {
      scope.setTag("source", source);
    }
    if (error instanceof Error) {
      Sentry.captureException(error);
      return;
    }
    if (typeof error === "string") {
      Sentry.captureMessage(error);
      return;
    }
    scope.setExtra("value", error);
    Sentry.captureMessage(String(error ?? "Unknown client error"));
  });
}

export function captureClientMessage(
  message: string,
  level: Sentry.SeverityLevel = "error",
  source?: string,
): void {
  if (!sentryEnabled()) return;
  Sentry.withScope((scope) => {
    if (source) {
      scope.setTag("source", source);
    }
    scope.setLevel(level);
    Sentry.captureMessage(message);
  });
}
